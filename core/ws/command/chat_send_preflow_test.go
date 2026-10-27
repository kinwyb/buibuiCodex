// 本文件验证 chat.send 与界面前置路由的衔接：消息经会话队列投递，Note 有内容才改写内容，
// 其余情况原样投递。

package command

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/preflow"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// recordingPreflow 会被队列 worker 调用，读写都要加锁
type recordingPreflow struct {
	mu   sync.Mutex
	reqs []*preflow.Request
	res  *preflow.Result
}

func (f *recordingPreflow) Route(_ context.Context, req *preflow.Request) *preflow.Result {
	f.mu.Lock()
	f.reqs = append(f.reqs, req)
	res := f.res
	f.mu.Unlock()
	return res
}

func (f *recordingPreflow) calls() []*preflow.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*preflow.Request(nil), f.reqs...)
}

func newPreflowRegistry(msgBus *bus.MessageBus, pf Preflow) (*Registry, *conn.Hub) {
	hub := conn.NewHub()
	return NewRegistry(Deps{
		Bus:          msgBus,
		Hub:          hub,
		Agents:       fakeAgents{"codex"},
		Channel:      "ws",
		DefaultAgent: "codex",
		Preflow:      pf,
	}), hub
}

func consumeInput(t *testing.T, sub *bus.InboundSubscription) *types.InputMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	in, ok := sub.Consume(ctx)
	if !ok || in == nil {
		t.Fatal("未收到入站消息")
	}
	return in
}

func TestChatSendPreflowAnnotatesContent(t *testing.T) {
	msgBus := bus.NewMessageBus(10)
	defer msgBus.Close()
	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅入站消息失败: %v", err)
	}
	defer sub.Close()

	note := "[界面已就绪] 已打开页面「生产工单」(view=OrderList)。请勿重复 ui_navigate。"
	pf := &recordingPreflow{res: &preflow.Result{
		Steps: []preflow.Step{{Kind: "navigate", Target: "OrderList", Label: "生产工单"}},
		Note:  note,
	}}
	reg, hub := newPreflowRegistry(msgBus, pf)
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{ChatID: "c1", Content: "帮我建个工单"})
	reg.Dispatch(context.Background(), c, env)

	in := consumeInput(t, sub)
	if in.Content != "帮我建个工单\n"+note {
		t.Fatalf("消息应带上界面说明，实际 %q", in.Content)
	}
	calls := pf.calls()
	if len(calls) != 1 {
		t.Fatalf("前置路由应被调用一次，实际 %d", len(calls))
	}
	if calls[0].SessionID != "ws_web_c1" || calls[0].Content != "帮我建个工单" {
		t.Errorf("传给前置路由的入参 = %+v", calls[0])
	}
	// 日志关联用的 ReqID 必须与最终投递的消息 ID 一致
	if calls[0].ReqID != in.ID {
		t.Errorf("ReqID = %s, 消息 ID = %s", calls[0].ReqID, in.ID)
	}

	// ack 照常返回，reqID 与消息一致
	select {
	case raw := <-c.Outbound():
		e, body := protocol.DecodeEnvelope(raw)
		if body != nil {
			t.Fatalf("解析 ack 失败: %+v", body)
		}
		var data protocol.AckData
		if err := e.DecodeData(&data); err != nil {
			t.Fatalf("解析 ack 载荷失败: %v", err)
		}
		if data.ReqID != in.ID || !data.OK {
			t.Errorf("ack 载荷 = %+v", data)
		}
	case <-time.After(time.Second):
		t.Fatal("未收到 ack")
	}
}

func TestChatSendPreflowKeepsMessageUntouched(t *testing.T) {
	cases := map[string]*preflow.Result{
		"没有动作":  {Reason: "无需切换页面"},
		"只有待办":  {Pending: []preflow.Step{{Kind: "act", Target: "detail"}}, Note: ""},
		"实现返回空": nil,
	}
	for name, res := range cases {
		msgBus := bus.NewMessageBus(4)
		sub, err := msgBus.SubscribeInput()
		if err != nil {
			t.Fatalf("订阅失败: %v", err)
		}
		pf := &recordingPreflow{res: res}
		reg, hub := newPreflowRegistry(msgBus, pf)
		c := newTestClient(hub)
		env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
			WithData(&protocol.ChatSendData{ChatID: "c1", Content: "你好"})
		reg.Dispatch(context.Background(), c, env)

		if in := consumeInput(t, sub); in.Content != "你好" {
			t.Errorf("%s: 消息不该被改写，实际 %q", name, in.Content)
		}
		sub.Close()
		msgBus.Close()
	}
}

// 未配置决策模型时（Deps.Preflow 为空）走的是原来的路径
func TestChatSendWithoutPreflow(t *testing.T) {
	msgBus := bus.NewMessageBus(4)
	defer msgBus.Close()
	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}
	defer sub.Close()

	reg, hub := newPreflowRegistry(msgBus, nil)
	c := newTestClient(hub)
	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{ChatID: "c1", Content: "你好"})
	reg.Dispatch(context.Background(), c, env)

	if in := consumeInput(t, sub); in.Content != "你好" {
		t.Errorf("消息应原样投递，实际 %q", in.Content)
	}
}
