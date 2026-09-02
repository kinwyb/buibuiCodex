package jsonRpc

type InputItem struct {
	Type string `json:"type"` // 固定为 "text"
	Text string `json:"text"`
	URL  string `json:"url,omitempty"`
	Path string `json:"path,omitempty"`
}

// StartTurnParams 发起 Turn 请求参数
type StartTurnParams struct {
	ThreadID          string         `json:"threadId"` // 小驼峰
	Input             []InputItem    `json:"input"`    // 注意：这里必须是数组 sequence
	ApprovalPolicy    ApprovalPolicy `json:"approvalPolicy,omitempty"`
	ApprovalsReviewer string         `json:"approvalsReviewer,omitempty"`
	SandboxPolicy     *SandboxPolicy `json:"sandboxPolicy,omitempty"`
}

// StartTurnResult 发起 Turn 返回结果
type StartTurnResult struct {
	Turn Turn `json:"turn"`
}

type Turn struct {
	TurnID      string      `json:"id"`
	Status      string      `json:"status"`
	StartedAt   int64       `json:"startedAt"`
	CompletedAt int64       `json:"completedAt"`
	DurationMs  int64       `json:"durationMs"`
	Error       *CodexError `json:"error,omitempty"`
}

type TurnEvent struct {
	ThreadID string `json:"threadId"`
	Turn     Turn   `json:"turn"`
}

type TurnStartEvent = TurnEvent

type TurnCompletedEvent = TurnEvent

type TurnInterruptParams struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}
