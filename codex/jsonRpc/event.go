package jsonRpc

import "encoding/json"

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

// ToolCallItemDetail 函数调用
type ToolCallItemDetail struct {
	CallId    string          `json:"callId"`
	Namespace string          `json:"namespace"`
	Tool      string          `json:"tool"`
	Arguments map[string]any  `json:"arguments"`
	Result    json.RawMessage `json:"result"`
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
