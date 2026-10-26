package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

type fakeAgents []string

func (f fakeAgents) ListAgents() []string { return []string(f) }

func dial(t *testing.T, s *Server, query string) *websocket.Conn {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws" + query
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("握手失败: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readEnvelope(t *testing.T, conn *websocket.Conn) *protocol.Envelope {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("读取消息失败: %v", err)
	}
	env, body := protocol.DecodeEnvelope(raw)
	if body != nil {
		t.Fatalf("解析消息失败: %+v", body)
	}
	return env
}

func TestHandshakeAndPing(t *testing.T) {
	msgBus := bus.NewMessageBus(4)
	defer msgBus.Close()

	s := New(Options{
		Config: &config.WebSocketConfig{Enabled: true, AuthToken: "tk", SendBuffer: 8},
		Bus:    msgBus,
		Agents: fakeAgents{"codex"},
	})
	conn := dial(t, s, "?token=tk")

	hello := readEnvelope(t, conn)
	if hello.Kind != protocol.KindSystem || hello.Type != protocol.TypeHello {
		t.Fatalf("首帧应为 hello，实际 %s/%s", hello.Kind, hello.Type)
	}
	var hd protocol.HelloData
	if err := hello.DecodeData(&hd); err != nil {
		t.Fatalf("解析 hello 载荷失败: %v", err)
	}
	if hd.ClientID == "" || hd.Protocol != protocol.ProtocolVersion {
		t.Fatalf("hello 载荷不符合预期: %+v", hd)
	}
	if hd.UserID != DefaultUser {
		t.Fatalf("单用户模式下身份应为 %s，实际 %s", DefaultUser, hd.UserID)
	}
	if len(hd.Agents) != 1 || hd.Agents[0].Name != "codex" {
		t.Fatalf("hello 应携带 agent 列表: %+v", hd.Agents)
	}

	ping := protocol.NewEnvelope(protocol.KindClient, protocol.TypePing).
		WithData(&protocol.PingData{TS: 123})
	if err := conn.WriteJSON(ping); err != nil {
		t.Fatalf("发送 ping 失败: %v", err)
	}
	pong := readEnvelope(t, conn)
	if pong.Kind != protocol.KindSystem || pong.Type != protocol.TypePong || pong.ReplyTo != ping.ID {
		t.Fatalf("pong 不符合预期: %+v", pong)
	}
	var pd protocol.PongData
	if err := pong.DecodeData(&pd); err != nil {
		t.Fatalf("解析 pong 载荷失败: %v", err)
	}
	if pd.ClientTS != 123 || pd.ServerTS == 0 {
		t.Fatalf("pong 载荷不符合预期: %+v", pd)
	}
}

func TestAgentList(t *testing.T) {
	msgBus := bus.NewMessageBus(4)
	defer msgBus.Close()

	s := New(Options{
		Config: &config.WebSocketConfig{Enabled: true, SendBuffer: 8, DefaultAgent: "codex"},
		Bus:    msgBus,
		Agents: fakeAgents{"codex", "coder"},
	})
	conn := dial(t, s, "")
	_ = readEnvelope(t, conn) // hello

	req := protocol.NewEnvelope(protocol.KindClient, protocol.TypeAgentList)
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("发送 agent.list 失败: %v", err)
	}
	resp := readEnvelope(t, conn)
	if resp.Type != protocol.TypeAgentList || resp.ReplyTo != req.ID {
		t.Fatalf("agent.list 响应不符合预期: %+v", resp)
	}
	var data protocol.AgentListData
	if err := resp.DecodeData(&data); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if len(data.Agents) != 2 || data.DefaultAgent != "codex" {
		t.Fatalf("agent 列表不符合预期: %+v", data)
	}
}

func TestRejectsUnauthorized(t *testing.T) {
	s := New(Options{Config: &config.WebSocketConfig{Enabled: true, AuthToken: "tk"}})
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("未携带 token 应握手失败")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("期望 401，实际 %+v", resp)
	}
}

func TestMultiUserToken(t *testing.T) {
	msgBus := bus.NewMessageBus(4)
	defer msgBus.Close()

	s := New(Options{
		Config: &config.WebSocketConfig{
			Enabled:    true,
			SendBuffer: 8,
			Users:      map[string]string{"tok-a": "alice", "tok-b": "bob"},
		},
		Bus: msgBus,
	})

	helloOf := func(token string) protocol.HelloData {
		conn := dial(t, s, "?token="+token)
		env := readEnvelope(t, conn)
		var hd protocol.HelloData
		if err := env.DecodeData(&hd); err != nil {
			t.Fatalf("解析 hello 失败: %v", err)
		}
		return hd
	}

	if got := helloOf("tok-a").UserID; got != "alice" {
		t.Fatalf("tok-a 应识别为 alice，实际 %s", got)
	}
	if got := helloOf("tok-b").UserID; got != "bob" {
		t.Fatalf("tok-b 应识别为 bob，实际 %s", got)
	}

	// 未登记的 token 必须被拒绝
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?token=unknown"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("未知 token 应握手失败")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("期望 401，实际 %+v", resp)
	}
}

func TestStartRejectsInvalidUserID(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	defer msgBus.Close()

	s := New(Options{
		Config: &config.WebSocketConfig{
			Enabled: true,
			Users:   map[string]string{"tok": "bad_user"}, // 含下划线会破坏会话键前缀判断
		},
		Bus: msgBus,
	})
	if err := s.Start(t.Context()); err == nil {
		t.Fatal("非法 userID 应导致启动失败")
	}
}

func TestRejectsTooManyClients(t *testing.T) {
	msgBus := bus.NewMessageBus(4)
	defer msgBus.Close()

	s := New(Options{
		Config: &config.WebSocketConfig{Enabled: true, SendBuffer: 8, MaxClients: 1},
		Bus:    msgBus,
	})
	_ = dial(t, s, "")
	time.Sleep(50 * time.Millisecond)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("超过连接上限应握手失败")
	}
	if resp == nil || resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("期望 503，实际 %+v", resp)
	}
}
