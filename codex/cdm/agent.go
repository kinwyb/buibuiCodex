package cdm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"uuid"

	"gitee.com/kinwyb/conv"
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
	// turnRootMap 记录实际 app-server turn ID 到逻辑 TurnID 的映射。
	// 恢复执行会生成新的 turn ID，但事件仍要落到原始 TurnID 下供前端回放。
	turnRootMap  map[string]string //runTurnID => logical TurnID
	isInit       bool
	initMu       sync.Mutex
	tools        []*toolGroup
	historyTurnN int
	reconciler   *turnReconciler
	recovery     *turnRecovery

	// 以下字段只保护 Agent 自身的启动/停止生命周期。
	// Prompt 在开始时登记，退出时注销；Stop 先停止接收新任务，再等待登记数归零。
	lifeMu          sync.Mutex
	stopping        bool
	activeTurns     int
	drainCh         chan struct{}
	stopSignal      context.Context
	stopSignalClose context.CancelFunc
	stopDrain       time.Duration
	stopForceWait   time.Duration
}

var errAgentStopping = errors.New("agent is stopping")

func NewAgent(cfg *config.AgentConfig, historyTurnN int) *Agent {
	ret := &Agent{
		cfg:           cfg,
		eps:           make(map[string]*codexTurn),
		turnMap:       make(map[string]string),
		turnRootMap:   make(map[string]string),
		historyTurnN:  historyTurnN,
		stopDrain:     30 * time.Second,
		stopForceWait: 5 * time.Second,
	}
	ret.resetStopSignal()
	var sessionDB db.ISession
	if cfg != nil {
		sessionDB = cfg.SessionDB
	}
	ret.reconciler = newTurnReconciler(sessionDB, ret.readTurns)
	ret.recovery = newTurnRecovery(ret, sessionDB)
	return ret
}

// SetRecoveryNotifier 注入管理器的事件出口。
// 恢复 turn 不是由 Prompt 触发的，没有现成的调用方 EventHandler；
// 这个出口让恢复过程仍然能像普通对话一样向用户推送事件。
func (a *Agent) SetRecoveryNotifier(notify func(*types.Event)) {
	if a.recovery != nil {
		a.recovery.SetNotifier(notify)
	}
}

func (a *Agent) resetStopSignal() {
	a.stopSignal, a.stopSignalClose = context.WithCancel(context.Background())
}

func (a *Agent) beginTurn() error {
	a.lifeMu.Lock()
	defer a.lifeMu.Unlock()
	if a.stopping {
		return errAgentStopping
	}
	a.activeTurns++
	return nil
}

func (a *Agent) endTurn() {
	a.lifeMu.Lock()
	defer a.lifeMu.Unlock()
	if a.activeTurns > 0 {
		a.activeTurns--
	}
	if a.stopping && a.activeTurns == 0 && a.drainCh != nil {
		close(a.drainCh)
		a.drainCh = nil
	}
}

func (a *Agent) AgentID() string {
	return a.cfg.Name
}

func (a *Agent) Init() error {
	if a.cfg.DockerImage == "" {
		return errors.New("docker image is required")
	}
	// 允许 Stop 后重新 Init；正常运行时这里不会改变状态。
	a.lifeMu.Lock()
	a.stopping = false
	a.activeTurns = 0
	a.drainCh = nil
	a.resetStopSignal()
	a.lifeMu.Unlock()
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
		ModelProvider: a.cfg.ProviderName,
		//APIKey:        a.cfg.Provider.APIKey,
		Env: []string{
			"OPENAI_API_KEY_" + a.cfg.ProviderName + "=" + a.cfg.Provider.APIKey,
			"RUST_LOG=debug",
		},
		MCP:            make(map[string]docker.MCPServerConfig),
		ModelProviders: make(map[string]docker.ModelProvider),
		Memories:       a.cfg.EnableMemory,
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
	client.RegisterDefaultEventHandler(a.clientEvent)
	a.client = client
	a.isInit = true
	a.reconciler.Start()
	// 恢复必须在 client 和 reconciler 就绪后启动：它需要读 app-server 的
	// interrupted 状态，也可能借助 reconciler 先补齐旧 run 的缺失完成帧。
	a.recovery.Start()
	// 禁用默认的skill
	// 默认的图片生成skill
	skill := codex.NewSkill(client)
	skilErr := skill.Disable(context.Background(), "/root/.codex/skills/.system/imagegen/SKILL.md")
	if skilErr != nil {
		slog.Error("disable system imagegen skill failed", "error", skilErr)
	}
	return nil
}

// getThread 获取线程，返回：线程,历史消息,错误。
// requestedThreadID 非空时优先恢复该历史线程；显式 /new 仍然创建新线程。
func (a *Agent) getThread(ctx context.Context, sessionID string, cmd *Command, requestedThreadID string) (*codex.Thread, string, error) {
	a.epMu.Lock()
	defer a.epMu.Unlock()
	historyContent := ""
	thread := a.client.Thread()
	thread.SubUnknowTurnEvent(a.unknowThreadEventHandler)
	waitResume := true
	requestedThreadID = strings.TrimSpace(requestedThreadID)
	if cmd.NewThread {
		// /new 是用户显式要求新开线程，优先于历史选择
		requestedThreadID = ""
	}
	turn, ok := a.eps[sessionID]
	if requestedThreadID != "" {
		if !ok || turn.threadID != requestedThreadID {
			turn = &codexTurn{
				threadID:       requestedThreadID,
				startTime:      time.Now().Unix(),
				lastUpdateTime: time.Now().Unix(),
			}
		}
	} else if cmd.NewThread || !ok {
		// 查询数据库中使用过的thread
		dbThread := a.cfg.SessionDB.LastThread(ctx, sessionID, a.AgentID())
		if dbThread != nil && time.Now().Sub(dbThread.LastUpdate) > 1*time.Hour {
			dbThread = nil
		}
		if !cmd.NewThread && dbThread != nil {
			turn = &codexTurn{
				startTime:      dbThread.CreateTime.Unix(),
				threadID:       dbThread.ThreadID,
				lastUpdateTime: dbThread.LastUpdate.Unix(),
			}
		} else {
			// 没有有效的thread，创建新的thread
			waitResume = false
			if !cmd.NewThread && a.historyTurnN > 0 { //用户没有指定要求新开线程，注入历史消息内容
				history := NewHistory(filepath.Join(a.cfg.WorkSpace, "root"), a.cfg.SessionDB, a.historyTurnN)
				historyContent = history.History(sessionID, a.AgentID())
			}
			nTurn, err := a.newThread(ctx, thread, sessionID)
			if err != nil {
				return nil, historyContent, err
			}
			turn = nTurn
		}
	}
	if waitResume {
		err := thread.Resume(ctx, turn.threadID)
		if err != nil {
			if strings.HasPrefix(err.Error(), "no rollout found for thread id") {
				if requestedThreadID != "" {
					// 指定线程无法恢复时不能悄悄换新线程，否则历史回复会落到错误上下文。
					return nil, historyContent, fmt.Errorf("恢复指定 thread %s 失败: %w", requestedThreadID, err)
				}
				nTurn, err := a.newThread(ctx, thread, sessionID)
				if err != nil {
					return nil, historyContent, err
				}
				turn = nTurn
			} else {
				slog.Error(err.Error())
				return nil, historyContent, err
			}
		}
		turn.thread = thread
		turn.lastUpdateTime = time.Now().Unix()
	}
	a.eps[sessionID] = turn

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
	return thread, historyContent, nil
}

func (a *Agent) readTurns(ctx context.Context, threadID string, turnIDs []string) (map[string]*jsonRpc.Turn, error) {
	a.initMu.Lock()
	client := a.client
	a.initMu.Unlock()
	if client == nil {
		return nil, errors.New("codex client is not initialized")
	}
	return client.ReadTurns(ctx, threadID, turnIDs)
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
	if err := a.beginTurn(); err != nil {
		return err
	}
	defer a.endTurn()
	// 初始化进程
	err := a.initProcess()
	if err != nil {
		return err
	}
	// initProcess 可能刚启动 recovery 扫描，因此必须先初始化再等待恢复标记。
	// 恢复 turn 期间同一 session 不能并发启动普通 turn，否则 app-server 会拒绝。
	if err := a.recovery.WaitSession(ctx, state.SessionID); err != nil {
		return err
	}
	cmd := CommandParse(state.Input.Content)
	state.Input.Content = cmd.Msg
	thread, hisotryContent, err := a.getThread(ctx, state.SessionID, cmd, state.Input.ThreadID())
	if err != nil {
		return err
	}
	turn, err := thread.Turn()
	if err != nil {
		err = fmt.Errorf("thread build turn fail: %w", err)
		slog.Error(err.Error())
		return err
	}
	// 注册消息处理
	completed := make(chan error, 1)
	turn.RegisterEventHandler(a.turnEvent(state, completed))
	var opts []codex.TurnStartOptions
	opts = append(opts, codex.TurnStartWithSandboxPolicy(jsonRpc.SandboxPolicy{
		Type:          jsonRpc.SandboxPolicyTypeDangerFullAccess,
		NetworkAccess: true,
	}), codex.TurnStartWithApprovalPolicy(jsonRpc.Never))
	var inputs []jsonRpc.InputItem
	// 指定不需要历史记录，那就忽略历史记录
	if state.Input != nil && state.Input.Metadata != nil && state.Input.Metadata[types.MetadataNoHistory] != nil {
		noHistory := conv.ToBool(state.Input.Metadata[types.MetadataNoHistory])
		if noHistory {
			hisotryContent = ""
		}
	}
	if hisotryContent != "" {
		inputs = append(inputs, turn.BuildInputWithText(hisotryContent)...)
	}
	inputs = a.buildInput(turn, state.Input, inputs...)
	err = turn.Start(ctx, inputs, opts...)
	if err != nil {
		err = fmt.Errorf("turn start fail: %w", err)
		slog.Error(err.Error())
		return err
	}
	a.epMu.Lock()
	if ct, ok := a.eps[state.SessionID]; ok {
		ct.turn = turn
		ct.state = state
		a.turnMap[turn.TurnID()] = state.SessionID
		a.turnRootMap[turn.TurnID()] = turn.RootTurnID()
	}
	a.epMu.Unlock()
	// 保存turn
	dbErr := a.cfg.SessionDB.TurnSave(ctx, &db.SessionTurn{
		ReqID:          state.ReqID,
		TurnID:         turn.RootTurnID(),
		RunTurnID:      turn.TurnID(),
		ThreadID:       turn.ThreadID(),
		SessionID:      state.SessionID,
		Agent:          a.AgentID(),
		RecoveryStatus: db.RecoveryStatusRunning,
	})
	if dbErr != nil {
		slog.Error("fail to save db turn ", "error", dbErr.Error())
	}
	select {
	case err = <-completed:
	case <-a.stopSignal.Done():
		// 只有 Stop 超时或强制退出时才会触发。正常关闭会等待 turn 自己完成。
		err = errAgentStopping
	}
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

// Steer 请求运行时添加附带信息
func (a *Agent) Steer(ctx context.Context, state *types.State) error {
	a.epMu.Lock()
	ct, ok := a.eps[state.SessionID]
	a.epMu.Unlock()
	if !ok {
		return fmt.Errorf("no running turn instance for session %s", state.SessionID)
	}
	turn := ct.turn
	if turn == nil {
		return fmt.Errorf("no running turn instance for session %s", state.SessionID)
	}
	inputs := a.buildInput(turn, state.Input)
	err := turn.Steer(ctx, inputs)
	if err != nil {
		return fmt.Errorf("turn steer fail: %w", err)
	}
	return nil
}

func (a *Agent) buildInput(turn *codex.Turn, message *types.InputMessage, inputs ...jsonRpc.InputItem) []jsonRpc.InputItem {
	var result = inputs
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
			// metadata 可能为空：外部渠道未必填 filename，缺了就按路径末段或随机名兜底。
			// 这里不能直接做 .(string) 单值断言——值不存在时会 panic，整轮对话直接断掉。
			filename := ""
			if item.Metadata != nil {
				if v, ok := item.Metadata["filename"].(string); ok {
					filename = v
				}
			}
			if filename == "" && item.URL != "" && !strings.HasPrefix(item.URL, "http") {
				filename = filepath.Base(item.URL)
			}
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

// Stop 先停止接收新任务，再等待当前活跃 turn 结束；超过宽限期后中断并强制退出。
//
// 这里的“等待”只针对已经进入 Prompt/恢复流程的有限任务集合。Manager 在调用
// Stop 之前已经关闭入站订阅，因此不会出现“等任务时又不断进来新消息”的死循环。
func (a *Agent) Stop() error {
	a.lifeMu.Lock()
	a.stopping = true
	var drain <-chan struct{}
	if a.activeTurns > 0 {
		ch := make(chan struct{})
		a.drainCh = ch
		drain = ch
	} else {
		closed := make(chan struct{})
		close(closed)
		drain = closed
	}
	a.lifeMu.Unlock()
	if a.recovery != nil {
		// 停止扫描新的恢复任务，但正在执行的 recovery continuation 仍计入 activeTurns，
		// 由下面的 drain 逻辑等待或超时中断。
		a.recovery.BeginStop()
	}

	timer := time.NewTimer(a.stopDrain)
	defer timer.Stop()
	forced := false
	select {
	case <-drain:
	case <-timer.C:
		slog.Warn("agent stop drain timeout, interrupting active turns")
		ctx, cancel := context.WithTimeout(context.Background(), a.stopForceWait)
		a.interruptActive(ctx)
		cancel()

		// 给 Interrupt 一个短暂的收敛窗口，之后主动解除 Prompt 的等待，
		// 防止 app-server 已经失联时整个进程永远停在 Stop。
		forceTimer := time.NewTimer(a.stopForceWait)
		select {
		case <-drain:
			forceTimer.Stop()
		case <-forceTimer.C:
			forced = true
			slog.Error("agent stop force timeout, abandoning active turns")
			// 只有在确认无法收敛时才标记 interrupted。若中断后 turn 正常结束，
			// turnEvent 已经写入了真实的 completed/failed 状态，不能被这里覆盖。
			a.markActiveTurnsInterrupted(context.Background())
			a.stopSignalClose()
		}
	}

	// reconciler 会读取 client，必须在关闭 client 前停止；同时放在 drain 之后，
	// 让正常结束的 turn 还有机会把最后一批 TurnCompleted 事件补全并落库。
	a.reconciler.Stop()
	if a.recovery != nil {
		a.recovery.Wait()
	}
	a.initMu.Lock()
	defer a.initMu.Unlock()
	var errs []error
	if a.client != nil {
		errs = append(errs, a.client.Close())
	}
	if a.process != nil {
		errs = append(errs, a.process.Stop())
	}
	a.isInit = false
	a.epMu.Lock()
	clear(a.eps)
	clear(a.turnMap)
	clear(a.turnRootMap)
	a.epMu.Unlock()
	if forced {
		errs = append(errs, errAgentStopping)
	}
	return errors.Join(errs...)
}

func (a *Agent) interruptActive(ctx context.Context) {
	a.epMu.Lock()
	turns := make([]*codex.Turn, 0, len(a.eps))
	for _, ct := range a.eps {
		if ct.turn != nil {
			turns = append(turns, ct.turn)
		}
	}
	a.epMu.Unlock()
	for _, turn := range turns {
		if err := turn.Interrupt(ctx); err != nil {
			slog.Warn("interrupt active turn failed", "turnID", turn.TurnID(), "error", err.Error())
		}
	}
}

func (a *Agent) markActiveTurnsInterrupted(ctx context.Context) {
	if a.cfg == nil || a.cfg.SessionDB == nil {
		return
	}
	a.epMu.Lock()
	type activeTurn struct {
		reqID string
		runID string
	}
	active := make([]activeTurn, 0, len(a.eps))
	for _, ct := range a.eps {
		if ct.state == nil || ct.turn == nil {
			continue
		}
		active = append(active, activeTurn{reqID: ct.state.ReqID, runID: ct.turn.TurnID()})
	}
	a.epMu.Unlock()
	for _, turn := range active {
		if err := a.cfg.SessionDB.TurnRecoveryUpdate(
			ctx, turn.reqID, turn.runID, db.RecoveryStatusInterrupted, "service stopping", 0,
		); err != nil {
			slog.Warn("mark turn interrupted failed", "reqID", turn.reqID, "error", err.Error())
		}
	}
}
