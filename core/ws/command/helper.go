// 本文件存放命令间共用的小工具。

package command

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
)

// mediaTypeByName 按扩展名判断媒体类型，图片/音频单独归类，其余按文档处理
func mediaTypeByName(name string) types.MediaType {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico", ".heic":
		return types.MediaTypeImage
	case ".mp3", ".wav", ".m4a", ".aac", ".ogg", ".flac", ".amr", ".silk":
		return types.MediaTypeAudio
	default:
		return types.MediaTypeFile
	}
}

// truncateRunes 按字符数截断，超出时补省略号
func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "…"
}

// millis 转毫秒时间戳，零值返回 0 而不是负数
func millis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

// turnUsageOf 把库里一轮的用量还原成协议结构：只有 Last 与上下文窗口有值。
// 没有任何用量记录时返回 nil，前端据此不展示「0 tokens」。
func turnUsageOf(turn db.SessionTurn) *types.TokenUsage {
	usage := types.TokenUsage{
		Last:               tokenDetailOf(turn.TokenUsage),
		ModelContextWindow: turn.ModelContextWindow,
	}
	if usage.Last == (types.TokenDetail{}) && usage.ModelContextWindow == 0 {
		return nil
	}
	return &usage
}

// threadUsageOf 把库里一个 thread 的累计用量还原成协议结构，取 Total。
// 未产生过用量时返回 nil。
func threadUsageOf(thread db.SessionThread) *types.TokenUsage {
	total := tokenDetailOf(thread.TokenUsage)
	if total == (types.TokenDetail{}) {
		return nil
	}
	return &types.TokenUsage{Total: total}
}

// tokenDetailOf 库表字段 → 协议明细
func tokenDetailOf(usage db.TokenUsage) types.TokenDetail {
	return types.TokenDetail{
		TotalTokens:           usage.TotalTokens,
		InputTokens:           usage.InputTokens,
		CachedInputTokens:     usage.CachedInputTokens,
		CacheWriteInputTokens: usage.CacheWriteInputTokens,
		OutputTokens:          usage.OutputTokens,
		ReasoningOutputTokens: usage.ReasoningOutputTokens,
	}
}
