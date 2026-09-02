package codex

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
)

type ThreadStartOption func(*jsonRpc.StartThreadParams)

// NewThreadStartParam 创建一个线程启动参数
func NewThreadStartParam(workspace string, options ...ThreadStartOption) *jsonRpc.StartThreadParams {
	ret := &jsonRpc.StartThreadParams{
		Workspace: workspace,
	}
	for _, option := range options {
		option(ret)
	}
	return ret
}

// ThreadStartWithSendbox 配置沙盒模式
func ThreadStartWithSendbox(sendbox jsonRpc.SandboxMode) ThreadStartOption {
	return func(params *jsonRpc.StartThreadParams) {
		params.Sandbox = sendbox
	}
}

// ThreadStartWithConfig 线程默认配置
func ThreadStartWithConfig(config *jsonRpc.ThreadConfig) ThreadStartOption {
	return func(params *jsonRpc.StartThreadParams) {
		params.Config = config
	}
}

// ThreadStartWithDynamicTool 添加动态函数
func ThreadStartWithDynamicTool(dyts ...jsonRpc.DynamicTool) ThreadStartOption {
	return func(params *jsonRpc.StartThreadParams) {
		params.DynamicTools = append(params.DynamicTools, dyts...)
	}
}

func ThreadStartWithInstructions(instructions string) ThreadStartOption {
	return func(params *jsonRpc.StartThreadParams) {
		params.Instructions = instructions
	}
}

func ThreadStartWithModelProvider(modelProvider string) ThreadStartOption {
	return func(params *jsonRpc.StartThreadParams) {
		params.ModelProvider = modelProvider
	}
}

type Thread struct {
	client        *Client
	threadID      string
	turnMux       sync.Mutex
	turnSub       map[string]EventHandler
	unknowTurnSub EventHandler
}

// NewThread 创建Thread
func NewThread(client *Client) *Thread {
	return &Thread{
		client:  client,
		turnSub: make(map[string]EventHandler),
	}
}

func (t *Thread) ThreadID() string {
	return t.threadID
}

// Start 启动一个线程,返回线程ID
func (t *Thread) Start(ctx context.Context, params jsonRpc.StartThreadParams) (string, error) {
	var res jsonRpc.StartThreadResult
	err := t.client.Call(ctx, ThreadStart, params, &res)
	if err != nil {
		return "", err
	}
	t.threadID = res.Thread.ID
	t.client.SubScribeEvent(t.threadID, t.eventHandler)
	return res.Thread.ID, nil
}

// Resume 恢复一个线程
func (t *Thread) Resume(ctx context.Context, threadID string) error {
	var res jsonRpc.StartThreadResult
	err := t.client.Call(ctx, ThreadResume, map[string]string{
		"threadId": threadID,
	}, &res)
	if err != nil {
		return err
	}
	t.threadID = res.Thread.ID
	t.client.SubScribeEvent(t.threadID, t.eventHandler)
	return nil
}

func (t *Thread) eventHandler(event *Event) {
	t.turnMux.Lock()
	sub, ok := t.turnSub[event.TurnID]
	t.turnMux.Unlock()
	if ok {
		sub(event)
		if event.Method == string(TurnCompleted) {
			t.UnSubScribeEvent(event.TurnID)
		}
		return
	} else if t.unknowTurnSub != nil {
		t.unknowTurnSub(event)
		return
	}
	slog.Debug("turn eventHandler => " + event.Method + " " + event.RawData.String())
}

// SubUnknowTurnEvent 订阅未知的Turn信息,这是因为turn如果中断，后续thread恢复后可能继续执行，而turn整个已经不存在于运行状态中了
func (t *Thread) SubUnknowTurnEvent(handler EventHandler) {
	t.unknowTurnSub = handler
}

// SubScribeEvent 订阅事件
func (t *Thread) SubScribeEvent(turnID string, handler EventHandler) error {
	if turnID == "" {
		return fmt.Errorf("turn id is empty")
	}
	t.turnMux.Lock()
	defer t.turnMux.Unlock()
	t.turnSub[turnID] = handler
	return nil
}

// UnSubScribeEvent 取消事件订阅
func (t *Thread) UnSubScribeEvent(turnID string) {
	if turnID == "" {
		return
	}
	t.turnMux.Lock()
	defer t.turnMux.Unlock()
	delete(t.turnSub, turnID)
}

// Turn 创建一个turn
func (t *Thread) Turn() (*Turn, error) {
	if t.threadID == "" {
		return nil, errors.New("threadId is empty")
	}
	return NewTurn(t.client, t), nil
}
