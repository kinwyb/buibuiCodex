package cdm

import (
	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
)

// tokenUsageOf 把上游的 token 统计转成对外事件载荷。
// 上游字段与 types.TokenDetail 一一对应，这里只做结构搬运，不做单位换算。
func tokenUsageOf(usage jsonRpc.ThreadTokenUsage) *types.TokenUsage {
	return &types.TokenUsage{
		Total:              tokenDetailOf(usage.Total),
		Last:               tokenDetailOf(usage.Last),
		ModelContextWindow: usage.ModelContextWindow,
	}
}

// tokenDetailOf 单条用量明细转换
func tokenDetailOf(detail jsonRpc.TokenDetail) types.TokenDetail {
	return types.TokenDetail{
		TotalTokens:           detail.TotalTokens,
		InputTokens:           detail.InputTokens,
		CachedInputTokens:     detail.CachedInputTokens,
		CacheWriteInputTokens: detail.CacheWriteInputTokens,
		OutputTokens:          detail.OutputTokens,
		ReasoningOutputTokens: detail.ReasoningOutputTokens,
	}
}

// dbTokenUsage 落库用的用量明细转换
func dbTokenUsage(detail jsonRpc.TokenDetail) db.TokenUsage {
	return db.TokenUsage{
		TotalTokens:           detail.TotalTokens,
		InputTokens:           detail.InputTokens,
		CachedInputTokens:     detail.CachedInputTokens,
		CacheWriteInputTokens: detail.CacheWriteInputTokens,
		OutputTokens:          detail.OutputTokens,
		ReasoningOutputTokens: detail.ReasoningOutputTokens,
	}
}
