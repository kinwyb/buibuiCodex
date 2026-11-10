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

func TestClientEventLifecycleIsIdempotent(t *testing.T) {
	storage, err := db.NewSQLiteStorage(filepath.Join(t.TempDir(), "cdm-idempotent.db"))
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	data := db.NewData(storage)
	defer data.Close()

	sess := data.Session()
	a := NewAgent(&config.AgentConfig{Name: "codex", SessionDB: sess}, 0)
	event := &codex.Event{
		Type:     codex.EventTypeNotification,
		Method:   string(codex.ItemCompleted),
		ThreadID: "t1",
		TurnID:   "u1",
		RawData:  []byte(`{"item":{"id":"i1","type":"commandExecution","status":"completed"}}`),
	}
	a.clientEvent(event)
	a.clientEvent(event)

	events, err := sess.TurnEventList(context.Background(), db.TurnEventQuery{
		TurnIDs: []string{"u1"},
		Methods: []string{"item/completed"},
	})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("同一 item 的完成事件应幂等，实际 %d 条", len(events))
	}
}

func TestClientEventRecoveryRunUsesLogicalTurnID(t *testing.T) {
	storage, err := db.NewSQLiteStorage(filepath.Join(t.TempDir(), "cdm-recovery-run.db"))
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	data := db.NewData(storage)
	defer data.Close()

	sess := data.Session()
	a := NewAgent(&config.AgentConfig{Name: "codex", SessionDB: sess}, 0)
	a.epMu.Lock()
	a.turnRootMap["run-1"] = "logical-1"
	a.turnRootMap["run-2"] = "logical-1"
	a.epMu.Unlock()

	notify := func(method codex.NotificationMethod, runTurnID, raw string) {
		a.clientEvent(&codex.Event{
			Type:     codex.EventTypeNotification,
			Method:   string(method),
			ThreadID: "t1",
			TurnID:   runTurnID,
			RawData:  []byte(raw),
		})
	}
	notify(codex.ItemAgentMessageDelta, "run-1", `{"itemId":"i1","delta":"旧"}`)
	notify(codex.ItemAgentMessageDelta, "run-2", `{"itemId":"i2","delta":"新"}`)
	notify(codex.ItemCompleted, "run-2", `{"item":{"id":"i2","type":"agentMessage","text":"新"}}`)

	events, err := sess.TurnEventList(context.Background(), db.TurnEventQuery{TurnIDs: []string{"logical-1"}})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("逻辑 TurnID 下应保留旧 run delta 和新 run 完成帧，实际 %+v", events)
	}
	for _, event := range events {
		if event.TurnID != "logical-1" {
			t.Fatalf("事件应按逻辑 TurnID 落库: %+v", event)
		}
	}
	if events[0].RunTurnID != "run-1" || events[1].RunTurnID != "run-2" {
		t.Fatalf("RunTurnID 应保留实际运行归属: %+v", events)
	}
}

func TestClientEventTurnStartedLearnsRootTurnID(t *testing.T) {
	storage, err := db.NewSQLiteStorage(filepath.Join(t.TempDir(), "cdm-root-turn.db"))
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	data := db.NewData(storage)
	defer data.Close()

	sess := data.Session()
	a := NewAgent(&config.AgentConfig{Name: "codex", SessionDB: sess}, 0)
	a.clientEvent(&codex.Event{
		Type:     codex.EventTypeNotification,
		Method:   string(codex.TurnStarted),
		ThreadID: "t1",
		TurnID:   "run-2",
		RawData:  []byte(`{"threadId":"t1","turn":{"id":"run-2","rootTurnId":"logical-1","status":"inProgress"}}`),
	})

	events, err := sess.TurnEventList(context.Background(), db.TurnEventQuery{TurnIDs: []string{"logical-1"}})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(events) != 1 || events[0].RunTurnID != "run-2" {
		t.Fatalf("turn/started 应学习 rootTurnId 并落到逻辑 TurnID 下: %+v", events)
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
			if call.Type != types.ToolTypeFunction {
				t.Fatalf("本地动态工具应保持 type=function: %+v", call)
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

func Test_lifeEventParse_uiEventBatchAsToolStep(t *testing.T) {
	raw := `{"item":{"id":"ui_1","type":"userMessage","text":"<ui_events>{\"v\":1,\"events\":[{\"ts\":1,\"view\":\"V1\",\"event\":\"selection.change\"}]}</ui_events>\n请分析"}}`
	a := &Agent{}
	state := types.NewState(&types.InputMessage{ID: "req_ui", Session: "ws_user_chat", Channel: "ws"})
	var got *types.Event
	state.EventHandler = func(event *types.Event) { got = event }

	a.codexEventParse(state, &codex.Event{
		Type:    codex.EventTypeNotification,
		Method:  string(codex.ItemCompleted),
		RawData: []byte(raw),
	})

	if got == nil || got.Type != types.EventToolCompleted {
		t.Fatalf("UI 事件批次应映射成 tool_completed: %+v", got)
	}
	if len(got.Message.ToolCalls) != 1 {
		t.Fatalf("应带一个工具步骤: %+v", got.Message)
	}
	call := got.Message.ToolCalls[0]
	if call.Name != "ui_event" || call.Extra["uiEvent"] != true || call.Extra["userText"] != "请分析" {
		t.Fatalf("UI 事件工具步骤内容不符: %+v", call)
	}
}

// Test_lifeEventParse_mcpToolCall 用线上真实报文覆盖 MCP 工具调用。
// mcpToolCall 曾经没有映射分支，落到 default 只打日志，事件类型为空被丢弃，
// 前端因此完全看不到 MCP 调用；这里锁住开始帧、成功帧、失败帧三种形态。
func Test_lifeEventParse_mcpToolCall(t *testing.T) {
	started := `{"item":{"type":"mcpToolCall","id":"call_eba67ffeb6db4ba28df91e32","server":"boda","tool":"order_efficiency","status":"inProgress","arguments":{"startDate":"2026-09-15","endDate":"2026-09-15"},"appContext":null,"pluginId":null,"readOnlyHint":null,"result":null,"error":null,"durationMs":null},"threadId":"th","turnId":"tu","startedAtMs":1789528357382}`
	completed := `{"item":{"type":"mcpToolCall","id":"call_8be43da6f2b845edbcd438a6","server":"codex","tool":"list_mcp_resources","status":"completed","arguments":{},"result":{"content":[{"type":"text","text":"{\"resources\":[]}"}],"structuredContent":null,"_meta":null},"error":null,"durationMs":120},"threadId":"th","turnId":"tu","completedAtMs":1789528358000}`
	failed := `{"item":{"type":"mcpToolCall","id":"call_5d509bb8a3084dfab40983a4","server":"boda","tool":"list_mcp_resources","status":"failed","arguments":{"server":"boda"},"result":null,"error":{"message":"resources/list failed: MCP startup failed"},"durationMs":0},"threadId":"th","turnId":"tu","completedAtMs":1789528359000}`

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
	// type 就是前端区分 MCP 与本地工具的标记
	if startCall.Name != "order_efficiency" || startCall.Type != types.ToolTypeMCP {
		t.Fatalf("MCP 工具应标记为 type=mcp: %+v", startCall)
	}
	if startCall.Params != `{"endDate":"2026-09-15","startDate":"2026-09-15"}` {
		t.Fatalf("工具入参不符: %q", startCall.Params)
	}
	if startCall.Result != "" {
		t.Fatalf("开始帧 result 为 null，不应渲染出内容: %q", startCall.Result)
	}
	if startCall.Extra["server"] != "boda" {
		t.Fatalf("开始帧应带 MCP 服务名: %+v", startCall.Extra)
	}
	if _, ok := startCall.Extra["success"]; ok {
		t.Fatalf("inProgress 阶段不应表态 success: %+v", startCall.Extra)
	}

	doneEvent := parse(codex.ItemCompleted, completed)
	if doneEvent == nil || doneEvent.Type != types.EventToolCompleted {
		t.Fatalf("完成帧应为 tool_completed: %+v", doneEvent)
	}
	doneCall := doneEvent.Message.ToolCalls[0]
	if doneCall.Result != `{"resources":[]}` {
		t.Fatalf("结果应展开 result.content[].text，实际 %q", doneCall.Result)
	}
	if doneCall.Params != "" {
		t.Fatalf("空入参应为空串，实际 %q", doneCall.Params)
	}
	if doneCall.Extra["success"] != true || doneCall.Extra["server"] != "codex" {
		t.Fatalf("完成帧 extra 不符: %+v", doneCall.Extra)
	}
	if doneCall.Extra["durationMs"] != int64(120) {
		t.Fatalf("完成帧应带耗时: %+v", doneCall.Extra)
	}

	failEvent := parse(codex.ItemCompleted, failed)
	if failEvent == nil || failEvent.Type != types.EventToolCompleted {
		t.Fatalf("失败也应是 tool_completed，让前端收敛这一条: %+v", failEvent)
	}
	failCall := failEvent.Message.ToolCalls[0]
	if failCall.Extra["success"] != false {
		t.Fatalf("失败帧应带 success=false: %+v", failCall.Extra)
	}
	if failCall.Extra["error"] != "resources/list failed: MCP startup failed" {
		t.Fatalf("失败帧应带错误信息: %+v", failCall.Extra)
	}
	if failCall.Result != "resources/list failed: MCP startup failed" {
		t.Fatalf("无结果时应退回错误信息，实际 %q", failCall.Result)
	}
}
