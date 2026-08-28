package types

import "context"

type Agent interface {
	// AgentID 返回agentID
	AgentID() string
	// Prompt 对话请求
	Prompt(ctx context.Context, state *State) error
	// Cancel 取消
	Cancel(ctx context.Context, sessionID string) error
	// Stop 停止Agent
	Stop() error
}
