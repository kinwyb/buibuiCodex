package cdm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
)

// deltaEventMethods 增量事件的 method 清单：逐 token 的文本增量与命令/文件输出流。
// 这类事件只在生成过程中有意义——item 一完成，完整内容就随 item/completed 落库了，
// 碎片再留着既不能用于回放（回放只取关键帧），又会让 turn_events 表无限膨胀。
var deltaEventMethods = []string{
	string(codex.ItemAgentMessageDelta),
	string(codex.ItemReasoningTextDelta),
	string(codex.ItemReasoningSummaryTextDelta),
	string(codex.ItemPlanDelta),
	string(codex.ItemCommandExecutionOutputDelta),
	string(codex.ItemFileChangeOutputDelta),
	string(codex.CommandExecOutputDelta),
	string(codex.ProcessOutputDelta),
}

// clientEvent 这里可以获取到所有客户的事件,注意这里事件和thread，turn等事件是重复分发的,注意重复处理问题
func (a *Agent) clientEvent(event *codex.Event) {
	if a.cfg == nil || a.cfg.SessionDB == nil {
		return
	}
	// 将事件保存入数据库
	dbErr := a.cfg.SessionDB.TurnEventSave(context.Background(), &db.TurnEvent{
		ID:         uuid.NewV4().String(),
		ThreadID:   event.ThreadID,
		TurnID:     event.TurnID,
		Type:       string(event.Type),
		Method:     event.Method,
		RawData:    string(event.RawData),
		CreateTime: time.Now(),
	})
	if dbErr != nil {
		slog.Error("fail to save db thread event", "error", dbErr.Error())
	}
	// item 结束即回收该轮的增量事件。codex 的 item 在一轮内是串行的，
	// 此刻已完成 item 的 delta 全部失效，未开始 item 的 delta 尚未产生，
	// 因此按 turn 粒度清理是安全的，而且不必逐 item 比对 ID。
	if codex.NotificationMethod(event.Method) == codex.ItemCompleted {
		if err := a.cfg.SessionDB.TurnEventDeleteByMethods(context.Background(), event.TurnID, deltaEventMethods); err != nil {
			slog.Error("fail to delete turn delta events", "error", err.Error(), "turnID", event.TurnID)
		}
	}
}

// unknowThreadEventHandler 无法被消化的thread事件处理，正常进入thread的事件都会被turn消化，但是比如turn已经结束或者异常导致无法分发的消息会被这里接手
func (a *Agent) unknowThreadEventHandler(event *codex.Event) {
	slog.Warn("未知的turn事件", "event", event)
	turn := a.cfg.SessionDB.TurnQueryByID(context.Background(), event.TurnID)
	if turn == nil {
		slog.Error("fail to find turn", "turnID", event.TurnID)
		return
	}
	var state *types.State
	a.epMu.Lock()
	if ct, ok := a.eps[turn.SessionID]; ok {
		state = ct.state
	}
	a.epMu.Unlock()
	if state == nil {
		session := a.cfg.SessionDB.SessionQueryByID(context.Background(), turn.SessionID)
		if session == nil {
			slog.Error("fail to find session", "sessionID", turn.SessionID)
			return
		}
		state = types.NewState(&types.InputMessage{
			ID:        turn.ReqID,
			Session:   turn.SessionID,
			Channel:   session.Channel,
			AccountID: session.AccountID,
			SenderID:  session.UserID,
			ChatID:    session.ChatID,
			Content:   turn.Question,
			AgentName: a.AgentID(),
			Timestamp: time.Now(),
		})
	}
	a.codexEventParse(state, event)
}

// turnEvent 对话事件处理
func (a *Agent) turnEvent(state *types.State, completed chan<- error) func(event *codex.Event) {
	return func(event *codex.Event) {
		a.codexEventParse(state, event)
		if event.Method == string(codex.TurnCompleted) {
			a.epMu.Lock()
			if sessionID, ok := a.turnMap[event.TurnID]; ok {
				if ct, ok2 := a.eps[sessionID]; ok2 {
					ct.turn = nil
					ct.lastUpdateTime = time.Now().Unix()
				}
				delete(a.turnMap, event.TurnID)
			}
			turnEvent, _ := event.To[jsonRpc.TurnEvent]()
			if turnEvent.Turn.Error != nil {
				completed <- errors.New(turnEvent.Turn.Error.Message)
			} else {
				completed <- nil
			}
			a.epMu.Unlock()
		}
	}
}

// codexEventParse 通知事件解析
func (a *Agent) codexEventParse(state *types.State, notification *codex.Event) {
	if notification.Type == codex.EventTypeRequest {
		a.codexRequestParse(state, notification)
		return
	}
	if state.EventHandler == nil {
		return
	}
	event := state.BuildEvent()
	event.Metadata = map[string]any{
		"codex_event_mehtod": notification.Method,
		"codex_thread_id":    notification.ThreadID,
		"codex_turn_id":      notification.TurnID,
	}
	msg := &types.Message{
		Role:             types.RoleAssistant,
		IsDetla:          false,
		ReasoningContent: "",
		Content:          "",
		ToolCalls:        nil,
		Media:            nil,
		Extra:            nil,
	}
	event.Message = msg
	switch codex.NotificationMethod(notification.Method) {
	// 1. 节点状态更新（捕获思考开始、工具调用开始）
	case codex.ItemStarted, codex.ItemCompleted:
		a.lifeEventParse(event, notification)
		if event.Type == types.EventMessageCompleted {
			// 存入state中
			state.AddNewMessage(event.Message)
		}
	case codex.ItemReasoningSummaryTextDelta, codex.ItemReasoningTextDelta, codex.ItemAgentMessageDelta:
		itemEvt, err := notification.To[jsonRpc.TextDeltaEvent]()
		if err != nil {
			fmt.Printf("item lifecycle event error: %v\n", err)
			return
		}
		msg.IsDetla = true
		event.EventID = itemEvt.ItemID
		event.Metadata["codex_item_type"] = itemEvt.DeltaType
		switch itemEvt.DeltaType {
		case jsonRpc.AgentMessage:
			event.Type = types.EventMessageDelta
			msg.Content = itemEvt.Delta
		case jsonRpc.Reasoning:
			event.Type = types.EventReasoningDelta
			msg.ReasoningContent = itemEvt.Delta
		}
	case codex.ThreadTokenUsageUpdated:
		itemEvt, err := notification.To[jsonRpc.TokenUsageNotification]()
		if err != nil {
			fmt.Printf("item lifecycle event error: %v\n", err)
			return
		}
		usage := tokenUsageOf(itemEvt.TokenUsage)
		if a.cfg != nil && a.cfg.SessionDB != nil {
			dErr := a.cfg.SessionDB.TurnTokenUsage(context.Background(), itemEvt.TurnID,
				dbTokenUsage(itemEvt.TokenUsage.Last), itemEvt.TokenUsage.ModelContextWindow)
			if dErr != nil {
				slog.Error("fail to save token usage", "error", dErr.Error())
			}
			dErr = a.cfg.SessionDB.ThreadTokenUsage(context.Background(), itemEvt.ThreadID,
				dbTokenUsage(itemEvt.TokenUsage.Total))
			if dErr != nil {
				slog.Error("fail to save thread token usage", "error", dErr.Error())
			}
		}
		// 计数器是给前端展示的，不落 Message：整帧只有 Usage 有意义。
		// 同一轮内可能多次推送，EventID 统一用 turnID，前端据此覆盖而不是追加。
		event.Type = types.EventTokenUsage
		event.EventID = itemEvt.TurnID
		event.Usage = usage
		if event.Metadata == nil {
			event.Metadata = map[string]any{}
		}
		event.Metadata["codex_token_total"] = usage.Total.TotalTokens
		event.Metadata["codex_token_last"] = usage.Last.TotalTokens
		slog.Debug("turn token 用量更新",
			"turnID", itemEvt.TurnID,
			"totalTokens", usage.Total.TotalTokens,
			"lastTokens", usage.Last.TotalTokens)
	}
	if event.Type != "" {
		state.EventHandler(event)
	}
}

// codexRequestParse 请求事件解析
func (a *Agent) codexRequestParse(state *types.State, request *codex.Event) {
	switch codex.ServerRequestMethod(request.Method) {
	case codex.ItemToolCall:
		itemEvt, err := request.To[jsonRpc.ToolCall]()
		if err != nil {
			slog.Error("request tool call error", "error", err.Error())
			return
		}
		slog.Info("request tool call", "request id", request.ID, "call_id", itemEvt.CallId)
		resp, err := a.dynamicToolDo(state, &itemEvt)
		if err != nil {
			slog.Error("request tool call result error", "error", err.Error())
			resp = &jsonRpc.ToolResponse{
				Success: false,
				Content: []jsonRpc.InputItem{
					{
						Type: "inputText",
						Text: "工具执行错误：" + err.Error(),
					},
				},
			}
		}
		err = a.client.Response(request.ID, resp)
		if err != nil {
			slog.Error("response tool request error", "error", err.Error())
		}
		slog.Info("request tool call result success", "request id", request.ID, "call_id", itemEvt.CallId, "result", resp)
	case codex.ItemCommandExecutionRequestApproval, codex.ExecCommandApproval,
		codex.ItemPermissionsRequestApproval, codex.ItemFileChangeRequestApproval:
		itemEvt, err := request.To[jsonRpc.BaseApprovalEvent]()
		if err != nil {
			slog.Error("request tool call error", "error", err.Error())
			return
		}
		event := state.BuildEvent()
		event.EventID = fmt.Sprintf("%v", request.ID)
		event.Type = types.EventApproval
		event.Message = &types.Message{
			Role:             types.RoleSystem,
			IsDetla:          false,
			ReasoningContent: "",
			Content:          itemEvt.Reason,
			Extra:            nil,
			ApprovalID:       request.ID,
		}
		if strings.Contains(strings.ToLower(request.Method), "command") {
			event.Message.Content = fmt.Sprintf("执行命令：%v,理由：%s", itemEvt.Command, itemEvt.Reason)
		}
		if event.Metadata == nil {
			event.Metadata = map[string]any{}
		}
		event.Metadata["approval_request"] = string(request.RawData)
		state.EventHandler(event)
	}
}

// lifeEventParse 开始结束通知统一处理。
//
// item 类型 -> 事件类型的映射只在下面这一个 switch 里做：填充载荷与决定事件类型必须来自
// 同一处判断。此处曾存在第二份映射表覆盖 event.Type，dynamicToolCall 不在那份表里，
// 结果事件类型被清空、在 codexEventParse 里被静默丢弃，前端永远收不到工具调用帧。
func (a *Agent) lifeEventParse(event *types.Event, notification *codex.Event) {
	itemEvt, err := notification.To[jsonRpc.ItemLifecycleEvent]()
	if err != nil {
		slog.Warn("item lifecycle event error", "error", err)
		return
	}
	event.EventID = itemEvt.Item.ID
	if event.Metadata == nil {
		event.Metadata = map[string]any{}
	}
	event.Metadata["codex_item_type"] = itemEvt.Item.Type
	msg := event.Message
	item := itemEvt.Item

	var eType types.EventType
	switch item.Type {
	case jsonRpc.Reasoning:
		msg.ReasoningContent = strings.Join(itemEvt.Item.ReasoningItemDetail.Summary, "\n")
		eType = types.EventReasoningStart
	case jsonRpc.AgentMessage:
		msg.Content = item.AgentMessageItemDetail.Text
		eType = types.EventMessageStart
	case jsonRpc.CommandExecution:
		msg.Role = types.RoleTool
		calls := types.ToolCall{
			ID:     item.ID,
			Type:   types.ToolTypeCommand,
			Name:   "command",
			Params: item.Command,
			Result: item.CommandExecutionItemDetail.AggregatedOutput,
			Extra:  make(map[string]any),
		}
		for i, v := range item.CommandActions {
			calls.Extra[fmt.Sprintf("action_%d", i)] = fmt.Sprintf("%s_%s", v.Type, v.Command)
		}
		msg.ToolCalls = append(msg.ToolCalls, calls)
		eType = types.EventToolStart
	case jsonRpc.WebSearch:
		msg.Role = types.RoleTool
		calls := types.ToolCall{
			ID:     item.ID,
			Type:   types.ToolTypeWebSearch,
			Name:   "web_search",
			Params: item.Query,
			Result: rawJSONText(item.WebSearchItemDetail.Results),
			Extra:  make(map[string]any),
		}
		for i, v := range item.WebSearchItemDetail.Action.Queries {
			calls.Extra[fmt.Sprintf("query_%d", i)] = v
		}
		msg.ToolCalls = append(msg.ToolCalls, calls)
		eType = types.EventToolStart
	case jsonRpc.DynamicToolCall:
		msg.Role = types.RoleTool
		calls := types.ToolCall{
			ID:     item.ID,
			Type:   types.ToolTypeFunction,
			Name:   item.ToolCallItemDetail.Tool,
			Params: toolParamsText(item.ToolCallItemDetail.Arguments),
			Result: item.ToolCallItemDetail.ResultText(),
			Extra:  make(map[string]any),
		}
		if item.ToolCallItemDetail.Namespace != "" {
			calls.Extra["namespace"] = item.ToolCallItemDetail.Namespace
		}
		// 报文里没有 callId，item.ID 本身就是 call_xxx，回给模型时用的就是它
		if item.Status != "" {
			calls.Extra["status"] = item.Status
		}
		if item.ToolCallItemDetail.Success != nil {
			calls.Extra["success"] = *item.ToolCallItemDetail.Success
		}
		if item.DurationMs > 0 {
			calls.Extra["durationMs"] = item.DurationMs
		}
		msg.ToolCalls = append(msg.ToolCalls, calls)
		eType = types.EventToolStart
	case jsonRpc.UserMessage:
		msg.Content = notification.RawData.String()
		eType = types.EventUserMessageStart
	default:
		slog.Warn(fmt.Sprintf("⚠️ life item [%s %s] => %s \n", notification.Method, item.Type, notification.RawData.String()))
		fmt.Printf("⚠️ [%s %s] => %s \n", notification.Method, item.Type, notification.RawData.String())
	}

	// item/completed 是该条目的结束帧：上面的 starting 类型统一收敛为对应的 completed 类型。
	if eType != "" && codex.NotificationMethod(notification.Method) == codex.ItemCompleted {
		switch eType {
		case types.EventReasoningStart:
			eType = types.EventReasoningCompleted
		case types.EventMessageStart:
			eType = types.EventMessageCompleted
		case types.EventToolStart:
			eType = types.EventToolCompleted
		case types.EventUserMessageStart:
			eType = types.EventUserMessageCompleted
		}
	}
	event.Type = eType
}

// rawJSONText 把 json.RawMessage 转成前端可读文本。
// item/started 阶段结果往往还是 null，直接 String() 会让前端显示字面量 "null"。
func rawJSONText(raw json.RawMessage) string {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return ""
	}
	// 结果若不是字符串而是结构化 JSON，保留原始 JSON 文本，交给前端按需解析
	if unquoted, err := strconv.Unquote(text); err == nil {
		return unquoted
	}
	return text
}

// toolParamsText 序列化工具入参，空参数给空串而不是 "null"。
func toolParamsText(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	params, err := json.Marshal(args)
	if err != nil {
		return fmt.Sprint(args)
	}
	return string(params)
}
