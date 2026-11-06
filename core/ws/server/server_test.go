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
	"github.com/kinwyb/buibuiCodex/mcp/edp"
)

type fakeAgents []string

func (f fakeAgents) ListAgents() []string { return []string(f) }

func tokenAuthenticator(token string) (string, string, bool) {
	if !validUserID(token) {
		return "", "", false
	}
	return token, "", true
}

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
		Config:        &config.WebSocketConfig{Enabled: true, SendBuffer: 8},
		Bus:           msgBus,
		Agents:        fakeAgents{"codex"},
		Authenticator: tokenAuthenticator,
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
	if hd.UserID != "tk" {
		t.Fatalf("应使用自定义鉴权返回的 userID，实际 %s", hd.UserID)
	}
	if hd.ChatID == "" {
		t.Fatal("hello 应下发当前对话标识")
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
		Config:        &config.WebSocketConfig{Enabled: true, SendBuffer: 8, DefaultAgent: "codex"},
		Bus:           msgBus,
		Agents:        fakeAgents{"codex", "coder"},
		Authenticator: tokenAuthenticator,
	})
	conn := dial(t, s, "?token=tester")
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
	s := New(Options{Config: &config.WebSocketConfig{Enabled: true}})
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

func TestDefaultAuthenticatorUsesEDPUserInfo(t *testing.T) {
	userInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/login/userInfo" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		switch r.Header.Get("token") {
		case "token-1":
			_, _ = w.Write([]byte(`{"code":0,"data":{"EmpID":"007928","EmpName":"张三","RequestSucc":true,"Token":"token-1","UserID":"u-1"}}`))
		case "bad-empid":
			_, _ = w.Write([]byte(`{"code":0,"data":{"EmpID":"bad_user","RequestSucc":true,"Token":"bad-empid","UserID":"u-2"}}`))
		default:
			_, _ = w.Write([]byte(`{"code":10001,"errmsg":"用户未登入"}`))
		}
	}))
	defer userInfoServer.Close()
	edp.NewAPIClient(userInfoServer.URL)

	msgBus := bus.NewMessageBus(4)
	defer msgBus.Close()

	s := New(Options{
		Config: &config.WebSocketConfig{Enabled: true, SendBuffer: 8},
		Bus:    msgBus,
	})

	conn := dial(t, s, "?token=token-1")
	env := readEnvelope(t, conn)
	var hd protocol.HelloData
	if err := env.DecodeData(&hd); err != nil {
		t.Fatalf("解析 hello 失败: %v", err)
	}
	if hd.UserID != "007928" {
		t.Fatalf("应将 EmpID 作为 userID，实际 %s", hd.UserID)
	}
	if hd.ChatID == "" {
		t.Fatal("hello 应下发服务端生成的 chatID")
	}

	if _, _, ok := DefaultAuthenticator("bad-empid"); ok {
		t.Fatal("EmpID 不合法时应拒绝")
	}
	if _, _, ok := DefaultAuthenticator("invalid-token"); ok {
		t.Fatal("用户信息查询失败时应拒绝")
	}
}

func TestCustomAuthenticator(t *testing.T) {
	msgBus := bus.NewMessageBus(4)
	defer msgBus.Close()

	s := New(Options{
		Config: &config.WebSocketConfig{Enabled: true, SendBuffer: 8},
		Bus:    msgBus,
		// 模拟接入真实鉴权：token 校验后映射为业务用户 ID，并绑定固定对话
		Authenticator: func(token string) (string, string, bool) {
			if token == "jwt-alice" {
				return "u1001", "chat-01", true
			}
			return "", "", false
		},
	})
	conn := dial(t, s, "?token=jwt-alice")
	env := readEnvelope(t, conn)
	var hd protocol.HelloData
	if err := env.DecodeData(&hd); err != nil {
		t.Fatalf("解析 hello 失败: %v", err)
	}
	if hd.UserID != "u1001" {
		t.Fatalf("应使用自定义身份解析结果，实际 %s", hd.UserID)
	}
	if hd.ChatID != "chat-01" {
		t.Fatalf("应使用自定义对话标识，实际 %s", hd.ChatID)
	}

	// 未通过自定义校验的 token 必须被拒绝
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?token=other"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("自定义校验失败时应握手失败")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("期望 401，实际 %+v", resp)
	}
}

func TestRejectsTooManyClients(t *testing.T) {
	msgBus := bus.NewMessageBus(4)
	defer msgBus.Close()

	s := New(Options{
		Config:        &config.WebSocketConfig{Enabled: true, SendBuffer: 8, MaxClients: 1},
		Bus:           msgBus,
		Authenticator: tokenAuthenticator,
	})
	_ = dial(t, s, "?token=tester")
	time.Sleep(50 * time.Millisecond)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?token=tester"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("超过连接上限应握手失败")
	}
	if resp == nil || resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("期望 503，实际 %+v", resp)
	}
}
