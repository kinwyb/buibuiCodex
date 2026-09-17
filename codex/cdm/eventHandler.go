package cdm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"uuid"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
)

// clientEvent 这里可以获取到所有客户的事件,注意这里事件和thread，turn等事件是重复分发的,注意重复处理问题
func (a *Agent) clientEvent(event *codex.Event) {
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
		if a.cfg.SessionDB != nil {
			dErr := a.cfg.SessionDB.TurnTokenUsage(context.Background(), itemEvt.TurnID, db.TokenUsage{
				TotalTokens:           itemEvt.TokenUsage.Last.TotalTokens,
				InputTokens:           itemEvt.TokenUsage.Last.InputTokens,
				CachedInputTokens:     itemEvt.TokenUsage.Last.CachedInputTokens,
				CacheWriteInputTokens: itemEvt.TokenUsage.Last.CacheWriteInputTokens,
				OutputTokens:          itemEvt.TokenUsage.Last.OutputTokens,
				ReasoningOutputTokens: itemEvt.TokenUsage.Last.ReasoningOutputTokens,
			})
			if dErr != nil {
				slog.Error("fail to save token usage", "error", dErr.Error())
			}
			dErr = a.cfg.SessionDB.ThreadTokenUsage(context.Background(), itemEvt.ThreadID, db.TokenUsage{
				TotalTokens:           itemEvt.TokenUsage.Total.TotalTokens,
				InputTokens:           itemEvt.TokenUsage.Total.InputTokens,
				CachedInputTokens:     itemEvt.TokenUsage.Total.CachedInputTokens,
				CacheWriteInputTokens: itemEvt.TokenUsage.Total.CacheWriteInputTokens,
				OutputTokens:          itemEvt.TokenUsage.Total.OutputTokens,
				ReasoningOutputTokens: itemEvt.TokenUsage.Total.ReasoningOutputTokens,
			})
			if dErr != nil {
				slog.Error("fail to save thread token usage", "error", dErr.Error())
			}
		}
		fmt.Printf("\n\n🎉 [Turn token消耗统计] Total Tokens: %d,Last Tokens: %d\n",
			itemEvt.TokenUsage.Total.TotalTokens, itemEvt.TokenUsage.Last.TotalTokens)
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

// lifeEventParse 开始结束通知统一处理
func (a *Agent) lifeEventParse(event *types.Event, notification *codex.Event) {
	itemEvt, err := notification.To[jsonRpc.ItemLifecycleEvent]()
	if err != nil {
		slog.Warn("item lifecycle event error", "error", err)
		return
	}
	event.EventID = itemEvt.Item.ID
	event.Metadata["codex_item_type"] = itemEvt.Item.Type
	msg := event.Message
	item := itemEvt.Item
	switch item.Type {
	case jsonRpc.Reasoning:
		msg.ReasoningContent = strings.Join(itemEvt.Item.ReasoningItemDetail.Summary, "\n")
		event.Type = types.EventReasoningStart
	case jsonRpc.AgentMessage:
		msg.Content = item.AgentMessageItemDetail.Text
		event.Type = types.EventMessageStart
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
		event.Type = types.EventToolStart
	case jsonRpc.WebSearch:
		msg.Role = types.RoleTool
		calls := types.ToolCall{
			ID:     item.ID,
			Type:   types.ToolTypeWebSearch,
			Name:   "web_search",
			Params: item.Query,
			Result: item.WebSearchItemDetail.Results.String(),
			Extra:  make(map[string]any),
		}
		for i, v := range item.WebSearchItemDetail.Action.Queries {
			calls.Extra[fmt.Sprintf("query_%d", i)] = v
		}
		msg.ToolCalls = append(msg.ToolCalls, calls)
		event.Type = types.EventToolStart
	case jsonRpc.DynamicToolCall:
		msg.Role = types.RoleTool
		params, _ := json.Marshal(item.ToolCallItemDetail.Arguments)
		calls := types.ToolCall{
			ID:     item.ID,
			Type:   types.ToolTypeFunction,
			Name:   itemEvt.Item.Tool,
			Params: string(params),
			Result: item.ToolCallItemDetail.Result.String(),
			Extra:  make(map[string]any),
		}
		msg.ToolCalls = append(msg.ToolCalls, calls)
		event.Type = types.EventToolStart
	case jsonRpc.UserMessage:
		msg.Content = notification.RawData.String()
		event.Type = types.EventUserMessageStart
	default:
		slog.Warn(fmt.Sprintf("⚠️ life item [%s %s] => %s \n", notification.Method, item.Type, notification.RawData.String()))
		fmt.Printf("⚠️ [%s %s] => %s \n", notification.Method, item.Type, notification.RawData.String())
	}
	event.Type = func(itemType jsonRpc.EventItemType, method codex.NotificationMethod) types.EventType {
		var eType types.EventType
		switch item.Type {
		case jsonRpc.Reasoning:
			eType = types.EventReasoningStart
		case jsonRpc.AgentMessage:
			eType = types.EventMessageStart
		case jsonRpc.UserMessage:
			eType = types.EventUserMessageStart
		case jsonRpc.CommandExecution, jsonRpc.WebSearch:
			eType = types.EventToolStart
		}
		if eType != "" && method == codex.ItemCompleted {
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
		return eType
	}(item.Type, codex.NotificationMethod(notification.Method))
}
