package command

import (
	"context"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

type fakeAgents []string

func (f fakeAgents) ListAgents() []string { return []string(f) }

func newTestRegistry(msgBus *bus.MessageBus, agents AgentSource) (*Registry, *conn.Hub) {
	return newTestRegistryWithTurns(msgBus, agents, nil)
}

func newTestRegistryWithTurns(msgBus *bus.MessageBus, agents AgentSource, turns TurnReader) (*Registry, *conn.Hub) {
	hub := conn.NewHub()
	return NewRegistry(Deps{
		Bus:          msgBus,
		Hub:          hub,
		Agents:       agents,
		Turns:        turns,
		Channel:      "ws",
		DefaultAgent: "codex",
	}), hub
}

func newTestClient(hub *conn.Hub) *conn.Client {
	return newTestClientAs(hub, "web")
}

func newTestClientAs(hub *conn.Hub, userID string) *conn.Client {
	c := conn.NewClient(conn.Config{SendBuffer: 8}, nil, hub, nil, conn.Identity{UserID: userID})
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
	if got := reg.SessionKey("web", "c1"); got != in.SessionID() {
		t.Fatalf("会话键不一致: %s != %s", got, in.SessionID())
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

func TestChatCancelRejectsForeignSession(t *testing.T) {
	reg, hub := newTestRegistry(nil, nil)
	c := newTestClientAs(hub, "alice")

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatCancel).
		WithData(&protocol.ChatCancelData{SessionID: "ws_bob_c1"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeForbidden {
		t.Fatalf("跨用户取消应被拒绝，错误码 %s", body.Code)
	}
}

func TestSessionHistoryRejectsForeignSession(t *testing.T) {
	reg, hub := newTestRegistryWithTurns(nil, nil, &fakeTurnReader{})
	c := newTestClientAs(hub, "alice")

	resp := historyRequest(t, reg, c, protocol.SessionHistoryData{SessionID: "ws_bob_c1"})
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeForbidden {
		t.Fatalf("跨用户读历史应被拒绝，错误码 %s", body.Code)
	}
	if hub.SessionCount() != 0 {
		t.Fatalf("拒绝后不应订阅该会话，实际 %d", hub.SessionCount())
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

func TestChatCancelUsesClientChatID(t *testing.T) {
	reg, hub := newTestRegistry(nil, nil)
	c := newTestClientAs(hub, "alice")
	c.SetChatID("chat-a")

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatCancel)
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	// 未提供取消器时应走到 internal，说明 session 已从连接上下文推导出来
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeInternal {
		t.Fatalf("应从连接上下文推导会话，错误码 %s", body.Code)
	}
}

func TestOwnsSessionPrefix(t *testing.T) {
	reg, _ := newTestRegistry(nil, nil)
	cases := []struct {
		userID    string
		sessionID string
		want      bool
	}{
		{"alice", "ws_alice_c1", true},
		{"alice", "ws_alice_", true},
		{"alice", "ws_bob_c1", false},
		// 关键：alice 不能匹配到 alice2 的会话
		{"alice", "ws_alice2_c1", false},
		{"alice", "alice_c1", false},
		{"", "ws_alice_c1", false},
	}
	for _, c := range cases {
		if got := reg.ownsSession(c.userID, c.sessionID); got != c.want {
			t.Fatalf("ownsSession(%q, %q) = %v，期望 %v", c.userID, c.sessionID, got, c.want)
		}
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

// ---- 历史消息 ----

type fakeTurnReader struct {
	threads      []db.SessionThread
	turns        map[string][]db.SessionTurn
	queries      []db.TurnQuery
	turnEvents   map[string][]db.TurnEvent
	eventQueries []db.TurnEventQuery
}

func (f *fakeTurnReader) ThreadList(_ context.Context, _ string) ([]db.SessionThread, error) {
	return f.threads, nil
}

func (f *fakeTurnReader) TurnList(_ context.Context, q db.TurnQuery) ([]db.SessionTurn, error) {
	f.queries = append(f.queries, q)
	return f.turns[q.ThreadID], nil
}

func (f *fakeTurnReader) TurnQueryByID(_ context.Context, turnID string) *db.SessionTurn {
	for _, list := range f.turns {
		for i := range list {
			if list[i].TurnID == turnID {
				return &list[i]
			}
		}
	}
	return nil
}

func (f *fakeTurnReader) TurnEventList(_ context.Context, q db.TurnEventQuery) ([]db.TurnEvent, error) {
	f.eventQueries = append(f.eventQueries, q)
	var out []db.TurnEvent
	for _, id := range q.TurnIDs {
		out = append(out, f.turnEvents[id]...)
	}
	return out, nil
}

func historyRequest(t *testing.T, reg *Registry, c *conn.Client, data protocol.SessionHistoryData) *protocol.Envelope {
	t.Helper()
	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeSessionHistory).WithData(data)
	reg.Dispatch(context.Background(), c, env)

	select {
	case raw := <-c.Outbound():
		resp, body := protocol.DecodeEnvelope(raw)
		if body != nil {
			t.Fatalf("解析响应失败: %+v", body)
		}
		return resp
	default:
		t.Fatal("未返回响应")
		return nil
	}
}

func TestSessionHistoryGroupsByThread(t *testing.T) {
	reader := &fakeTurnReader{
		threads: []db.SessionThread{
			{ThreadID: "t1", SessionID: "s1", Agent: "codex"},
			{ThreadID: "t2", SessionID: "s1", Agent: "coder"}, // 无轮次，应被跳过
		},
		turns: map[string][]db.SessionTurn{
			"t1": {{ReqID: "r1", ThreadID: "t1", TurnID: "u1", Agent: "codex", Question: "q1", Answer: "a1", IsCompleted: true, Seq: 7}},
		},
	}
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	resp := historyRequest(t, reg, c, protocol.SessionHistoryData{SessionID: "ws_web_s1"})
	if resp.Kind != protocol.KindSystem || resp.Type != protocol.TypeSessionHistory {
		t.Fatalf("响应类型不符合预期: %s/%s", resp.Kind, resp.Type)
	}
	var result protocol.SessionHistoryResult
	if err := resp.DecodeData(&result); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if result.SessionID != "ws_web_s1" || len(result.Threads) != 1 {
		t.Fatalf("应按线程分组并跳过空线程: %+v", result)
	}
	th := result.Threads[0]
	if th.ThreadID != "t1" || len(th.Turns) != 1 {
		t.Fatalf("线程内容不符合预期: %+v", th)
	}
	if th.Turns[0].ReqID != "r1" || th.Turns[0].Seq != 7 || !th.Turns[0].Completed {
		t.Fatalf("轮次内容不符合预期: %+v", th.Turns[0])
	}
	if hub.SessionCount() != 1 {
		t.Fatalf("拉取历史后应订阅该会话，实际订阅数 %d", hub.SessionCount())
	}
}

func TestSessionHistoryWithThreadAndCursor(t *testing.T) {
	reader := &fakeTurnReader{
		turns: map[string][]db.SessionTurn{
			"t2": {{ReqID: "r9", ThreadID: "t2"}},
		},
	}
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	resp := historyRequest(t, reg, c, protocol.SessionHistoryData{
		SessionID: "ws_web_s1",
		ThreadID:  "t2",
		BeforeSeq: 30,
		Limit:     -1,
	})
	var result protocol.SessionHistoryResult
	_ = resp.DecodeData(&result)
	if len(result.Threads) != 1 || result.Threads[0].ThreadID != "t2" {
		t.Fatalf("指定线程应只返回该线程: %+v", result)
	}
	if len(reader.queries) != 1 {
		t.Fatalf("指定线程时只应查询一次，实际 %d", len(reader.queries))
	}
	q := reader.queries[0]
	if q.ThreadID != "t2" || q.BeforeSeq != 30 || q.SessionID != "ws_web_s1" {
		t.Fatalf("查询条件未透传: %+v", q)
	}
}

func TestSessionHistorySinceAndChatID(t *testing.T) {
	reader := &fakeTurnReader{
		turns: map[string][]db.SessionTurn{"": {{ReqID: "r1"}}},
	}
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	resp := historyRequest(t, reg, c, protocol.SessionHistoryData{ChatID: "c9", Since: 1700000000000})
	var result protocol.SessionHistoryResult
	_ = resp.DecodeData(&result)
	if result.SessionID != "ws_web_c9" {
		t.Fatalf("应通过 chatID 推导 sessionID，实际 %s", result.SessionID)
	}
	if len(reader.queries) != 1 {
		t.Fatalf("应查询一次，实际 %d", len(reader.queries))
	}
	if got := reader.queries[0].Since.UnixMilli(); got != 1700000000000 {
		t.Fatalf("since 未正确转换，实际 %d", got)
	}
}

func TestSessionHistoryFallbackWithoutThreads(t *testing.T) {
	reader := &fakeTurnReader{
		turns: map[string][]db.SessionTurn{"": {{ReqID: "legacy"}}},
	}
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	resp := historyRequest(t, reg, c, protocol.SessionHistoryData{SessionID: "ws_web_s1"})
	var result protocol.SessionHistoryResult
	_ = resp.DecodeData(&result)
	if len(result.Threads) != 1 || len(result.Threads[0].Turns) != 1 {
		t.Fatalf("没有线程记录时应回退为按会话查询: %+v", result)
	}
}

func TestSessionHistoryRejectsMissingSession(t *testing.T) {
	reg, hub := newTestRegistryWithTurns(nil, nil, &fakeTurnReader{})
	c := newTestClient(hub)

	resp := historyRequest(t, reg, c, protocol.SessionHistoryData{})
	if resp.Type != protocol.TypeError {
		t.Fatalf("缺少 sessionID 应返回 error，实际 %s", resp.Type)
	}
}

func TestSessionHistoryWithoutReader(t *testing.T) {
	reg, hub := newTestRegistryWithTurns(nil, nil, nil)
	c := newTestClient(hub)

	resp := historyRequest(t, reg, c, protocol.SessionHistoryData{SessionID: "ws_web_s1"})
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeInternal {
		t.Fatalf("未启用历史时错误码应为 internal，实际 %s", body.Code)
	}
}

// ---- 执行过程回放 ----

// replayReader 造一个带一轮对话及其事件的读取器
func replayReader() *fakeTurnReader {
	return &fakeTurnReader{
		threads: []db.SessionThread{{ThreadID: "t1", SessionID: "ws_web_s1", Agent: "codex"}},
		turns: map[string][]db.SessionTurn{
			"t1": {{
				Seq: 7, ReqID: "r1", ThreadID: "t1", TurnID: "u1", SessionID: "ws_web_s1",
				Agent: "codex", Question: "看下目录", Answer: "看完了", IsCompleted: true,
			}},
		},
		turnEvents: map[string][]db.TurnEvent{
			"u1": {
				{Seq: 11, TurnID: "u1", Method: "item/completed", RawData: `{"item":{"id":"i1","type":"reasoning","summary":["先看目录"]}}`},
				{Seq: 12, TurnID: "u1", Method: "item/completed", RawData: `{"item":{"id":"i2","type":"commandExecution","command":"ls","aggregatedOutput":"a.go","status":"completed"}}`},
				{Seq: 13, TurnID: "u1", Method: "item/completed", RawData: `{"item":{"id":"i3","type":"agentMessage","text":"看完了"}}`},
			},
		},
	}
}

// 历史里的中间过程必须是与实时同构的 agent 帧，前端才能共用一条渲染管线
func TestSessionHistoryIncludesAgentFrames(t *testing.T) {
	reader := replayReader()
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	resp := historyRequest(t, reg, c, protocol.SessionHistoryData{
		SessionID:     "ws_web_s1",
		IncludeFrames: true,
	})
	var result protocol.SessionHistoryResult
	if err := resp.DecodeData(&result); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if len(result.Frames) != 3 {
		t.Fatalf("应还原 3 帧，实际 %d", len(result.Frames))
	}

	wantTypes := []string{"reasoning_completed", "tool_completed", "message_completed"}
	for i, frame := range result.Frames {
		if frame.Kind != protocol.KindAgent {
			t.Fatalf("第 %d 帧应为 agent 域，实际 %s", i, frame.Kind)
		}
		if frame.Type != wantTypes[i] {
			t.Fatalf("第 %d 帧类型应为 %s，实际 %s", i, wantTypes[i], frame.Type)
		}
		// 帧要能和所属轮次对上，前端据此把过程挂到正确的问答下面
		if frame.ReqID != "r1" || frame.SessionID != "ws_web_s1" {
			t.Fatalf("第 %d 帧缺少会话/请求标识: %+v", i, frame)
		}
	}

	var reasoning protocol.AgentData
	if err := result.Frames[0].DecodeData(&reasoning); err != nil {
		t.Fatalf("解析思考帧失败: %v", err)
	}
	if reasoning.Reasoning != "先看目录" {
		t.Fatalf("思考内容不符合预期: %q", reasoning.Reasoning)
	}

	// 只查关键帧，不能把逐 token 的 delta 一起搬出来
	if len(reader.eventQueries) != 1 {
		t.Fatalf("应只查一次事件，实际 %d", len(reader.eventQueries))
	}
	for _, method := range reader.eventQueries[0].Methods {
		if method == "item/agentMessage/delta" {
			t.Fatalf("事件查询不应包含 delta: %v", reader.eventQueries[0].Methods)
		}
	}
}

// 不带 includeFrames 时不查事件，省掉无谓的 IO 与载荷
func TestSessionHistorySkipsFramesByDefault(t *testing.T) {
	reader := replayReader()
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	resp := historyRequest(t, reg, c, protocol.SessionHistoryData{SessionID: "ws_web_s1"})
	var result protocol.SessionHistoryResult
	_ = resp.DecodeData(&result)
	if len(result.Frames) != 0 {
		t.Fatalf("默认不应返回帧，实际 %d", len(result.Frames))
	}
	if len(reader.eventQueries) != 0 {
		t.Fatalf("默认不应查事件表，实际查询 %d 次", len(reader.eventQueries))
	}
}

func TestTurnReplayReturnsFullFrames(t *testing.T) {
	reader := replayReader()
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeTurnReplay).
		WithData(protocol.TurnReplayData{SessionID: "ws_web_s1", TurnID: "u1"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	if resp.Type != protocol.TypeTurnReplay || resp.ReplyTo != env.ID {
		t.Fatalf("响应类型不符合预期: %s/%s", resp.Type, resp.ReplyTo)
	}
	var result protocol.TurnReplayResult
	if err := resp.DecodeData(&result); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if result.TurnID != "u1" || len(result.Frames) != 3 {
		t.Fatalf("单轮回放结果不符合预期: %+v", result)
	}
	if result.Frames[0].Seq != 11 {
		t.Fatalf("帧应回填事件 seq，实际 %d", result.Frames[0].Seq)
	}
	if result.Frames[2].Kind != protocol.KindAgent {
		t.Fatalf("回放帧应与实时同域: %s", result.Frames[2].Kind)
	}
}

func TestTurnReplayRejectsForeignTurn(t *testing.T) {
	reader := replayReader()
	// 轮次属于别人的会话
	reader.turns["t1"][0].SessionID = "ws_bob_s1"
	reg, hub := newTestRegistryWithTurns(nil, nil, reader)
	c := newTestClientAs(hub, "alice")

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeTurnReplay).
		WithData(protocol.TurnReplayData{SessionID: "ws_alice_s1", TurnID: "u1"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeForbidden {
		t.Fatalf("跨会话读过程应被拒绝，错误码 %s", body.Code)
	}
}

func TestTurnReplayRejectsMissingTurnID(t *testing.T) {
	reg, hub := newTestRegistryWithTurns(nil, nil, replayReader())
	c := newTestClient(hub)

	env := protocol.NewEnvelope(protocol.KindClient, protocol.TypeTurnReplay).
		WithData(protocol.TurnReplayData{SessionID: "ws_web_s1"})
	reg.Dispatch(context.Background(), c, env)

	raw := <-c.Outbound()
	resp, _ := protocol.DecodeEnvelope(raw)
	var body protocol.ErrorBody
	_ = resp.DecodeData(&body)
	if body.Code != protocol.CodeBadRequest {
		t.Fatalf("缺少 turnID 应返回 bad_request，实际 %s", body.Code)
	}
}
