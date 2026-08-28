package jsonRpc

import (
	"encoding/json"
	"fmt"
)

// --- JSON-RPC 2.0 基础协议定义 ---

type RPCData struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

func (r *RPCData) Request() *RPCRequest {
	return &RPCRequest{
		JSONRPC: r.JSONRPC,
		ID:      r.ID,
		Method:  r.Method,
		Params:  r.Params,
	}
}

func (r *RPCData) Response() *RPCResponse {
	return &RPCResponse{
		JSONRPC: r.JSONRPC,
		ID:      r.ID,
		Result:  r.Result,
		Error:   r.Error,
	}
}

func (r *RPCData) IsResponse() bool {
	return r.ID != nil && (len(r.Result) > 0 || r.Error != nil)
}

type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

func (r RPCResponse) IDString() string {
	if r.ID == nil {
		return ""
	}
	return fmt.Sprintf("%v", r.ID)
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return e.Message
}

// --- Codex 业务参数定义 ---

// InitializeParams 初始化请求参数
type InitializeParams struct {
	ClientInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
	Capabilitie *Capabilities `json:"capabilities,omitempty"`
}

type Capabilities struct {
	ExperimentalApi           bool     `json:"experimentalApi"`
	OptOutNotificationMethods []string `json:"optOutNotificationMethods,omitempty"`
}

// ToolCall 函数调用
type ToolCall struct {
	ThreadID  string         `json:"threadId"`
	TurnID    string         `json:"turnId"`
	CallId    string         `json:"callId"`
	Namespace string         `json:"namespace"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
}

type ToolResponse struct {
	Success bool        `json:"success"`
	Content []InputItem `json:"contentItems"`
}

type EventItemType string

const (
	AgentMessage        EventItemType = "agentMessage"
	CollabAgentToolCall EventItemType = "collabAgentToolCall"
	CommandExecution    EventItemType = "commandExecution"
	ContextCompaction   EventItemType = "contextCompaction"
	DynamicToolCall     EventItemType = "dynamicToolCall"
	EnteredReviewMode   EventItemType = "enteredReviewMode"
	ExitedReviewMode    EventItemType = "exitedReviewMode"
	FileChange          EventItemType = "fileChange"
	HookPrompt          EventItemType = "hookPrompt"
	ImageGeneration     EventItemType = "imageGeneration"
	ImageView           EventItemType = "imageView"
	Sleep               EventItemType = "sleep"
	SubAgentActivity    EventItemType = "subAgentActivity"
	MCPToolCall         EventItemType = "mcpToolCall"
	Plan                EventItemType = "plan"
	Reasoning           EventItemType = "reasoning"
	UserMessage         EventItemType = "userMessage"
	WebSearch           EventItemType = "webSearch"
)

type ApprovalPolicy string

const (
	AskForApprovalUntrusted ApprovalPolicy = "untrusted"
	Never                   ApprovalPolicy = "never"
	OnRequest               ApprovalPolicy = "on-request"
)

// CodexErrorEvent 对应日志中的完整 JSON 结构
type CodexErrorEvent struct {
	Error             CodexError `json:"error"`
	WillRetry         bool       `json:"willRetry"`
	ThreadID          string     `json:"threadId"`
	TurnID            string     `json:"turnId"`
	AdditionalDetails string     `json:"additionalDetails,omitempty"`
}

// CodexError 包含错误的具体描述与内部错误分类
type CodexError struct {
	Message           string         `json:"message"`
	CodexErrorInfo    CodexErrorInfo `json:"codexErrorInfo"`
	AdditionalDetails string         `json:"additionalDetails,omitempty"`
}

// CodexErrorInfo 内部具体的错误类型枚举/映射
type CodexErrorInfo struct {
	ResponseStreamDisconnected *ResponseStreamDisconnected `json:"responseStreamDisconnected,omitempty"`
}

// ResponseStreamDisconnected 流断开的具体状态信息
type ResponseStreamDisconnected struct {
	HTTPStatusCode *int `json:"httpStatusCode"` // 使用指针支持 null 值
}
