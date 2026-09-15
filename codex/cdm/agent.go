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
	"uuid"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/codex/proc"
	"github.com/kinwyb/buibuiCodex/codex/proc/docker"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
)

type codexTurn struct {
	startTime      int64         //启动时间
	lastUpdateTime int64         //最后消息时间
	threadID       string        //线程ID
	thread         *codex.Thread //启动的线程
	turn           *codex.Turn   //启动的会话
	state          *types.State  //消息状态
}

type Agent struct {
	cfg     *config.AgentConfig
	process proc.Process
	client  *codex.Client
	epMu    sync.Mutex
	eps     map[string]*codexTurn
	turnMap map[string]string //turnID => sessionID
	isInit  bool
	initMu  sync.Mutex
	tools   []*toolGroup
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

func (a *Agent) Init() error {
	if a.cfg.DockerImage == "" {
		return errors.New("docker image is required")
	}
	return a.initProcess()
}

// RegisterTool 注册工具函数
func (a *Agent) RegisterTool(namespace string, desc string, tool ...types.Tool) {
	var group *toolGroup
	for _, t := range a.tools {
		if t.nameSpace == namespace {
			group = t
			group.spaceDesc = desc
		}
	}
	if group == nil {
		group = &toolGroup{
			nameSpace: namespace,
			spaceDesc: desc,
		}
		a.tools = append(a.tools, group)
	}
	group.tools = append(group.tools, tool...)
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
		ImageName:     a.cfg.DockerImage,
		WorkSpace:     filepath.Join(a.cfg.WorkSpace, "workspace"),
		TmpSpace:      filepath.Join(a.cfg.WorkSpace, "tmp"),
		CodexHome:     filepath.Join(a.cfg.WorkSpace, "root"),
		APIBaseURL:    a.cfg.Provider.APIBaseURL,
		SkillDir:      a.cfg.SkillDir,
		//APIKey:        a.cfg.Provider.APIKey,
		Env: []string{
			"OPENAI_API_KEY_" + a.cfg.ProviderName + "=" + a.cfg.Provider.APIKey,
			"RUST_LOG=debug",
		},
		MCP:            make(map[string]docker.MCPServerConfig),
		ModelProviders: make(map[string]docker.ModelProvider),
	}
	startParam.ModelProviders[a.cfg.ProviderName] = docker.ModelProvider{
		BaseURL:            a.cfg.Provider.APIBaseURL,
		EnvKey:             "OPENAI_API_KEY_" + a.cfg.ProviderName,
		Name:               a.cfg.ProviderName,
		SupportsWebsockets: new(false),
	}

	for k, v := range a.cfg.MCPProvider {
		mcpServer := docker.MCPServerConfig{
			Args:    v.Args,
			Env:     v.Env,
			Headers: v.Headers,
		}
		if v.Command != "" {
			mcpServer.Command = new(v.Command)
		}
		if v.URL != "" {
			mcpServer.URL = new(v.URL)
		}
		if v.Auth != "" {
			mcpServer.Headers = make(map[string]string)
			mcpServer.Headers["Authorization"] = "Bearer " + v.Auth
		}
		startParam.MCP[k] = mcpServer
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
	client.RegisterDefaultEventHandler(a.allEventHandler)
	a.client = client
	a.isInit = true
	return nil
}

func (a *Agent) getThread(ctx context.Context, sessionID string, cmd *command) (*codex.Thread, error) {
	a.epMu.Lock()
	defer a.epMu.Unlock()
	thread := a.client.Thread()
	thread.SubUnknowTurnEvent(a.unknowTurnEventHandler)
	waitResume := true
	turn, ok := a.eps[sessionID]
	if cmd.newThread || !ok {
		// 查询数据库中使用过的thread
		dbThread := a.cfg.SessionDB.LastThread(ctx, sessionID, a.AgentID())
		if !cmd.newThread && dbThread != nil {
			turn = &codexTurn{
				startTime:      dbThread.CreateTime.Unix(),
				threadID:       dbThread.ThreadID,
				lastUpdateTime: dbThread.LastUpdate.Unix(),
			}
		} else {
			// 没有有效的thread，创建新的thread
			waitResume = false
			nTurn, err := a.newThread(ctx, thread, sessionID)
			if err != nil {
				return nil, err
			}
			turn = nTurn
		}
	}
	if waitResume {
		err := thread.Resume(ctx, turn.threadID)
		if err != nil {
			slog.Error(err.Error())
			return nil, err
		}
		turn.thread = thread
		turn.lastUpdateTime = time.Now().Unix()
	}

	// 数据库保存新建成功的thread
	dbErr := a.cfg.SessionDB.ThreadSave(ctx, &db.SessionThread{
		ThreadID:   turn.threadID,
		SessionID:  sessionID,
		Agent:      a.AgentID(),
		CreateTime: time.Now(),
		LastUpdate: time.Now(),
	})
	if dbErr != nil {
		slog.Error("fail to save db thread ", "error", dbErr.Error())
	}
	return thread, nil
}

func (a *Agent) newThread(ctx context.Context, thread *codex.Thread, sessionID string) (*codexTurn, error) {
	cfg := &jsonRpc.ThreadConfig{
		Model:          a.cfg.Model,
		OpenAIBaseURL:  a.cfg.Provider.APIBaseURL,
		OpenAIAPIKey:   a.cfg.Provider.APIKey,
		ModelProviders: make(map[string]jsonRpc.ModelProviderInfo),
	}
	cfg.ModelProviders[a.cfg.ProviderName] = jsonRpc.ModelProviderInfo{
		BaseURL:            a.cfg.Provider.APIBaseURL,
		EnvKey:             "OPENAI_API_KEY_" + a.cfg.ProviderName,
		Name:               a.cfg.ProviderName,
		SupportsWebsockets: new(false),
	}
	slog.Debug("OPENAI_BASE_URL=" + a.cfg.Provider.APIBaseURL)
	workspace := filepath.Join(a.cfg.WorkSpace, "workspace")
	opts := []codex.ThreadStartOption{codex.ThreadStartWithConfig(cfg),
		codex.ThreadStartWithSendbox(jsonRpc.SandboxModeDangerFullAccess), codex.ThreadStartWithModelProvider(a.cfg.ProviderName)}
	dynamicTool := a.dynamicTools()
	if len(dynamicTool) > 0 {
		opts = append(opts, codex.ThreadStartWithDynamicTool(dynamicTool...))
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
	turn := &codexTurn{
		threadID:       threadID,
		startTime:      time.Now().Unix(),
		lastUpdateTime: time.Now().Unix(),
		thread:         thread,
	}
	a.eps[sessionID] = turn

	return turn, nil
}

// Prompt 执行一个请求
func (a *Agent) Prompt(ctx context.Context, state *types.State) error {
	// 初始化进程
	err := a.initProcess()
	if err != nil {
		return err
	}
	cmd := commandParse(state.Input.Content)
	state.Input.Content = cmd.msg
	thread, err := a.getThread(ctx, state.SessionID, cmd)
	if err != nil {
		return err
	}
	turn, err := thread.Turn()
	if err != nil {
		err = fmt.Errorf("thread build turn fail: %w", err)
		slog.Error(err.Error())
		return err
	}
	var opts []codex.TurnStartOptions
	opts = append(opts, codex.TurnStartWithSandboxPolicy(jsonRpc.SandboxPolicy{
		Type:          jsonRpc.SandboxPolicyTypeDangerFullAccess,
		NetworkAccess: true,
	}), codex.TurnStartWithApprovalPolicy(jsonRpc.Never))
	err = turn.Start(ctx, a.buildInput(turn, state.Input), opts...)
	if err != nil {
		err = fmt.Errorf("turn start fail: %w", err)
		slog.Error(err.Error())
		return err
	}
	a.epMu.Lock()
	if ct, ok := a.eps[state.SessionID]; ok {
		ct.turn = turn
		ct.state = state
	}
	a.epMu.Unlock()
	// 保存turn
	dbErr := a.cfg.SessionDB.TurnSave(ctx, &db.SessionTurn{
		ReqID:     state.ReqID,
		TurnID:    turn.TurnID(),
		ThreadID:  turn.ThreadID(),
		SessionID: state.SessionID,
		Agent:     a.AgentID(),
	})
	if dbErr != nil {
		slog.Error("fail to save db turn ", "error", dbErr.Error())
	}
	completed := make(chan error, 1)
	turn.RegisterEventHandler(func(event *codex.Event) {
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
	})
	err = <-completed
	close(completed)
	return err
}

// Approve 提交审批结果
func (a *Agent) Approve(reqID any, approve jsonRpc.ApprovalDecision) error {
	approveResult := &jsonRpc.ApprovalResult{
		Decision: approve,
	}
	err := a.client.Response(reqID, approveResult)
	if err != nil {
		slog.Error("response approve request error", "error", err.Error())
	}
	return err
}

func (a *Agent) allEventHandler(event *codex.Event) {
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

func (a *Agent) unknowTurnEventHandler(event *codex.Event) {
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

func (a *Agent) buildInput(turn *codex.Turn, message *types.InputMessage) []jsonRpc.InputItem {
	var result []jsonRpc.InputItem
	if message.Content != "" {
		result = turn.BuildInputWithText(message.Content, result...)
	}
	for _, item := range message.Media {
		if item.Type == types.MediaTypeImage {
			if item.URL != "" {
				if strings.HasPrefix(item.URL, "http") {
					result = turn.BuildInputWithImage(item.URL, result...)
				} else {
					result = turn.BuildInputWithLocalImage(item.URL, result...)
				}
			} else if item.Base64 != "" {
				result = turn.BuildInputWithImage("data:"+item.MimeType+";base64,"+item.Base64, result...)
			}
		} else if item.Type == types.MediaTypeFile {
			filename := item.Metadata["filename"].(string)
			if filename == "" {
				filename = uuid.NewV4().String()
			}
			if item.URL != "" {
				result = turn.BuildInputWithText("文件["+filename+"]地址:"+item.URL, result...)
			}
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

func (a *Agent) Cancel(ctx context.Context, sessionID string) error {
	a.epMu.Lock()
	ct, ok := a.eps[sessionID]
	a.epMu.Unlock()
	if ok {
		if ct.turn != nil {
			ct.turn.Interrupt(ctx)
		}
	}
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
