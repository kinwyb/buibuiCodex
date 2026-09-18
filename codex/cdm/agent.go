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
	cfg          *config.AgentConfig
	process      proc.Process
	client       *codex.Client
	epMu         sync.Mutex
	eps          map[string]*codexTurn
	turnMap      map[string]string //turnID => sessionID
	isInit       bool
	initMu       sync.Mutex
	tools        []*toolGroup
	historyTurnN int
}

func NewAgent(cfg *config.AgentConfig, historyTurnN int) *Agent {
	ret := &Agent{
		cfg:          cfg,
		eps:          make(map[string]*codexTurn),
		historyTurnN: historyTurnN,
	}
	return ret
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
	// 禁用默认的skill
	// 默认的图片生成skill
	skill := codex.NewSkill(client)
	skilErr := skill.Disable(context.Background(), "/root/.codex/skills/.system/imagegen/SKILL.md")
	if skilErr != nil {
		slog.Error("disable system imagegen skill failed", "error", skilErr)
	}
	return nil
}

// getThread 获取线程，返回：线程,历史消息,错误
func (a *Agent) getThread(ctx context.Context, sessionID string, cmd *command) (*codex.Thread, string, error) {
	a.epMu.Lock()
	defer a.epMu.Unlock()
	historyContent := ""
	thread := a.client.Thread()
	thread.SubUnknowTurnEvent(a.unknowThreadEventHandler)
	waitResume := true
	turn, ok := a.eps[sessionID]
	if cmd.newThread || !ok {
		// 查询数据库中使用过的thread
		dbThread := a.cfg.SessionDB.LastThread(ctx, sessionID, a.AgentID())
		if dbThread != nil && time.Now().Sub(dbThread.LastUpdate) > 1*time.Hour {
			dbThread = nil
		}
		if !cmd.newThread && dbThread != nil {
			turn = &codexTurn{
				startTime:      dbThread.CreateTime.Unix(),
				threadID:       dbThread.ThreadID,
				lastUpdateTime: dbThread.LastUpdate.Unix(),
			}
		} else {
			// 没有有效的thread，创建新的thread
			waitResume = false
			if !cmd.newThread && a.historyTurnN > 0 { //用户没有指定要求新开线程，注入历史消息内容
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
			slog.Error(err.Error())
			return nil, historyContent, err
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
	return thread, historyContent, nil
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
	thread, hisotryContent, err := a.getThread(ctx, state.SessionID, cmd)
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
