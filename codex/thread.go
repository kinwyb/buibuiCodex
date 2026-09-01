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
func ThreadStartWithDynamicTool(dyt *jsonRpc.DynamicTool) ThreadStartOption {
	return func(params *jsonRpc.StartThreadParams) {
		params.DynamicTools = append(params.DynamicTools, *dyt)
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
	client   *Client
	threadID string
	turnMux  sync.Mutex
	turnSub  map[string]EventHandler
}

// NewThread 创建Thread
func NewThread(client *Client) *Thread {
	return &Thread{
		client:  client,
		turnSub: make(map[string]EventHandler),
	}
}

// Start 启动一个线程,返回线程ID
func (s *Thread) Start(ctx context.Context, params jsonRpc.StartThreadParams) (string, error) {
	var res jsonRpc.StartThreadResult
	err := s.client.Call(ctx, ThreadStart, params, &res)
	if err != nil {
		return "", err
	}
	s.threadID = res.Thread.ID
	s.client.SubScribeEvent(s.threadID, s.eventHandler)
	return res.Thread.ID, nil
}

// Resume 恢复一个线程
func (s *Thread) Resume(ctx context.Context, threadID string) error {
	var res jsonRpc.StartThreadResult
	err := s.client.Call(ctx, ThreadResume, map[string]string{
		"threadId": threadID,
	}, &res)
	if err != nil {
		return err
	}
	s.threadID = res.Thread.ID
	s.client.SubScribeEvent(s.threadID, s.eventHandler)
	return nil
}

func (s *Thread) eventHandler(event *Event) {
	s.turnMux.Lock()
	sub, ok := s.turnSub[event.TurnID]
	s.turnMux.Unlock()
	if ok {
		sub(event)
		if event.Method == string(TurnCompleted) {
			s.UnSubScribeEvent(event.TurnID)
		}
	}
	// 处理其他消息
	slog.Debug("turn eventHandler => " + event.Method + " " + event.RawData.String())
}

// SubScribeEvent 订阅事件
func (s *Thread) SubScribeEvent(turnID string, handler EventHandler) error {
	if turnID == "" {
		return fmt.Errorf("turn id is empty")
	}
	s.turnMux.Lock()
	defer s.turnMux.Unlock()
	s.turnSub[turnID] = handler
	return nil
}

// UnSubScribeEvent 取消事件订阅
func (s *Thread) UnSubScribeEvent(turnID string) {
	if turnID == "" {
		return
	}
	s.turnMux.Lock()
	defer s.turnMux.Unlock()
	delete(s.turnSub, turnID)
}

// Turn 创建一个turn
func (s *Thread) Turn() (*Turn, error) {
	if s.threadID == "" {
		return nil, errors.New("threadId is empty")
	}
	return NewTurn(s.client, s), nil
}
