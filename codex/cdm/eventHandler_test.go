package cdm

import (
	"testing"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/core/types"
)

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
