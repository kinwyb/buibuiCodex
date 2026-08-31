package types

import (
	"context"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
)

type Agent interface {
	// AgentID 返回agentID
	AgentID() string
	// Prompt 对话请求
	Prompt(ctx context.Context, state *State) error
	// Approve 提交审批结果
	Approve(approvalID any, approve jsonRpc.ApprovalDecision) error
	// Cancel 取消
	Cancel(ctx context.Context, sessionID string) error
	// Stop 停止Agent
	Stop() error
}
