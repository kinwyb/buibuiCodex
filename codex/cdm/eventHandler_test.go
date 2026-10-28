package cdm

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
)

// TestClientEventClearsTurnDelta 覆盖增量事件的回收时机：
// 事件照旧落库，但 item/completed 一到，该轮逐 token 的 delta 必须被清掉；
// 清理按 turn 隔离，不能动其他轮次的增量。
func TestClientEventClearsTurnDelta(t *testing.T) {
	storage, err := db.NewSQLiteStorage(filepath.Join(t.TempDir(), "cdm.db"))
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	data := db.NewData(storage)
	defer data.Close()

	sess := data.Session()
	ctx := context.Background()
	a := NewAgent(&config.AgentConfig{Name: "codex", SessionDB: sess}, 0)

	notify := func(method codex.NotificationMethod, turnID, raw string) {
		a.clientEvent(&codex.Event{
			Type:    codex.EventTypeNotification,
			Method:  string(method),
			TurnID:  turnID,
			RawData: []byte(raw),
		})
	}

	notify(codex.ItemAgentMessageDelta, "u1", `{"itemId":"i1","delta":"你"}`)
	notify(codex.ItemAgentMessageDelta, "u1", `{"itemId":"i1","delta":"好"}`)
	notify(codex.ItemAgentMessageDelta, "u2", `{"itemId":"i2","delta":"保"}`)
	notify(codex.ItemCompleted, "u1", `{"item":{"id":"i1","type":"agentMessage","text":"你好"}}`)

	u1, err := sess.TurnEventList(ctx, db.TurnEventQuery{TurnIDs: []string{"u1"}})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(u1) != 1 || u1[0].Method != string(codex.ItemCompleted) {
		t.Fatalf("u1 应只剩 item/completed，实际 %+v", u1)
	}

	u2, err := sess.TurnEventList(ctx, db.TurnEventQuery{TurnIDs: []string{"u2"}})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(u2) != 1 || u2[0].Method != string(codex.ItemAgentMessageDelta) {
		t.Fatalf("u2 的增量不应被清理，实际 %+v", u2)
	}
}

// Test_lifeEventParse_dynamicToolCall 覆盖回归场景：
// dynamicToolCall 的 item/started、item/completed 必须产出 tool_start / tool_completed，
// 且载荷里带上工具名与入参。曾经因为类型映射存在第二份表，该 item 的事件类型被清空，
// 事件在 codexEventParse 里被丢弃，前端收不到任何工具调用帧。
func Test_lifeEventParse_dynamicToolCall(t *testing.T) {
	started := `{"item":{"id":"item_1","type":"dynamicToolCall","callId":"call_1","namespace":"heldiamfs","tool":"read_file","arguments":{"path":"a.go"}}}`
	completed := `{"item":{"id":"item_1","type":"dynamicToolCall","callId":"call_1","namespace":"heldiamfs","tool":"read_file","arguments":{"path":"a.go"},"result":"package a"}}`

	cases := []struct {
		name       string
		method     codex.NotificationMethod
		raw        string
		wantType   types.EventType
		wantResult string
	}{
		{
			name:     "started 产出 tool_start",
			method:   codex.ItemStarted,
			raw:      started,
			wantType: types.EventToolStart,
		},
		{
			name:       "completed 产出 tool_completed 并带结果",
			method:     codex.ItemCompleted,
			raw:        completed,
			wantType:   types.EventToolCompleted,
			wantResult: "package a",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &Agent{}
			state := types.NewState(&types.InputMessage{
				ID:      "req_1",
				Session: "ws_user_chat",
				Channel: "ws",
				ChatID:  "chat",
			})
			var got []*types.Event
			state.EventHandler = func(event *types.Event) { got = append(got, event) }

			notification := &codex.Event{
				Type:    codex.EventTypeNotification,
				Method:  string(tc.method),
				RawData: []byte(tc.raw),
			}
			a.codexEventParse(state, notification)

			if len(got) != 1 {
				t.Fatalf("应发布 1 个事件，实际 %d 个（事件类型被清空会导致整条被丢弃）", len(got))
			}
			event := got[0]
			if event.Type != tc.wantType {
				t.Fatalf("事件类型不符：want %s, got %s", tc.wantType, event.Type)
			}
			if event.Message == nil || len(event.Message.ToolCalls) != 1 {
				t.Fatalf("工具调用载荷缺失: %+v", event.Message)
			}
			call := event.Message.ToolCalls[0]
			if call.Name != "read_file" || call.ID != "item_1" {
				t.Fatalf("工具标识不符: %+v", call)
			}
			if call.Params != `{"path":"a.go"}` {
				t.Fatalf("工具入参不符: %q", call.Params)
			}
			if tc.wantResult != "" && call.Result != tc.wantResult {
				t.Fatalf("工具结果不符: %q", call.Result)
			}
			if event.EventID != "item_1" {
				t.Fatalf("EventID 应复用 item ID: %q", event.EventID)
			}
		})
	}
}

// Test_lifeEventParse_toolResultFromContentItems 用线上真实报文验证工具结果能被带出去。
// codex 的 item/completed 里没有 result 字段，结果在 contentItems[].text；
// 早期实现只读 result，导致前端永远拿到空结果。
func Test_lifeEventParse_toolResultFromContentItems(t *testing.T) {
	started := `{"item":{"type":"dynamicToolCall","id":"call_4d19a68d5f2f4ba7b5f27273","namespace":"base","tool":"send_file","arguments":{"caption":"测试文件重发","file_path":"/workspace/test_file.txt"},"status":"inProgress","contentItems":null,"success":null,"durationMs":null},"threadId":"th","turnId":"tu","startedAtMs":1789434686293}`
	completed := `{"item":{"type":"dynamicToolCall","id":"call_4d19a68d5f2f4ba7b5f27273","namespace":"base","tool":"send_file","arguments":{"caption":"测试文件重发","file_path":"/workspace/test_file.txt"},"status":"completed","contentItems":[{"type":"inputText","text":"✅ 文件已成功发送"}],"success":true,"durationMs":865},"threadId":"th","turnId":"tu","completedAtMs":1789434687158}`

	parse := func(method codex.NotificationMethod, raw string) *types.Event {
		state := types.NewState(&types.InputMessage{ID: "req", Session: "ws_user_chat", Channel: "ws"})
		var got *types.Event
		state.EventHandler = func(event *types.Event) { got = event }
		(&Agent{}).codexEventParse(state, &codex.Event{
			Type:    codex.EventTypeNotification,
			Method:  string(method),
			RawData: []byte(raw),
		})
		return got
	}

	startEvent := parse(codex.ItemStarted, started)
	if startEvent == nil || startEvent.Type != types.EventToolStart {
		t.Fatalf("开始帧应为 tool_start: %+v", startEvent)
	}
	startCall := startEvent.Message.ToolCalls[0]
	if startCall.Result != "" {
		t.Fatalf("开始帧不应带结果，实际 %q", startCall.Result)
	}
	if startCall.Extra["status"] != "inProgress" {
		t.Fatalf("开始帧应带 status=inProgress: %+v", startCall.Extra)
	}

	doneEvent := parse(codex.ItemCompleted, completed)
	if doneEvent == nil || doneEvent.Type != types.EventToolCompleted {
		t.Fatalf("完成帧应为 tool_completed: %+v", doneEvent)
	}
	doneCall := doneEvent.Message.ToolCalls[0]
	if doneCall.Result != "✅ 文件已成功发送" {
		t.Fatalf("结果应取自 contentItems[].text，实际 %q", doneCall.Result)
	}
	if doneCall.Name != "send_file" {
		t.Fatalf("工具名不符: %q", doneCall.Name)
	}
	if doneCall.Extra["success"] != true {
		t.Fatalf("完成帧应带 success: %+v", doneCall.Extra)
	}
	if doneCall.Extra["durationMs"] != int64(865) {
		t.Fatalf("完成帧应带耗时: %+v", doneCall.Extra)
	}
	if doneCall.Extra["status"] != "completed" {
		t.Fatalf("完成帧 status 不符: %+v", doneCall.Extra)
	}
	// 同一 item 的开始帧与完成帧必须共享 ID，前端按它覆盖更新同一条工具记录
	if startEvent.EventID != doneEvent.EventID {
		t.Fatalf("开始/完成帧 eventID 应一致: %q != %q", startEvent.EventID, doneEvent.EventID)
	}
}

// Test_lifeEventParse_startedResultNotLiteralNull 确认开始帧不会把 null 结果渲染成字面量。
func Test_lifeEventParse_startedResultNotLiteralNull(t *testing.T) {
	raw := `{"item":{"id":"item_2","type":"dynamicToolCall","tool":"read_file","arguments":{},"result":null}}`
	a := &Agent{}
	state := types.NewState(&types.InputMessage{ID: "req_2", Session: "ws_user_chat", Channel: "ws"})
	var got *types.Event
	state.EventHandler = func(event *types.Event) { got = event }

	a.codexEventParse(state, &codex.Event{
		Type:    codex.EventTypeNotification,
		Method:  string(codex.ItemStarted),
		RawData: []byte(raw),
	})

	if got == nil || len(got.Message.ToolCalls) != 1 {
		t.Fatalf("未产出工具事件: %+v", got)
	}
	if result := got.Message.ToolCalls[0].Result; result != "" {
		t.Fatalf("空结果应为空串，实际 %q", result)
	}
	if params := got.Message.ToolCalls[0].Params; params != "" {
		t.Fatalf("空入参应为空串，实际 %q", params)
	}
}
