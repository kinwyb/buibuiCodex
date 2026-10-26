// 本文件测试 session.history，含执行过程帧的还原。

package command

import (
	"context"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

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
