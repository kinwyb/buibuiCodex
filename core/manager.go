package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/kinwyb/buibuiCodex/codex/cdm"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
)

// Manager 管理多个 Agent 实例
type Manager struct {
	agents       map[string]types.Agent // agentID -> Agent
	defaultAgent types.Agent            // 默认 Agent
	bus          *bus.MessageBus
	workspace    string // 工作区根目录
	mu           sync.RWMutex
	cancelCh     chan string // sessionKey 取消信号 channel
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	sequence     *sessionSequence
	dataStorage  *db.Data
}

// NewManager 创建 Agent 管理器
func NewManager(msgBus *bus.MessageBus) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	ret := &Manager{
		agents:   make(map[string]types.Agent),
		bus:      msgBus,
		ctx:      ctx,
		cancel:   cancel,
		cancelCh: make(chan string, 10),
	}
	return ret
}

// MessageBus 消息总线
func (m *Manager) MessageBus() *bus.MessageBus {
	return m.bus
}

// log 发布日志到总线
func (m *Manager) log(ctx context.Context, level, source, message string) {
	if m.bus == nil {
		return
	}
	event := &bus.Log{
		Level:     level,
		Source:    source,
		Message:   message,
		Timestamp: time.Now(),
	}
	_ = m.bus.PublishLog(ctx, event)
}

// RegisterAgent 注册 Agent
func (m *Manager) RegisterAgent(agentID string, agent types.Agent, isDefault bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.agents[agentID] = agent
	if isDefault {
		m.defaultAgent = agent
	}
}

// InitFromConfig 根据配置自动注册所有 agent
// 处理子 agent 依赖关系，确保子 agent 先于父 agent 创建
func (m *Manager) InitFromConfig(ctx context.Context, cfg *config.ManagerConfig) error {
	if cfg.Workspace == "" {
		return errors.New("workspace is required")
	}
	if cfg == nil || len(cfg.Agents) == 0 {
		return errors.New("no agents in config")
	}
	sqlite, err := db.NewSQLiteStorage(filepath.Join(cfg.Workspace, "buibui.db"))
	if err != nil {
		return err
	}
	m.dataStorage = db.NewData(sqlite)

	// 解析所有 agent 配置
	agentNames := make([]string, 0, len(cfg.Agents))
	resolvedConfigs := make(map[string]*config.AgentConfig)
	for _, agentCfg := range cfg.Agents {
		if slices.Contains(agentNames, agentCfg.Name) {
			return errors.New("agent name [" + agentCfg.Name + "] is duplicated")
		}
		agentNames = append(agentNames, agentCfg.Name)
		resolvedConfigs[agentCfg.Name] = agentCfg
	}

	// 按顺序创建和注册 agent
	createdAgents := make(map[string]types.Agent)
	defaultAgentName := cfg.Agent
	if defaultAgentName == "" {
		defaultAgentName = agentNames[0]
		cfg.Agent = defaultAgentName
	}

	tos := GetTools()

	for _, name := range agentNames {
		resolved, err := cfg.ResolveAgentConfig(name)

		if err != nil {
			return fmt.Errorf("failed to resolve agent config: %w", err)
		}

		// 创建工作区文件夹
		workspace := resolved.WorkSpace
		if _, err = os.Stat(workspace); os.IsNotExist(err) {
			_ = os.MkdirAll(workspace, 0755)
		}
		workspacetmp := filepath.Join(workspace, "tmp")
		if _, err = os.Stat(workspacetmp); os.IsNotExist(err) {
			_ = os.MkdirAll(workspacetmp, 0755)
		}
		resolved.SessionDB = m.dataStorage.Session()

		// 创建 Agent
		ag := cdm.NewAgent(resolved)

		ag.RegisterTool("base", "基本工具允许获取运行所需的基础信息", tos...)

		createdAgents[name] = ag

		// 注册到 Manager
		isDefault := name == defaultAgentName
		m.RegisterAgent(name, ag, isDefault)

		m.log(ctx, bus.LogLevelInfo, "manager", fmt.Sprintf("Agent '%s' registered (default=%v)", name, isDefault))
	}

	return nil
}

// GetAgent 获取 Agent
func (m *Manager) GetAgent(agentID string) (types.Agent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, ok := m.agents[agentID]
	return agent, ok
}

// GetDefaultAgent 获取默认 Agent
func (m *Manager) GetDefaultAgent() types.Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultAgent
}

// ListAgents 列出所有 Agent ID
func (m *Manager) ListAgents() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.agents))
	for id := range m.agents {
		ids = append(ids, id)
	}
	return ids
}

// Start 启动消息处理器
func (m *Manager) Start(ctx context.Context) error {
	// 使用传入的 ctx 作为父 context，这样 Gateway 取消时会传递到这里
	m.ctx, m.cancel = context.WithCancel(ctx)

	// 启动消息处理器
	inBoundSub, err := m.bus.SubscribeInput()
	if err != nil {
		slog.Warn("Failed to consume inbound messages", "error", err)
		return err
	}
	m.sequence = newSessionSequence(m.ctx, inBoundSub)
	// 启动消息处理队列
	go m.sequence.Start(m)
	// 启动取消信号处理器
	m.wg.Add(1)
	go m.processCancel()

	return nil
}

// Stop 停止所有 Agent
func (m *Manager) Stop() error {

	if m.dataStorage != nil {
		m.dataStorage.Close()
	}

	// 先取消 context，让 processMessages 和 scheduler 退出
	if m.cancel != nil {
		m.cancel()
	}

	if m.sequence != nil {
		m.sequence.Stop()
	}

	// 等待所有 goroutine 退出（processMessages、processCancel、processPendingScheduler、业务 goroutine）
	m.wg.Wait()

	// 然后停止所有 agents
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, agent := range m.agents {
		if err := agent.Stop(); err != nil {
			m.log(context.Background(), bus.LogLevelError, "manager", fmt.Sprintf("Failed to stop agent %s: %v", id, err))
		}
	}
	return nil
}

// processCancel 处理取消信号（独立 goroutine，不被 Prompt() 阻塞）
func (m *Manager) processCancel() {
	defer m.wg.Done()
	for {
		select {
		case <-m.ctx.Done():
			return
		case sessionKey := <-m.cancelCh:
			m.CancelSession(sessionKey)
		}
	}
}

// Cancel 向取消 channel 发送取消信号，非阻塞
// 供 channel 等外部调用，所有来源的停止请求都走这个入口
func (m *Manager) Cancel(sessionKey string) {
	select {
	case m.cancelCh <- sessionKey:
	default:
		slog.Warn("cancel channel full, dropping cancel request", "sessionKey", sessionKey)
	}
}

// CancelSession 取消指定 sessionKey 对应的运行中 agent 会话
func (m *Manager) CancelSession(sessionKey string) bool {
	// 要从sequence中获取到运行的状态，进行取消
	slog.Info("取消 agent 会话", "sessionKey", sessionKey)
	return true
}

func (m *Manager) messageHandler(ctx context.Context, state *types.State) {
	// 保存session信息
	dbErr := m.dataStorage.Session().SessionSave(ctx, &db.Session{
		SessionID: state.SessionID,
		Channel:   state.Input.Channel,
		AccountID: state.Input.AccountID,
		UserID:    state.Input.SenderID,
		ChatID:    state.Input.ChatID,
	})
	if dbErr != nil {
		slog.Error("failed to save session", "error", dbErr)
	}
	msg := state.Input

	// 识别定时任务消息
	//if msg.Metadata != nil && msg.Metadata[meta.KeyFormScheduledTask] == true {
	//	slog.Info("Scheduled task inbound message routed",
	//		"task_id", msg.Metadata[meta.KeyScheduledID],
	//		"task_name", msg.Metadata[meta.KeyScheduledName],
	//		"channel", msg.Channel,
	//		"chat_id", msg.ChatID,
	//		"sender_id", msg.SenderID,
	//	)
	//}

	// 仅在 agent 选择阶段持有读锁，避免在 handleInboundMessage（含 agent.Prompt）期间阻塞 RegisterAgent
	agent, skipAgent, err := m.resolveAgent(ctx, msg)
	if err != nil {
		slog.Error("Failed to resolve agent", "error", err)
		return
	}
	if skipAgent {
		return
	}
	// 处理消息
	m.agentDo(ctx, state, agent)
}

func (m *Manager) busyNotification(ctx context.Context, msg *types.InputMessage, isStore bool) {
	outbound := &types.Event{
		EventID:   "busy_" + msg.ID,
		SessionID: "",
		ReqID:     msg.ID,
		Type:      types.EventError,
		Channel:   msg.Channel,
		AccountID: msg.AccountID,
		SenderID:  msg.SenderID,
		ChatID:    msg.ChatID,
		Message: &types.Message{
			Role:             types.RoleSystem,
			IsDetla:          false,
			ReasoningContent: "",
			Content:          "请求繁忙,请稍后再试",
			ToolCalls:        nil,
			Media:            nil,
			Extra:            nil,
		},
		Metadata:  nil,
		Timestamp: time.Now(),
	}
	if isStore {
		outbound.Message.Content = "上一任务处理中,本次请求已进入队列等待后续执行..."
	}
	_ = m.bus.PublishEvent(ctx, outbound)
}

// resolveAgent 在持有读锁期间选择目标 Agent，返回 (agent, skipAgent, error)
// 锁仅在选择阶段持有，不会延续到后续的 handleInboundMessage
func (m *Manager) resolveAgent(ctx context.Context, msg *types.InputMessage) (agent types.Agent, skipAgent bool, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 选择 Agent
	agent = m.defaultAgent
	if msg.AgentName != "" {
		if ag, exists := m.agents[msg.AgentName]; exists {
			agent = ag
		} else {
			m.log(ctx, bus.LogLevelWarn, "manager", fmt.Sprintf("Target agent '%s' not found, using default", msg.AgentName))
		}
	}

	if agent == nil {
		return nil, false, fmt.Errorf("no agent available")
	}
	// 修正agent名称
	msg.AgentName = agent.AgentID()

	return agent, false, nil
}

func (m *Manager) handleInboundContext(ctx context.Context, msg *types.InputMessage) context.Context {
	// 设置 context values，工具可通过 ctx.Value() 获取
	//ctx = context.WithValue(ctx, meta.KeyContextChannel, msg.Channel)
	//ctx = context.WithValue(ctx, meta.KeyContextChatID, msg.ChatID)
	//ctx = context.WithValue(ctx, meta.KeyContextAccountID, msg.AccountID)
	//ctx = context.WithValue(ctx, meta.KeyContextSenderID, msg.SenderID)
	//ctx = context.WithValue(ctx, agcore.MessageExtraReqIDKey, msg.ID)
	//ctx = context.WithValue(ctx, meta.KeyContextAgentName, msg.Metadata[meta.KeyRouteAgent])
	return ctx
}

// agent执行
func (m *Manager) agentDo(ctx context.Context, state *types.State, agent types.Agent) error {

	// 保存请求信息
	quest := state.Input.Content
	if quest == "" {
		questV, _ := json.Marshal(state.Input.Media)
		quest = string(questV)
	}
	dbErr := m.dataStorage.Session().RequestSave(ctx, state.ReqID, quest, agent.AgentID())
	if dbErr != nil {
		slog.Error("failed to save request", "error", dbErr)
	}

	// 注册回调，通过闭包携带 msg 和 agentName 信息
	eventSeq := 0
	firstStart := true // 标记是否是第一次 start 事件
	state.EventHandler = func(event *types.Event) {
		if event == nil {
			return
		}
		eventSeq++
		m.handleAgentEvent(ctx, event, eventSeq, &firstStart)
	}
	//for i := range 100 {
	//	event := state.BuildEvent()
	//	event.EventID = uuid.NewV4().String()
	//	if i == 0 {
	//		event.Type = types.EventReasoningStart
	//		event.Message = &types.Message{
	//			Role: types.RoleAssistant,
	//		}
	//	} else if i < 50 {
	//		event.Type = types.EventReasoningDelta
	//		event.Message = &types.Message{
	//			Role:             types.RoleAssistant,
	//			IsDetla:          true,
	//			ReasoningContent: fmt.Sprintf("%d", i),
	//		}
	//	} else if i == 50 {
	//		event.Type = types.EventReasoningCompleted
	//		event.Message = &types.Message{
	//			Role:             types.RoleAssistant,
	//			IsDetla:          false,
	//			ReasoningContent: "1.....50",
	//		}
	//	} else if i == 51 {
	//		event.Type = types.EventMessageStart
	//		event.Message = &types.Message{
	//			Role: types.RoleAssistant,
	//		}
	//	} else if i == 99 {
	//		event.Type = types.EventMessageCompleted
	//		event.Message = &types.Message{
	//			Role:    types.RoleAssistant,
	//			IsDetla: false,
	//			Content: "51....99",
	//		}
	//	} else if i > 50 {
	//		event.Type = types.EventMessageDelta
	//		event.Message = &types.Message{
	//			Role:    types.RoleAssistant,
	//			IsDetla: true,
	//			Content: fmt.Sprintf("%d", i),
	//		}
	//	}
	//	time.Sleep(100 * time.Millisecond)
	//	state.EventHandler(event)
	//}
	//var err error
	// 使用 Agent 处理消息
	err := agent.Prompt(ctx, state)
	eventSeq++

	// 保存进数据库turn完成
	answer := state.LastMessage()
	if err != nil {
		answer = "错误：" + err.Error()
	}
	dbErr = m.dataStorage.Session().TurnSave(ctx, &db.SessionTurn{
		ReqID:       state.ReqID,
		Answer:      answer,
		IsCompleted: true,
		CompletedAt: time.Now(),
	})
	if dbErr != nil {
		slog.Error("fail to save db completed turn ", "error", dbErr.Error())
	}

	//for _, h := range m.agHandler {
	//	isContinue := h.AfterAgent(ctx, state, err)
	//	if !isContinue {
	//		slog.Warn("stop continue to execute agent handler", "sessionKey", state.SessionID, "error", err)
	//		return err
	//	}
	//}

	if err != nil {
		// 同时发布 OutboundMessage，确保调用方能收到响应不会卡住
		outBound := state.BuildEvent()
		outBound.Type = types.EventError
		outBound.Message = &types.Message{
			Content: "失败：" + err.Error(),
		}
		if pubErr := m.bus.PublishEvent(ctx, outBound); pubErr != nil {
			m.log(ctx, bus.LogLevelError, "manager", fmt.Sprintf("Failed to publish error outbound: %v", pubErr))
		}
		return err
	}

	//if len(state.NewMessage) == 0 {
	//	// 没有响应，发布空响应确保调用方不会卡住
	//	outbound := &bus.OutboundMessage{
	//		Channel:     msg.Channel,
	//		ChatID:      msg.ChatID,
	//		Content:     "no response",
	//		ReplyTo:     getReplyTo(msg),
	//		IsStreaming: agent.IsStreaming(),
	//		IsFinal:     true,
	//		Timestamp:   time.Now(),
	//		ChunkIndex:  eventSeq,
	//		Metadata:    msg.Metadata,
	//	}
	//	if pubErr := m.bus.PublishOutbound(ctx, outbound); pubErr != nil {
	//		m.log(ctx, bus.LogLevelError, "manager", fmt.Sprintf("Failed to publish empty outbound: %v", pubErr))
	//	}
	//	return nil
	//}
	//
	//// 这是一条特殊的消息，收到这条消息代表整个事件处理完成了
	//outbound := &bus.OutboundMessage{
	//	Channel:     msg.Channel,
	//	ChatID:      msg.ChatID,
	//	ReplyTo:     getReplyTo(msg),
	//	IsStreaming: agent.IsStreaming(),
	//	IsFinal:     true,
	//	ChunkIndex:  eventSeq,
	//	Timestamp:   time.Now(),
	//	Metadata:    msg.Metadata,
	//}
	//if err = m.bus.PublishOutbound(ctx, outbound); err != nil {
	//	return err
	//}
	//
	//// 发布 complete 事件（状态通知）
	//m.publishChatEvent(ctx, msg.Channel, msg.ChatID, agentName, bus.EventTypeComplete, eventSeq, msg.ID, nil, msg.Metadata)

	return nil
}

// handleAgentEvent 处理 Agent 事件，转发到消息总线
// 使用 firstStart 标记确保每个请求只发送一次 start 状态
func (m *Manager) handleAgentEvent(ctx context.Context, event *types.Event, seq int, firstStart *bool) {
	event.ChunkIndex = seq
	_ = m.bus.PublishEvent(ctx, event)
}

// publishStreamOutbound 发布流式 OutboundMessage
// 根据 StreamingMode 决定发送增量或累积内容
//func (m *Manager) publishStreamOutbound(ctx context.Context, msg *bus.InboundMessage, agentName string, event *agcore.Event, seq int) {
//	// 获取累积模式配置
//	accumulateMode := msg.StreamingMode == bus.StreamingModeAccumulate
//	// 新增的内容
//	contentToAdd := event.Message.Content
//	thinkToAdd := event.Message.ReasoningContent
//
//	// 累积内容（使用入站消息ID作为标识，避免同chatID多条消息混淆）
//	acc := m.getOrCreateAccumulator(msg.ID)
//
//	// 第一个 chunk 去掉开头的回车或空格
//	if acc.content == "" {
//		contentToAdd = strings.TrimLeftFunc(contentToAdd, unicode.IsSpace)
//	}
//	if acc.thinking == "" {
//		thinkToAdd = strings.TrimLeftFunc(thinkToAdd, unicode.IsSpace)
//		thinkToAdd = cleanModelOutput(thinkToAdd)
//	}
//
//	var content, reasoning string
//	if thinkToAdd != "" {
//		acc.thinking += thinkToAdd
//		if accumulateMode {
//			reasoning = acc.thinking
//			content = acc.content
//		} else {
//			reasoning = thinkToAdd
//			content = ""
//		}
//	} else {
//		acc.content += contentToAdd
//		if accumulateMode {
//			content = acc.content
//			reasoning = acc.thinking
//		} else {
//			content = contentToAdd
//			reasoning = ""
//		}
//	}
//
//	// 复制入站消息 metadata
//	outboundMeta := make(map[string]interface{})
//	outboundMeta[meta.KeyStreamingMode] = string(msg.StreamingMode)
//	if msg.Metadata != nil {
//		for k, v := range msg.Metadata {
//			outboundMeta[k] = v
//		}
//	}
//
//	outbound := &bus.OutboundMessage{
//		Channel:          msg.Channel,
//		ChatID:           msg.ChatID,
//		Content:          content,
//		ReasoningContent: reasoning,
//		IsStreaming:      true,
//		IsThinking:       thinkToAdd != "",
//		IsFinal:          false,
//		ChunkIndex:       seq,
//		ReplyTo:          getReplyTo(msg),
//		Timestamp:        time.Now(),
//		Metadata:         outboundMeta,
//	}
//	_ = m.bus.PublishOutbound(ctx, outbound)
//}

// publishEndOutbound 发布完整流式 OutboundMessage
//func (m *Manager) publishEndOutbound(ctx context.Context, msg *bus.InboundMessage, isStreaming bool, event *agcore.Event, seq int) {
//	// 清理累积器（使用入站消息ID）
//	m.removeAccumulator(msg.ID)
//
//	// 复制入站消息 metadata
//	outboundMeta := make(map[string]interface{})
//	outboundMeta[meta.KeyStreamingMode] = string(msg.StreamingMode)
//	if msg.Metadata != nil {
//		for k, v := range msg.Metadata {
//			outboundMeta[k] = v
//		}
//	}
//	content := strings.TrimLeftFunc(event.Message.Content, unicode.IsSpace)
//	think := strings.TrimLeftFunc(event.Message.ReasoningContent, unicode.IsSpace)
//
//	// 最终消息直接使用完整内容
//	outbound := &bus.OutboundMessage{
//		Channel:          msg.Channel,
//		ChatID:           msg.ChatID,
//		Content:          content,
//		ReasoningContent: think,
//		IsStreaming:      isStreaming,
//		IsEnd:            true,
//		IsThinking:       think != "",
//		ChunkIndex:       seq,
//		ReplyTo:          getReplyTo(msg),
//		Timestamp:        time.Now(),
//		Metadata:         outboundMeta,
//	}
//	_ = m.bus.PublishOutbound(ctx, outbound)
//}

func cleanModelOutput(raw string) string {
	// 如果字符串不是以双引号开头，说明可能不是转义 JSON，直接返回
	if !strings.HasPrefix(raw, "\"") {
		return raw
	}

	var decoded string
	// 尝试将原始字符串解析为标准的 Go string
	err := json.Unmarshal([]byte(raw), &decoded)
	if err != nil {
		// 如果解析失败（说明不是标准 JSON 格式），返回原始内容
		return raw
	}
	return decoded
}
