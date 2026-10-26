package command

import (
	"context"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

type fakeAgents []string

func (f fakeAgents) ListAgents() []string { return []string(f) }

func newTestRegistry(msgBus *bus.MessageBus, agents AgentSource) (*Registry, *conn.Hub) {
	hub := conn.NewHub()
	return NewRegistry(Deps{
		Bus:          msgBus,
		Hub:          hub,
		Agents:       agents,
		Channel:      "ws",
		AccountID:    "web",
		DefaultAgent: "codex",
	}), hub
}

func newTestClient(hub *conn.Hub) *conn.Client {
	c := conn.NewClient(conn.Config{SendBuffer: 8}, nil, hub, nil)
	hub.Add(c)
	return c
}

func TestChatSendPublishesInput(t *testing.T) {
	msgBus := bus.NewMessageBus(10)
	defer msgBus.Close()

	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅入站消息失败: %v", err)
	}
	defer sub.Close()

	reg, hub := newTestRegistry(msgBus, fakeAgents{"codex"})
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{ChatID: "c1", Content: "你好"})
	reg.Dispatch(context.Background(), c, env)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	in, ok := sub.Consume(ctx)
	if !ok || in == nil {
		t.Fatal("未收到入站消息")
	}
	if in.Channel != "ws" || in.AccountID != "web" || in.ChatID != "c1" || in.Content != "你好" {
		t.Fatalf("入站消息不符合预期: %+v", in)
	}
	if in.Session != "ws_web_c1" {
		t.Fatalf("会话标识不符合预期: %s", in.Session)
	}
	if in.ID == "" {
		t.Fatal("应生成请求 ID")
	}

	select {
	case raw := <-c.Outbound():
		ack, body := protocol.DecodeEnvelope(raw)
		if body != nil {
			t.Fatalf("解析 ack 失败: %+v", body)
		}
		if ack.Kind != protocol.KindSystem || ack.Type != protocol.TypeAck {
			t.Fatalf("应答类型不符合预期: %s/%s", ack.Kind, ack.Type)
		}
		if ack.ReplyTo != env.ID {
			t.Fatalf("replyTo 应回填上行 ID，实际 %s", ack.ReplyTo)
		}
		var data protocol.AckData
		if err := ack.DecodeData(&data); err != nil {
			t.Fatalf("解析 ack 载荷失败: %v", err)
		}
		if !data.OK || data.SessionID != "ws_web_c1" || data.ReqID != in.ID {
			t.Fatalf("ack 载荷不符合预期: %+v", data)
		}
	case <-time.After(time.Second):
		t.Fatal("未收到 ack")
	}
}

func TestChatSendRejectsEmptyContent(t *testing.T) {
	msgBus := bus.NewMessageBus(2)
	defer msgBus.Close()

	reg, hub := newTestRegistry(msgBus, nil)
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{ChatID: "c1"})
	reg.Dispatch(context.Background(), c, env)

	select {
	case raw := <-c.Outbound():
		e, body := protocol.DecodeEnvelope(raw)
		if body != nil {
			t.Fatalf("解析失败: %+v", body)
		}
		if e.Type != protocol.TypeError {
			t.Fatalf("应返回 error，实际 %s", e.Type)
		}
	default:
		t.Fatal("未返回错误消息")
	}
}

func TestChatSendRejectsUnknownAgent(t *testing.T) {
	msgBus := bus.NewMessageBus(2)
	defer msgBus.Close()

	reg, hub := newTestRegistry(msgBus, fakeAgents{"codex"})
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{ChatID: "c1", Content: "hi", Agent: "missing"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	e, _ := protocol.DecodeEnvelope(raw)
	var data protocol.ErrorBody
	_ = e.DecodeData(&data)
	if data.Code != protocol.CodeAgentNotFound {
		t.Fatalf("错误码应为 agent_not_found，实际 %s", data.Code)
	}
}

func TestDispatchRejectsNonClientKind(t *testing.T) {
	msgBus := bus.NewMessageBus(2)
	defer msgBus.Close()

	reg, hub := newTestRegistry(msgBus, nil)
	c := newTestClient(hub)

	reg.Dispatch(context.Background(), c, protocol.NewEnvelope(protocol.KindAgent, "message_delta"))

	raw := <-c.Outbound()
	e, _ := protocol.DecodeEnvelope(raw)
	var data protocol.ErrorBody
	_ = e.DecodeData(&data)
	if data.Code != protocol.CodeUnsupported {
		t.Fatalf("错误码应为 unsupported，实际 %s", data.Code)
	}
}

func TestDispatchRejectsUnknownType(t *testing.T) {
	msgBus := bus.NewMessageBus(2)
	defer msgBus.Close()

	reg, hub := newTestRegistry(msgBus, nil)
	c := newTestClient(hub)

	reg.Dispatch(context.Background(), c, protocol.NewEnvelope(protocol.KindClient, "chat.unknown"))

	raw := <-c.Outbound()
	e, _ := protocol.DecodeEnvelope(raw)
	if e.Type != protocol.TypeError {
		t.Fatalf("未知命令应返回 error，实际 %s", e.Type)
	}
}

func TestSessionKeyMatchesInputMessage(t *testing.T) {
	reg, _ := newTestRegistry(nil, nil)
	in := &types.InputMessage{Channel: "ws", AccountID: "web", ChatID: "c1"}
	if got := reg.SessionKey("c1"); got != in.SessionID() {
		t.Fatalf("会话键不一致: %s != %s", got, in.SessionID())
	}
}

func TestAgentInfos(t *testing.T) {
	infos := AgentInfos(fakeAgents{"a", "b"})
	if len(infos) != 2 || infos[0].Name != "a" || infos[1].Name != "b" {
		t.Fatalf("agent 列表不符合预期: %+v", infos)
	}
	if AgentInfos(nil) != nil {
		t.Fatal("空来源应返回 nil")
	}
}
