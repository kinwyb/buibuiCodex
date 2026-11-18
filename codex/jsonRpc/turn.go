package jsonRpc

import "strings"

type InputItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	URL  string `json:"url,omitempty"`
	Path string `json:"path,omitempty"`
	Name string `json:"name,omitempty"`
}

// FileInputText 文件附件写入 turn 输入的文本格式。
// userMessage 回显解析按同一格式把附件还原成结构化数据，两端共用以免格式漂移。
func FileInputText(name, url string) string {
	return "文件[" + name + "]地址:" + url
}

// ParseFileInputText 反解 FileInputText 格式；不是文件行时返回 ok=false。
func ParseFileInputText(text string) (name, url string, ok bool) {
	const open = "文件["
	const sep = "]地址:"
	if !strings.HasPrefix(text, open) {
		return "", "", false
	}
	idx := strings.Index(text[len(open):], sep)
	if idx < 0 {
		return "", "", false
	}
	name = text[len(open) : len(open)+idx]
	url = text[len(open)+idx+len(sep):]
	if name == "" || url == "" {
		return "", "", false
	}
	return name, url, true
}

// StartTurnParams 发起 Turn 请求参数
type StartTurnParams struct {
	ThreadID          string         `json:"threadId"` // 小驼峰
	Input             []InputItem    `json:"input"`    // 注意：这里必须是数组 sequence
	ApprovalPolicy    ApprovalPolicy `json:"approvalPolicy,omitempty"`
	ApprovalsReviewer string         `json:"approvalsReviewer,omitempty"`
	SandboxPolicy     *SandboxPolicy `json:"sandboxPolicy,omitempty"`
	// ParentTurnID/RootTurnID 用于 recovery continuation：实际 turn ID 会变化，
	// 但 app-server 需要继续挂到原始逻辑轮次上。
	ParentTurnID string `json:"parentTurnId,omitempty"`
	RootTurnID   string `json:"rootTurnId,omitempty"`
}

// StartTurnResult 发起 Turn 返回结果
type StartTurnResult struct {
	Turn Turn `json:"turn"`
}

type Turn struct {
	TurnID      string       `json:"id"`
	RootTurnID  string       `json:"rootTurnId,omitempty"`
	Status      string       `json:"status"`
	StartedAt   int64        `json:"startedAt"`
	CompletedAt int64        `json:"completedAt"`
	DurationMs  int64        `json:"durationMs"`
	Error       *CodexError  `json:"error,omitempty"`
	ItemsView   string       `json:"itemsView,omitempty"`
	Items       []ThreadItem `json:"items,omitempty"`
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

// TurnSteerParams 附加 Turn 信息请求参数
type TurnSteerParams struct {
	ThreadID string      `json:"threadId"`       // 小驼峰
	Input    []InputItem `json:"input"`          // 注意：这里必须是数组 sequence
	TurnID   string      `json:"expectedTurnId"` //执行的turnID
}
