// 本文件测试 chat.send。

package command

import (
	"context"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

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
		if !data.OK || data.SessionID != "ws_web_c1" || data.ChatID != "c1" || data.ReqID != in.ID {
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

func TestChatSendPassesValidatedSkills(t *testing.T) {
	msgBus := bus.NewMessageBus(10)
	defer msgBus.Close()

	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅入站消息失败: %v", err)
	}
	defer sub.Close()

	skill := types.Skill{
		Name:        "order-progress-analysis",
		DisplayName: "订单进度分析",
		Description: "查询订单进度",
		Path:        "/root/.codex/skills/order-progress-analysis/SKILL.md",
		Scope:       "user",
		Enabled:     true,
	}
	reg, hub := newTestRegistry(msgBus, fakeSkillAgents{
		names:  []string{"codex"},
		skills: map[string][]types.Skill{"codex": {skill}},
	})
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{
			ChatID:  "c1",
			Agent:   "codex",
			Content: "$" + skill.Name + " 看看这个订单",
		})
	reg.Dispatch(context.Background(), c, env)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	in, ok := sub.Consume(ctx)
	if !ok || in == nil {
		t.Fatal("未收到入站消息")
	}
	if len(in.Skills) != 1 || in.Skills[0] != skill {
		t.Fatalf("入站消息未携带校验后的技能: %+v", in.Skills)
	}
	if in.Content != "看看这个订单" {
		t.Fatalf("技能前缀应从正文移除，实际 %q", in.Content)
	}
}

func TestChatSendPassesSkillInSteerMessage(t *testing.T) {
	msgBus := bus.NewMessageBus(10)
	defer msgBus.Close()

	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅入站消息失败: %v", err)
	}
	defer sub.Close()

	skill := types.Skill{
		Name:    "order-progress-analysis",
		Path:    "/root/.codex/skills/order-progress-analysis/SKILL.md",
		Enabled: true,
	}
	reg, hub := newTestRegistry(msgBus, fakeSkillAgents{
		names:  []string{"codex"},
		skills: map[string][]types.Skill{"codex": {skill}},
	})
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{
			ChatID:  "c1",
			Agent:   "codex",
			Content: "/steer $order-progress-analysis 再看看交期",
		})
	reg.Dispatch(context.Background(), c, env)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	in, ok := sub.Consume(ctx)
	if !ok || in == nil {
		t.Fatal("未收到入站消息")
	}
	if len(in.Skills) != 1 || in.Skills[0] != skill {
		t.Fatalf("steer 消息未携带校验后的技能: %+v", in.Skills)
	}
	if in.Content != "/steer 再看看交期" {
		t.Fatalf("steer 前缀应保留且技能标记应移除，实际 %q", in.Content)
	}
}

func TestChatSendPassesSkillInSteerMessageWithoutSpace(t *testing.T) {
	msgBus := bus.NewMessageBus(10)
	defer msgBus.Close()

	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅入站消息失败: %v", err)
	}
	defer sub.Close()

	skill := types.Skill{
		Name:    "order-progress-analysis",
		Path:    "/root/.codex/skills/order-progress-analysis/SKILL.md",
		Enabled: true,
	}
	reg, hub := newTestRegistry(msgBus, fakeSkillAgents{
		names:  []string{"codex"},
		skills: map[string][]types.Skill{"codex": {skill}},
	})
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{
			ChatID:  "c1",
			Agent:   "codex",
			Content: "/steer$order-progress-analysis 再看看交期",
		})
	reg.Dispatch(context.Background(), c, env)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	in, ok := sub.Consume(ctx)
	if !ok || in == nil {
		t.Fatal("未收到入站消息")
	}
	if len(in.Skills) != 1 || in.Skills[0] != skill {
		t.Fatalf("无空格 steer 消息未携带校验后的技能: %+v", in.Skills)
	}
	if in.Content != "/steer 再看看交期" {
		t.Fatalf("无空格 steer 应规范化为命令前缀，实际 %q", in.Content)
	}
}

func TestChatSendRejectsSkillNotEnabledForAgent(t *testing.T) {
	msgBus := bus.NewMessageBus(2)
	defer msgBus.Close()

	reg, hub := newTestRegistry(msgBus, fakeSkillAgents{
		names: []string{"codex"},
		skills: map[string][]types.Skill{
			"codex": {{
				Name:    "disabled-skill",
				Path:    "/root/.codex/skills/disabled-skill/SKILL.md",
				Enabled: false,
			}},
		},
	})
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{
			ChatID:  "c1",
			Agent:   "codex",
			Content: "$disabled-skill hi",
		})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeSkillNotFound {
		t.Fatalf("错误码应为 skill_not_found，实际 %s", body.Code)
	}
}

func TestChatSendRejectsUnknownSkillPrefix(t *testing.T) {
	msgBus := bus.NewMessageBus(2)
	defer msgBus.Close()

	reg, hub := newTestRegistry(msgBus, fakeSkillAgents{names: []string{"codex"}})
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{
			ChatID:  "c1",
			Agent:   "codex",
			Content: "$missing-skill hi",
		})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeSkillNotFound {
		t.Fatalf("错误码应为 skill_not_found，实际 %s", body.Code)
	}
}

func TestChatSendIsolatesUsers(t *testing.T) {
	msgBus := bus.NewMessageBus(10)
	defer msgBus.Close()

	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅入站消息失败: %v", err)
	}
	defer sub.Close()

	reg, hub := newTestRegistry(msgBus, nil)
	c := newTestClientAs(hub, "alice")

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{ChatID: "c1", Content: "你好"})
	reg.Dispatch(context.Background(), c, env)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	in, ok := sub.Consume(ctx)
	if !ok || in == nil {
		t.Fatal("未收到入站消息")
	}
	if in.Session != "ws_alice_c1" {
		t.Fatalf("会话键应包含用户维度，实际 %s", in.Session)
	}
	if in.AccountID != "alice" || in.SenderID != "alice" {
		t.Fatalf("账号与发送者应为用户身份，实际 account=%s sender=%s", in.AccountID, in.SenderID)
	}
}

func TestChatSendRejectsForeignSession(t *testing.T) {
	reg, hub := newTestRegistry(nil, nil)
	c := newTestClientAs(hub, "alice")

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{SessionID: "ws_bob_c1", Content: "hi"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeForbidden {
		t.Fatalf("跨用户发送应被拒绝，错误码 %s", body.Code)
	}
}

func TestChatSendUsesClientChatID(t *testing.T) {
	msgBus := bus.NewMessageBus(10)
	defer msgBus.Close()

	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅入站消息失败: %v", err)
	}
	defer sub.Close()

	reg, hub := newTestRegistry(msgBus, nil)
	c := newTestClientAs(hub, "alice")
	// 模拟 hello 阶段服务端确定的当前对话
	c.SetChatID("chat-from-hello")

	// 前端只带 token，不传 chatID / sessionID
	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{Content: "继续"})
	reg.Dispatch(context.Background(), c, env)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	in, ok := sub.Consume(ctx)
	if !ok || in == nil {
		t.Fatal("未收到入站消息")
	}
	if in.ChatID != "chat-from-hello" {
		t.Fatalf("应沿用连接上的对话标识，实际 %s", in.ChatID)
	}
	if in.Session != "ws_alice_chat-from-hello" {
		t.Fatalf("会话键不符合预期: %s", in.Session)
	}
}

func TestChatSendSwitchesChatID(t *testing.T) {
	msgBus := bus.NewMessageBus(10)
	defer msgBus.Close()

	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅入站消息失败: %v", err)
	}
	defer sub.Close()

	reg, hub := newTestRegistry(msgBus, nil)
	c := newTestClientAs(hub, "alice")
	c.SetChatID("chat-a")

	// 显式传 chatID 时应切换到新对话，并更新连接上下文
	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{ChatID: "chat-b", Content: "新对话"})
	reg.Dispatch(context.Background(), c, env)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	in, ok := sub.Consume(ctx)
	if !ok || in == nil {
		t.Fatal("未收到入站消息")
	}
	if in.ChatID != "chat-b" || in.Session != "ws_alice_chat-b" {
		t.Fatalf("应切换到指定对话: chatID=%s session=%s", in.ChatID, in.Session)
	}
	if c.ChatID() != "chat-b" {
		t.Fatalf("连接上下文应更新为新对话，实际 %s", c.ChatID())
	}
}

func TestChatSendRestoresSelectedThread(t *testing.T) {
	msgBus := bus.NewMessageBus(10)
	defer msgBus.Close()

	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅入站消息失败: %v", err)
	}
	defer sub.Close()

	turns := &fakeTurnReader{threads: []db.SessionThread{
		{ThreadID: "t-history", SessionID: "ws_web_c1", Agent: "coder"},
	}}
	reg, hub := newTestRegistryWithTurns(msgBus, fakeAgents{"codex", "coder"}, turns)
	c := newTestClient(hub)

	// 只传 threadID 时从线程记录反推会话与对话，并沿用该线程的 agent。
	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{
			ThreadID: "t-history",
			Content:  "继续这个问题",
			// metadata 里的 thread 标记不可信，必须以顶层校验后的 threadID 为准
			Metadata: map[string]any{types.MetadataThreadID: "t-forged"},
		})
	reg.Dispatch(context.Background(), c, env)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	in, ok := sub.Consume(ctx)
	if !ok || in == nil {
		t.Fatal("未收到入站消息")
	}
	if in.Session != "ws_web_c1" || in.ChatID != "c1" {
		t.Fatalf("应从 thread 反推会话: session=%s chat=%s", in.Session, in.ChatID)
	}
	if in.AgentName != "coder" {
		t.Fatalf("应沿用 thread 的 agent，实际 %s", in.AgentName)
	}
	if got := in.ThreadID(); got != "t-history" {
		t.Fatalf("metadata 应携带指定 thread，实际 %q: %+v", got, in.Metadata)
	}
}

func TestChatSendRejectsThreadFromOtherSession(t *testing.T) {
	turns := &fakeTurnReader{threads: []db.SessionThread{
		{ThreadID: "t-history", SessionID: "ws_web_c2", Agent: "codex"},
	}}
	reg, hub := newTestRegistryWithTurns(nil, fakeAgents{"codex"}, turns)
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{SessionID: "ws_web_c1", ThreadID: "t-history", Content: "hi"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeBadRequest {
		t.Fatalf("thread 与 session 不一致应被拒绝，错误码 %s", body.Code)
	}
}

func TestChatSendRejectsForeignThread(t *testing.T) {
	turns := &fakeTurnReader{threads: []db.SessionThread{
		{ThreadID: "t-history", SessionID: "ws_bob_c1", Agent: "codex"},
	}}
	reg, hub := newTestRegistryWithTurns(nil, fakeAgents{"codex"}, turns)
	c := newTestClientAs(hub, "alice")

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend).
		WithData(&protocol.ChatSendData{ThreadID: "t-history", Content: "hi"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeForbidden {
		t.Fatalf("跨用户 thread 应被拒绝，错误码 %s", body.Code)
	}
}
