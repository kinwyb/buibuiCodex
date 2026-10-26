// 本文件测试界面操作的反向 RPC：服务端发起请求、浏览器回执认领、无宿主与会话校验。

package server

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// uiTestServer 起一个身份固定的服务：token 只用于握手，身份由 Authenticator 决定，
// 这样测试不依赖 DefaultAuthenticator 的本地调试实现。
func uiTestServer(t *testing.T, timeout time.Duration) (*Server, *websocket.Conn) {
	t.Helper()
	msgBus := bus.NewMessageBus(8)
	t.Cleanup(func() { msgBus.Close() })

	s := New(Options{
		Config:           &config.WebSocketConfig{Enabled: true, SendBuffer: 8},
		Bus:              msgBus,
		Agents:           fakeAgents{"codex"},
		Channel:          "ws",
		UIRequestTimeout: timeout,
		Authenticator: func(string) (string, string, bool) {
			return "u1", "c1", true
		},
	})
	conn := dial(t, s, "?token=tk")
	_ = readEnvelope(t, conn) // hello
	return s, conn
}

// subscribeSession 让连接订阅自己的会话（发一条 chat.send 并读到 ack）。
// 界面请求是按会话下发的，订阅是能收到请求的前提。
func subscribeSession(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	req := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{Content: "你好"})
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("发送 chat.send 失败: %v", err)
	}
	ack := readEnvelope(t, conn)
	var data protocol.AckData
	if err := ack.DecodeData(&data); err != nil {
		t.Fatalf("解析 ack 失败: %v", err)
	}
	if data.SessionID != "ws_u1_c1" {
		t.Fatalf("会话标识不符合预期: %s", data.SessionID)
	}
	return data.SessionID
}

func TestUIRequestRoundTrip(t *testing.T) {
	s, conn := uiTestServer(t, 2*time.Second)
	sessionID := subscribeSession(t, conn)

	type result struct {
		resp *protocol.UIResponseData
		err  error
	}
	done := make(chan result, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		resp, err := s.Request(ctx, sessionID, &protocol.UIRequestData{Op: "observe"})
		done <- result{resp: resp, err: err}
	}()

	reqEnv := readEnvelope(t, conn)
	if reqEnv.Kind != protocol.KindUI || reqEnv.Type != protocol.TypeUIRequest {
		t.Fatalf("应下发 ui.request，实际 %s/%s", reqEnv.Kind, reqEnv.Type)
	}
	if reqEnv.SessionID != sessionID {
		t.Fatalf("请求应携带会话标识，实际 %q", reqEnv.SessionID)
	}
	var reqData protocol.UIRequestData
	if err := reqEnv.DecodeData(&reqData); err != nil {
		t.Fatalf("解析 ui.request 失败: %v", err)
	}
	if reqData.Op != "observe" {
		t.Fatalf("op 不符合预期: %s", reqData.Op)
	}

	snapshot := `{"view":"V0189_V0279_V0280","rev":7}`
	reply := protocol.NewEnvelope(protocol.KindClient, protocol.TypeUIResponse)
	reply.ReplyTo = reqEnv.ID
	reply.SessionID = reqEnv.SessionID
	reply.WithData(&protocol.UIResponseData{OK: true, Rev: 7, Snapshot: json.RawMessage(snapshot)})
	if err := conn.WriteJSON(reply); err != nil {
		t.Fatalf("发送 ui.response 失败: %v", err)
	}

	got := <-done
	if got.err != nil {
		t.Fatalf("请求应成功，实际失败: %v", got.err)
	}
	if !got.resp.OK || got.resp.Rev != 7 {
		t.Fatalf("回执不符合预期: %+v", got.resp)
	}
	if string(got.resp.Snapshot) != snapshot {
		t.Fatalf("快照应原样透传，实际 %s", got.resp.Snapshot)
	}
}

func TestUIRequestWithoutHost(t *testing.T) {
	s, _ := uiTestServer(t, time.Second)

	// 没有页面订阅这个会话，应立即失败而不是等到超时
	_, err := s.Request(context.Background(), "ws_u1_c1", &protocol.UIRequestData{Op: "observe"})
	if !errors.Is(err, ErrNoUIHost) {
		t.Fatalf("未订阅的会话应返回 ErrNoUIHost，实际 %v", err)
	}
}

func TestUIResponseWithWrongSessionIsRejected(t *testing.T) {
	s, conn := uiTestServer(t, 300*time.Millisecond)
	sessionID := subscribeSession(t, conn)

	done := make(chan error, 1)
	go func() {
		_, err := s.Request(context.Background(), sessionID, &protocol.UIRequestData{Op: "observe"})
		done <- err
	}()

	reqEnv := readEnvelope(t, conn)
	reply := protocol.NewEnvelope(protocol.KindClient, protocol.TypeUIResponse)
	reply.ReplyTo = reqEnv.ID
	reply.SessionID = "ws_u1_other" // 会话对不上，必须被拒绝
	reply.WithData(&protocol.UIResponseData{OK: true})
	if err := conn.WriteJSON(reply); err != nil {
		t.Fatalf("发送 ui.response 失败: %v", err)
	}

	errEnv := readEnvelope(t, conn)
	if errEnv.Kind != protocol.KindSystem || errEnv.Type != protocol.TypeError {
		t.Fatalf("应回复错误帧，实际 %s/%s", errEnv.Kind, errEnv.Type)
	}
	var body protocol.ErrorBody
	if err := errEnv.DecodeData(&body); err != nil {
		t.Fatalf("解析错误载荷失败: %v", err)
	}
	if body.Code != protocol.CodeForbidden {
		t.Fatalf("错误码应为 forbidden，实际 %s", body.Code)
	}
	if err := <-done; err == nil {
		t.Fatal("会话不一致的回执不应被认领")
	}
}
