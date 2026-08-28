package cdm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/codex/proc"
	"github.com/kinwyb/buibuiCodex/codex/proc/docker"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/types"
)

type codexTurn struct {
	startTime      int64  //启动时间
	threadID       string //线程ID
	turnID         string //会话ID
	lastUpdateTime int64  //最后消息时间
}

type Agent struct {
	cfg               *config.AgentConfig
	process           proc.Process
	client            *codex.Client
	epMu              sync.Mutex
	eps               map[string]*codexTurn
	turnMap           map[string]string //turnID => sessionID
	isInit            bool
	initMu            sync.Mutex
	toolNamespace     string
	toolNamespaceDesc string
	tools             []types.Tool
}

func NewAgent(cfg *config.AgentConfig) *Agent {
	return &Agent{
		cfg: cfg,
		eps: make(map[string]*codexTurn),
	}
}

func (a *Agent) AgentID() string {
	return a.cfg.Name
}

// RegisterTool 注册工具函数
func (a *Agent) RegisterTool(namespace string, desc string, tool ...types.Tool) {
	a.toolNamespace = namespace
	a.toolNamespaceDesc = desc
	a.tools = append(a.tools, tool...)
}

func (a *Agent) dynamicTools() *jsonRpc.DynamicTool {
	if len(a.tools) == 0 {
		return nil
	}
	ret := &jsonRpc.DynamicTool{
		Type:        "namespace",
		Name:        a.toolNamespace,
		Description: a.toolNamespaceDesc,
		Tools:       nil,
	}
	for _, tool := range a.tools {
		dt := jsonRpc.ToolDescription{
			Type:         "function",
			Name:         tool.Name(),
			Description:  tool.Description(),
			DeferLoading: false,
			InputSchema:  tool.Parameters(),
		}
		ret.Tools = append(ret.Tools, dt)
	}
	return ret
}

func (a *Agent) initProcess() error {
	a.initMu.Lock()
	defer a.initMu.Unlock()
	if a.isInit {
		return nil
	}
	// docker容器运行方式
	startParam := docker.StartParam{
		ContainerName: "codex_" + a.cfg.Name,
		ImageName:     "codex_run:v2",
		WorkSpace:     filepath.Join(a.cfg.WorkSpace, "workspace"),
		CodexHome:     filepath.Join(a.cfg.WorkSpace, "root"),
		Env: []string{
			"OPENAI_BASE_URL=" + a.cfg.Provider.APIBaseURL,
			"OPENAI_API_KEY=" + a.cfg.Provider.APIKey,
		},
	}

	pm, err := proc.NewSafeProc(startParam)
	if err != nil {
		err = fmt.Errorf("proc new fail: %w", err)
		slog.Error(err.Error())
		return err
	}

	// 设置 10 秒启动超时
	startCtx, startCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer startCancel()
	startResult, err := pm.Start(startCtx)
	if err != nil {
		return fmt.Errorf("codex 进程启动失败: %w", err)
	}
	a.process = pm
	// 2. 初始化 Client 并建立 WebSocket 连接
	wsURL := startResult.WsUrl
	client, err := codex.NewClient(wsURL, startResult.Token)
	if err != nil {
		return fmt.Errorf("连接 app-server 失败: %v", err)
	}
	a.client = client
	a.isInit = true
	return nil
}

func (a *Agent) getThread(ctx context.Context, sessionID string) (*codex.Thread, error) {
	a.epMu.Lock()
	defer a.epMu.Unlock()
	thread := a.client.Thread()
	turn, ok := a.eps[sessionID]
	if !ok {
		cfg := &jsonRpc.ThreadConfig{
			Model:         a.cfg.Model,
			OpenAIBaseURL: a.cfg.Provider.APIBaseURL,
			OpenAIAPIKey:  a.cfg.Provider.APIKey,
			////MCPServers: map[string]jsonRpc.MCPServer{
			//	"boda": {
			//		URL: "http://localhost:9090/mcp",
			//	},
			//},
		}
		workspace := filepath.Join(a.cfg.WorkSpace, "workspace")
		opts := []codex.ThreadStartOption{codex.ThreadStartWithConfig(cfg),
			codex.ThreadStartWithSendbox(jsonRpc.SandboxModeDangerFullAccess)}
		dynamicTool := a.dynamicTools()
		if dynamicTool != nil {
			opts = append(opts, codex.ThreadStartWithDynamicTool(dynamicTool))
		}
		if a.cfg.Description != "" {
			opts = append(opts, codex.ThreadStartWithInstructions(a.cfg.Description))
		}
		startThreadParams := codex.NewThreadStartParam(workspace, opts...)
		threadID, err := thread.Start(ctx, *startThreadParams)
		if err != nil {
			err = fmt.Errorf("threadStart fail: %w", err)
			slog.Error(err.Error())
			return nil, err
		}
		slog.Debug(fmt.Sprintf("thread started : %s", threadID))
		turn = &codexTurn{
			threadID:       threadID,
			startTime:      time.Now().Unix(),
			lastUpdateTime: time.Now().Unix(),
		}
		a.eps[sessionID] = turn
	} else {
		err := thread.Resume(ctx, turn.threadID)
		if err != nil {
			slog.Error(err.Error())
			return nil, err
		}
		turn.lastUpdateTime = time.Now().Unix()
	}
	return thread, nil
}

// Prompt 执行一个请求
func (a *Agent) Prompt(ctx context.Context, state *types.State) error {
	// 初始化进程
	err := a.initProcess()
	if err != nil {
		return err
	}
	thread, err := a.getThread(ctx, state.SessionID)
	turn, err := thread.Turn()
	if err != nil {
		err = fmt.Errorf("thread build turn fail: %w", err)
		slog.Error(err.Error())
		return err
	}
	err = turn.Start(ctx, a.buildInput(turn, state.Input))
	if err != nil {
		err = fmt.Errorf("turn start fail: %w", err)
		slog.Error(err.Error())
		return err
	}
	a.epMu.Lock()
	if ct, ok := a.eps[state.SessionID]; ok {
		ct.turnID = turn.TurnID()
	}
	a.epMu.Unlock()
	turn.RegisterEventHandler(func(event *codex.Event) {
		a.codexEventParse(state, event)
		if event.Method == string(codex.TurnCompleted) {
			a.epMu.Lock()
			if sessionID, ok := a.turnMap[event.TurnID]; ok {
				if ct, ok2 := a.eps[sessionID]; ok2 {
					ct.turnID = ""
					ct.lastUpdateTime = time.Now().Unix()
				}
				delete(a.turnMap, event.TurnID)
			}
			a.epMu.Unlock()
		}
	})
	return nil
}

func (a *Agent) buildInput(turn *codex.Turn, message *types.InputMessage) []jsonRpc.InputItem {
	var result []jsonRpc.InputItem
	if message.Content != "" {
		result = turn.BuildInputWithText(message.Content, result...)
	}
	for _, item := range message.Media {
		if item.Type == "image" {
			result = turn.BuildInputWithImage(item.Base64, result...)
		}
	}
	return result
}

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
	switch notification.Method {
	// 1. 节点状态更新（捕获思考开始、工具调用开始）
	case string(codex.ItemStarted), string(codex.ItemCompleted):
		a.lifeEventParse(event, notification)
	case string(codex.ItemReasoningSummaryTextDelta), string(codex.ItemReasoningTextDelta), string(codex.ItemAgentMessageDelta):
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
	case string(codex.ThreadTokenUsageUpdated):
		itemEvt, err := notification.To[jsonRpc.TokenUsageNotification]()
		if err != nil {
			fmt.Printf("item lifecycle event error: %v\n", err)
			return
		}
		fmt.Printf("\n\n🎉 [Turn token消耗统计] Total Tokens: %d,Last Tokens: %d\n",
			itemEvt.TokenUsage.Total.TotalTokens, itemEvt.TokenUsage.Last.TotalTokens)
	}
	if event.Type != "" {
		state.EventHandler(event)
	}
}

func (a *Agent) codexRequestParse(state *types.State, request *codex.Event) {
	switch request.Method {
	case string(codex.ItemToolCall):
		itemEvt, err := request.To[jsonRpc.ToolCall]()
		if err != nil {
			slog.Error("request tool call error", "error", err.Error())
			return
		}
		resp, err := a.dynamicToolDo(&itemEvt)
		if err != nil {
			slog.Error("request tool call result error", "error", err.Error())
		}
		err = a.client.Response(request.ID, resp, err)
		if err != nil {
			slog.Error("response tool request error", "error", err.Error())
		}
	}
}

// lifeEventParse 开始结束通知统一处理
func (a *Agent) lifeEventParse(event *types.Event, notification *codex.Event) {
	itemEvt, err := notification.To[jsonRpc.ItemLifecycleEvent]()
	if err != nil {
		fmt.Printf("item lifecycle event error: %v\n", err)
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
	default:
		fmt.Printf("⚠️ [%s %s] => %s \n", notification.Method, item.Type, notification.RawData.String())
	}
	event.Type = func(itemType jsonRpc.EventItemType, method codex.NotificationMethod) types.EventType {
		var eType types.EventType
		switch item.Type {
		case jsonRpc.Reasoning:
			eType = types.EventReasoningStart
		case jsonRpc.AgentMessage:
			eType = types.EventMessageStart
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
			}
		}
		return eType
	}(item.Type, codex.NotificationMethod(notification.Method))
}

func (a *Agent) dynamicToolDo(call *jsonRpc.ToolCall) (*jsonRpc.ToolResponse, error) {
	slog.Info("dynamic tool do :" + call.Tool)
	funcname := strings.TrimLeft(call.Tool, a.toolNamespace+"__")
	for _, tool := range a.tools {
		if tool.Name() == funcname {
			result, err := tool.Execute(context.Background(), call.Arguments)
			if err != nil {
				return nil, err
			}
			return &jsonRpc.ToolResponse{
				Success: true,
				Content: []jsonRpc.InputItem{
					{
						Type: "inputText",
						Text: result,
					},
				},
			}, nil
		}
	}
	return &jsonRpc.ToolResponse{
		Success: false,
	}, errors.New("tool not found")
}

func (a *Agent) Cancel(ctx context.Context, sessionID string) error {
	return nil
}

// todo: 这里要控制等待所有任务完成，在停止codex进程
func (a *Agent) Stop() error {
	a.initMu.Lock()
	defer a.initMu.Unlock()
	if a.client != nil {
		a.client.Close()
	}
	if a.process != nil {
		a.process.Stop()
	}
	a.isInit = false
	clear(a.eps)
	return nil
}
