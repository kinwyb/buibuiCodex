// 本文件验证界面前置路由的真实装配：浏览器只按既有的 ui.request 协议收发消息，
// 页面与入口由决策模型在消息进 agent 之前替用户打开。

package server

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
	"github.com/kinwyb/buibuiCodex/decision"
)

// scriptedDecider 按问题 id 给答案：page 选页面，entry 选入口。
// 队列 worker 在另一个 goroutine 里调它，记录要加锁。
type scriptedDecider struct {
	mu      sync.Mutex
	answers map[string]choice
	touched []string
}

func (d *scriptedDecider) kinds() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.touched...)
}

type choice struct {
	answer     string
	confidence float64
}

func (d *scriptedDecider) AskRequest(_ context.Context, req *decision.Request) (*decision.Response, error) {
	answers := map[string]decision.Answer{}
	for id := range req.Questions {
		d.mu.Lock()
		d.touched = append(d.touched, id)
		d.mu.Unlock()
		a := d.answers[id]
		answers[id] = decision.Answer{Type: decision.TypeChoice, Choice: a.answer, Confidence: a.confidence}
	}
	return &decision.Response{Answers: answers}, nil
}

func snap(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("构造快照失败: %v", err)
	}
	return raw
}

// nextUIRequest 读一条界面请求
func nextUIRequest(t *testing.T, conn *websocket.Conn) (*protocol.UIRequestData, *protocol.Envelope) {
	t.Helper()
	env := readEnvelope(t, conn)
	if env.Kind != protocol.KindUI || env.Type != protocol.TypeUIRequest {
		t.Fatalf("应收到 ui.request，实际 %s/%s", env.Kind, env.Type)
	}
	var req protocol.UIRequestData
	if err := env.DecodeData(&req); err != nil {
		t.Fatalf("解析界面请求失败: %v", err)
	}
	return &req, env
}

// replyUI 扮演浏览器回执
func replyUI(t *testing.T, conn *websocket.Conn, reqEnv *protocol.Envelope, data *protocol.UIResponseData) {
	t.Helper()
	reply := protocol.NewEnvelope(protocol.KindClient, protocol.TypeUIResponse)
	reply.ReplyTo = reqEnv.ID
	reply.SessionID = reqEnv.SessionID // 回执必须把会话原样带回
	reply.WithData(data)
	if err := conn.WriteJSON(reply); err != nil {
		t.Fatalf("回执失败: %v", err)
	}
}

func consumeInput(t *testing.T, sub *bus.InboundSubscription) *types.InputMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	in, ok := sub.Consume(ctx)
	if !ok || in == nil {
		t.Fatal("未收到入站消息")
	}
	return in
}

func TestPreflowOpensPageAndEntryBeforeAgent(t *testing.T) {
	yes := true
	msgBus := bus.NewMessageBus(16)
	defer msgBus.Close()
	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅入站消息失败: %v", err)
	}
	defer sub.Close()

	dec := &scriptedDecider{answers: map[string]choice{
		"page":  {"生产工单", 0.94},
		"entry": {"新建工单", 0.86},
	}}
	s := New(Options{
		Config: &config.WebSocketConfig{
			Enabled:    true,
			SendBuffer: 16,
			PageRouter: &config.PageRouterConfig{
				Enabled: true, Timeout: 5, DeepAction: &yes, Annotate: &yes,
			},
		},
		Bus:              msgBus,
		Agents:           fakeAgents{"codex"},
		Channel:          "ws",
		UIRequestTimeout: 2 * time.Second,
		PreflowDecider:   dec,
		Authenticator:    func(string) (string, string, bool) { return "u1", "c1", true },
	})

	conn := dial(t, s, "?token=tk")
	readEnvelope(t, conn) // hello

	req := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{Content: "帮我建个工单，二号线停机了"})
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("发送 chat.send 失败: %v", err)
	}

	// 前置路由要等浏览器回执，不能占着读循环：所以先收到 ack，界面请求随后下发
	ack := readEnvelope(t, conn)
	if ack.Type != protocol.TypeAck || ack.ReplyTo != req.ID {
		t.Fatalf("应先收到 ack，实际 %s replyTo=%s", ack.Type, ack.ReplyTo)
	}
	var ackData protocol.AckData
	if err := ack.DecodeData(&ackData); err != nil || ackData.SessionID != "ws_u1_c1" {
		t.Fatalf("ack 载荷异常: %v %+v", err, ackData)
	}

	// 1) observe：浏览器回当前页面（带菜单）
	ui, env := nextUIRequest(t, conn)
	if ui.Op != "observe" {
		t.Fatalf("第一步应取快照，实际 op=%s", ui.Op)
	}
	replyUI(t, conn, env, &protocol.UIResponseData{OK: true, Rev: 1, Snapshot: snap(t, map[string]any{
		"view": "Home", "title": "工作台",
		"navigation": []any{map[string]any{"view": "OrderList", "title": "生产工单"}},
	})})

	// 2) navigate：决策模型选中的页面，用菜单里的 view
	ui, env = nextUIRequest(t, conn)
	if ui.Op != "navigate" || ui.View != "OrderList" {
		t.Fatalf("应打开 OrderList，实际 %+v", ui)
	}
	replyUI(t, conn, env, &protocol.UIResponseData{OK: true, Rev: 7, Snapshot: snap(t, map[string]any{
		"view": "OrderList", "title": "生产工单",
		"actions": []any{map[string]any{"id": "create", "label": "新建工单"}},
	})})

	// 3) act：在新页面点上入口，带的 rev 必须是新页面的
	ui, env = nextUIRequest(t, conn)
	if ui.Op != "act" || ui.Cmd != "create" || ui.Rev != 7 {
		t.Fatalf("应在新页面执行 create(rev=7)，实际 %+v", ui)
	}
	replyUI(t, conn, env, &protocol.UIResponseData{OK: true, Rev: 8, Snapshot: snap(t, map[string]any{
		"view": "OrderList", "title": "生产工单",
	})})

	// 消息进入总线时界面已经就位，说明文字随消息交给 agent
	in := consumeInput(t, sub)
	if in.Session != "ws_u1_c1" {
		t.Fatalf("会话键 = %s", in.Session)
	}
	if in.ID != ackData.ReqID {
		t.Errorf("ack 的 reqID 应就是这条消息的 ID，实际 %s / %s", ackData.ReqID, in.ID)
	}
	for _, want := range []string{"帮我建个工单，二号线停机了", "[界面已就绪]", "已打开页面「生产工单」", "已点开「新建工单」", "rev=8", "请勿重复 ui_navigate"} {
		if !strings.Contains(in.Content, want) {
			t.Errorf("消息应包含 %q，实际 %q", want, in.Content)
		}
	}

	if kinds := dec.kinds(); len(kinds) != 2 {
		t.Errorf("应做过两次判断（page + entry），实际 %v", kinds)
	}
}

// 模型选空白项时消息必须原样进 agent
func TestPreflowFallsThroughOnBlankChoice(t *testing.T) {
	yes := true
	msgBus := bus.NewMessageBus(8)
	defer msgBus.Close()
	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}
	defer sub.Close()

	dec := &scriptedDecider{answers: map[string]choice{"page": {"current", 0.2}}}
	s := New(Options{
		Config: &config.WebSocketConfig{
			Enabled: true, SendBuffer: 8,
			PageRouter: &config.PageRouterConfig{Enabled: true, Timeout: 5, Annotate: &yes},
		},
		Bus: msgBus, Agents: fakeAgents{"codex"}, Channel: "ws",
		UIRequestTimeout: 2 * time.Second, PreflowDecider: dec,
		Authenticator: func(string) (string, string, bool) { return "u1", "c1", true },
	})
	conn := dial(t, s, "?token=tk")
	readEnvelope(t, conn)

	if err := conn.WriteJSON(protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{Content: "建个工单"})); err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	readEnvelope(t, conn) // ack 先到

	ui, env := nextUIRequest(t, conn)
	if ui.Op != "observe" {
		t.Fatalf("应只有 observe，实际 op=%s", ui.Op)
	}
	replyUI(t, conn, env, &protocol.UIResponseData{OK: true, Rev: 1, Snapshot: snap(t, map[string]any{
		"view": "Home", "navigation": []any{map[string]any{"view": "OrderList", "title": "生产工单"}},
	})})

	in := consumeInput(t, sub)
	if in.Content != "建个工单" {
		t.Errorf("选了空白项时不该改写消息，实际 %q", in.Content)
	}
}

// 没配 page_router 时不应发起任何界面请求
func TestPreflowDisabledByDefault(t *testing.T) {
	msgBus := bus.NewMessageBus(8)
	defer msgBus.Close()
	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}
	defer sub.Close()

	s := New(Options{
		Config: &config.WebSocketConfig{Enabled: true, SendBuffer: 8},
		Bus:    msgBus, Agents: fakeAgents{"codex"}, Channel: "ws",
		Authenticator: func(string) (string, string, bool) { return "u1", "c1", true },
	})
	conn := dial(t, s, "?token=tk")
	readEnvelope(t, conn)

	if err := conn.WriteJSON(protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{Content: "建个工单"})); err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	in := consumeInput(t, sub)
	if in.Content != "建个工单" {
		t.Errorf("消息应原样投递，实际 %q", in.Content)
	}
	// ack 紧跟其后，中间没有界面请求
	ack := readEnvelope(t, conn)
	if ack.Type != protocol.TypeAck {
		t.Errorf("应直接收到 ack，实际 %s", ack.Type)
	}
}
