package types

import (
	"context"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
)

type Agent interface {
	// AgentID 返回agentID
	AgentID() string
	// Init 初始化agent
	Init() error
	// Prompt 对话请求
	Prompt(ctx context.Context, state *State) error
	// Approve 提交审批结果
	Approve(approvalID any, approve jsonRpc.ApprovalDecision) error
	// Steer 运行中附加信息
	Steer(ctx context.Context, state *State) error
	// Cancel 取消
	Cancel(ctx context.Context, sessionID string) error
	// Stop 停止Agent
	Stop() error
}

type RouterAgentResult struct {
	TargetAgent string `description:"目标agent"`   //如果agent存在会以这个agent作为处理目标,如果不存在或者为空按原逻辑agent处理
	SkipAgent   bool   `description:"跳过agent处理"` //如果这个为true，不会再进行后续处理
}

// AgentProcessHandler agent处理前的拦截器
type AgentProcessHandler interface {
	// RouterAgent agent路由处理
	RouterAgent(msg *InputMessage) *RouterAgentResult
	// BeforeAgent agent处理之前调用。
	// 返回 skip=true 时跳过大模型处理，直接把 reply 作为最终响应返回给用户。
	BeforeAgent(ctx context.Context, state *State) (skip bool, reply string, err error)
	// AfterAgent agent处理之后调用, 如果返回false代表后续处理中断，不再继续后续流程
	AfterAgent(ctx context.Context, state *State, err error) bool
}
