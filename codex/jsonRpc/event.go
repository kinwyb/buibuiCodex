package jsonRpc

import (
	"encoding/json"
	"strconv"
	"strings"
)

// EventNotification 统一下发事件包装
type EventNotification struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// baseNotificationEvent 通用事件基础字段
type baseNotificationEvent struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

// TokenUsageNotification 对应 thread/tokenUsage/updated 事件推送
type TokenUsageNotification struct {
	baseNotificationEvent
	TokenUsage ThreadTokenUsage `json:"tokenUsage"`
}

type ThreadTokenUsage struct {
	Total              TokenDetail `json:"total"`              // 当前 Thread 累计用量
	Last               TokenDetail `json:"last"`               // 本轮交互用量
	ModelContextWindow int64       `json:"modelContextWindow"` // 模型最大上下文窗口
}

// TokenDetail 单次或累计的详细 Token 拆分
type TokenDetail struct {
	TotalTokens           int64 `json:"totalTokens"`           // 总 Token 数
	InputTokens           int64 `json:"inputTokens"`           // 输入 Token 数
	CachedInputTokens     int64 `json:"cachedInputTokens"`     // 命中的缓存 Token
	CacheWriteInputTokens int64 `json:"cacheWriteInputTokens"` // 写入缓存的 Token
	OutputTokens          int64 `json:"outputTokens"`          // 输出 Token 数
	ReasoningOutputTokens int64 `json:"reasoningOutputTokens"` // 模型深度思考/推理消耗的 Token (o1/o3/R1等)
}

// ==========================================
// ItemLifecycleEvent Item声明与生命周期 (item/started & item/completed)
// ==========================================
type ItemLifecycleEvent struct {
	baseNotificationEvent
	StartedAtMs   int64         `json:"startedAtMs,omitempty"`
	CompletedAtMs int64         `json:"completedAtMs,omitempty"`
	Item          itemlifecycle `json:"item"`
}

type itemlifecycle struct {
	ID   string        `json:"id"`
	Type EventItemType `json:"type"` // "reasoning"
	ReasoningItemDetail
	CommandExecutionItemDetail
	AgentMessageItemDetail
	WebSearchItemDetail
	ToolCallItemDetail
}

// item 状态取值，commandExecution / dynamicToolCall / mcpToolCall 共用
const (
	ItemStatusInProgress = "inProgress"
	ItemStatusCompleted  = "completed"
	ItemStatusFailed     = "failed"
)

// ToolExtras 汇总工具调用帧的附加字段，实时流（cdm）与回放（replay）共用，
// 两条链路给出同一套 extra，前端才能用同一条渲染分支。
func (i itemlifecycle) ToolExtras() map[string]any {
	extra := make(map[string]any, 4)
	if i.Namespace != "" {
		extra["namespace"] = i.Namespace
	}
	if i.CallId != "" {
		extra["callID"] = i.CallId
	}
	// mcpToolCall 没有 namespace，改带 MCP 服务名。
	// 「是不是 MCP」由 ToolCall.Type 区分，这里不重复表态。
	if i.Server != "" {
		extra["server"] = i.Server
	}
	if i.Status != "" {
		extra["status"] = i.Status
	}
	// success：dynamicToolCall 报文里直接给；mcpToolCall 只能由 status/error 推导，
	// inProgress 阶段不表态，否则开始帧就会被前端画成失败。
	errText := i.ErrorText()
	switch {
	case i.Success != nil:
		extra["success"] = *i.Success
	case errText != "":
		extra["success"] = false
		extra["error"] = errText
	case i.Status == ItemStatusCompleted:
		extra["success"] = true
	case i.Status == ItemStatusFailed:
		extra["success"] = false
	}
	if i.DurationMs > 0 {
		extra["durationMs"] = i.DurationMs
	}
	return extra
}

// ToolCallItemDetail 函数调用，dynamicToolCall 与 mcpToolCall 共用。
//
// 注意字段来源：codex 的 dynamicToolCall item/completed 里工具结果是 contentItems，不是 result，
// 后者在该类型下并不存在（全库事件无一例带 result）。只读 result 会让工具结果永远为空。
// mcpToolCall 恰好相反：结果在 result 里（原样透传 MCP 的 CallToolResult），没有 contentItems，
// 失败信息在 error 里，且不带 success。
// status / durationMs 与本结构体的兄弟 CommandExecutionItemDetail 同名，共用同一 tag，
// 这里不再重复声明，直接读 item.Status / item.DurationMs。
type ToolCallItemDetail struct {
	CallId       string                  `json:"callId"`
	Namespace    string                  `json:"namespace"`
	Server       string                  `json:"server"`
	Tool         string                  `json:"tool"`
	Arguments    map[string]any          `json:"arguments"`
	Result       json.RawMessage         `json:"result"`
	Success      *bool                   `json:"success"`
	Error        json.RawMessage         `json:"error"`
	ContentItems []ToolOutputContentItem `json:"contentItems"`
}

// MCPToolResult MCP 工具返回体，codex 原样透传服务端的 CallToolResult。
type MCPToolResult struct {
	Content           []MCPContent    `json:"content"`
	StructuredContent json.RawMessage `json:"structuredContent"`
	IsError           bool            `json:"isError"`
}

// MCPContent MCP 返回内容项，type 取值 text | image | audio | resource
type MCPContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// MCPError MCP 工具调用错误，对应 item.error
type MCPError struct {
	Message string `json:"message"`
}

// ToolOutputContentItem 工具输出项，对应 codex 的 DynamicToolCallResponse.contentItems。
// 与回给 codex 的 ToolResponse.Content 是同一份结构的两个视角。
type ToolOutputContentItem struct {
	// Type 取值 inputText | inputImage | inputAudio
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
	AudioURL string `json:"audioUrl,omitempty"`
}

// ResultText 取工具执行结果的文本形式，供实时流与历史回放共用。
//
// dynamicToolCall 的结果主载体是 contentItems（其 item/completed 里没有 result 字段），
// mcpToolCall 的结果在 result.content 里，先展开成文本。逻辑集中在这里，
// 避免两条解析路径各写一份。
func (d ToolCallItemDetail) ResultText() string {
	if text := d.mcpResultText(); text != "" {
		return text
	}
	if text := rawJSONText(d.Result); text != "" {
		return text
	}
	parts := make([]string, 0, len(d.ContentItems))
	for _, item := range d.ContentItems {
		switch {
		case item.Text != "":
			parts = append(parts, item.Text)
		case item.ImageURL != "":
			parts = append(parts, "[图片] "+item.ImageURL)
		case item.AudioURL != "":
			parts = append(parts, "[音频] "+item.AudioURL)
		}
	}
	return strings.Join(parts, "\n")
}

// ResultOrError 结果文本，失败且没有结果时退回错误信息，
// 保证前端在工具报错时也有内容可渲染，而不是一个空结果。
func (d ToolCallItemDetail) ResultOrError() string {
	if text := d.ResultText(); text != "" {
		return text
	}
	return d.ErrorText()
}

// ErrorText 取 mcpToolCall 的错误信息，没有错误返回空串。
func (d ToolCallItemDetail) ErrorText() string {
	var mcpErr MCPError
	if err := json.Unmarshal(d.Error, &mcpErr); err == nil && mcpErr.Message != "" {
		return mcpErr.Message
	}
	return rawJSONText(d.Error)
}

// mcpResultText 展开 MCP 的 CallToolResult。非 MCP 结构（或没有内容）时返回空串，
// 由 ResultText 继续走 result 原文 / contentItems 分支。
func (d ToolCallItemDetail) mcpResultText() string {
	if len(d.Result) == 0 {
		return ""
	}
	var result MCPToolResult
	if err := json.Unmarshal(d.Result, &result); err != nil {
		return ""
	}
	parts := make([]string, 0, len(result.Content))
	for _, item := range result.Content {
		switch {
		case item.Text != "":
			parts = append(parts, item.Text)
		case item.Type == "image":
			parts = append(parts, "[图片] "+item.MimeType)
		case item.Type == "audio":
			parts = append(parts, "[音频] "+item.MimeType)
		case item.Type == "resource":
			parts = append(parts, "[资源]")
		}
	}
	if text := strings.Join(parts, "\n"); text != "" {
		return text
	}
	// 只有结构化结果时保留原始 JSON，交给前端按需解析
	return rawJSONText(result.StructuredContent)
}

// rawJSONText 把 json.RawMessage 转成可读文本：null / 空 给空串，
// JSON 字符串去引号，结构化内容保留原文。
func rawJSONText(raw json.RawMessage) string {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return ""
	}
	if unquoted, err := strconv.Unquote(text); err == nil {
		return unquoted
	}
	return text
}

// ReasoningItemDetail 对应 type: "reasoning" (深度思考/思维链节点)
type ReasoningItemDetail struct {
	Summary []string `json:"summary"`
	Content []any    `json:"content"`
}

// CommandExecutionItemDetail 对应 type: "commandExecution" (Shell 命令执行节点)
type CommandExecutionItemDetail struct {
	Status           string          `json:"status"` // "inProgress" | "completed"
	Command          string          `json:"command"`
	AggregatedOutput string          `json:"aggregatedOutput,omitempty"`
	Cwd              string          `json:"cwd"`
	ProcessID        string          `json:"processId,omitempty"`
	Source           string          `json:"source,omitempty"`
	ExitCode         int64           `json:"exitCode,omitempty"`
	DurationMs       int64           `json:"durationMs,omitempty"`
	CommandActions   []CommandAction `json:"commandActions,omitempty"`
	PluginID         string          `json:"pluginId,omitempty"`
	ScriptPath       string          `json:"scriptPath,omitempty"`
}

// CommandAction 描述命令行为（如解析出来的具体指令）
type CommandAction struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// AgentMessageItemDetail 对应 type: "agentMessage" (Agent 最终消息回复节点)
type AgentMessageItemDetail struct {
	Text           string  `json:"text"`
	Phase          *string `json:"phase,omitempty"`
	MemoryCitation *any    `json:"memoryCitation,omitempty"`
	Delivery       *any    `json:"delivery,omitempty"`
}

// WebSearchItemDetail 对应内部的 item 对象
type WebSearchItemDetail struct {
	Query   string          `json:"query"`
	Action  WebSearchAction `json:"action"`
	Results json.RawMessage `json:"results"` // 当前为 null，用 json.RawMessage 或 any 兼容未来扩展
}

// WebSearchAction 对应动作与查询数组
type WebSearchAction struct {
	Type    string   `json:"type"`            // 例如: "search"
	Query   *string  `json:"query,omitempty"` // 指针兼容 null 值
	Queries []string `json:"queries"`
}

// ==========================================
// Reasoning 思考流式事件 (summaryPartAdded & summaryTextDelta)
// ==========================================

// ReasoningPartAddedEvent item/reasoning/summaryPartAdded
type ReasoningPartAddedEvent struct {
	baseNotificationEvent
	ItemID       string `json:"itemId"`
	SummaryIndex int    `json:"summaryIndex"`
}

// TextDeltaEvent item/reasoning/summaryTextDelta，item/agentMessage/delta  (打字机吐出思考过程)
type TextDeltaEvent struct {
	baseNotificationEvent
	DeltaType    EventItemType `json:"-"`
	ItemID       string        `json:"itemId"`
	Delta        string        `json:"delta"` // 增量文本 (如 " user is asking what")
	SummaryIndex int           `json:"summaryIndex"`
}

// BaseApprovalEvent 基本审批字段
type BaseApprovalEvent struct {
	baseNotificationEvent
	Reason string `json:"reason"`
	CommandApprovalEvent
}

type CommandApprovalEvent struct {
	ItemID         string   `json:"itemId"`
	Command        []string `json:"command"`
	Cwd            string   `json:"cwd"`
	CommandActions []any    `json:"commandActions"`
}

type ApprovalDecision string

const (
	ApprovalAccept ApprovalDecision = "accept"  //通过
	ApprovalReject ApprovalDecision = "decline" //拒绝
)

type ApprovalResult struct {
	Decision ApprovalDecision `json:"decision"`
}
