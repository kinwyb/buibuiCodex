package types

// TokenUsage 一次对话的 token 用量快照。
//
// 语义与 codex 的 thread/tokenUsage/updated 通知一致：
//   - Total 是当前 thread 的累计用量，前端的「本对话共消耗」取它
//   - Last 是最近一次交互（一轮 turn）的用量，前端的「本条回答消耗」取它
//
// 两个字段都带上完整拆分，便于前端分别展示输入 / 输出 / 缓存命中。
type TokenUsage struct {
	Total TokenDetail `json:"total"`
	Last  TokenDetail `json:"last"`
	// ModelContextWindow 模型最大上下文窗口，0 表示上游未提供。
	// 前端可用 Total.InputTokens 与之相比，展示上下文占用率。
	ModelContextWindow int64 `json:"modelContextWindow,omitempty"`
}

// TokenDetail 单轮或累计的 token 拆分。
type TokenDetail struct {
	// TotalTokens 总 token 数
	TotalTokens int64 `json:"totalTokens"`
	// InputTokens 输入 token 数
	InputTokens int64 `json:"inputTokens"`
	// CachedInputTokens 命中缓存的输入 token
	CachedInputTokens int64 `json:"cachedInputTokens"`
	// CacheWriteInputTokens 写入缓存的输入 token
	CacheWriteInputTokens int64 `json:"cacheWriteInputTokens"`
	// OutputTokens 输出 token 数
	OutputTokens int64 `json:"outputTokens"`
	// ReasoningOutputTokens 深度思考消耗的 token
	ReasoningOutputTokens int64 `json:"reasoningOutputTokens"`
}
