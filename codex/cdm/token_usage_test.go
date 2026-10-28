package cdm

import (
	"testing"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/core/types"
)

// Test_codexEventParse_tokenUsage 确认 thread/tokenUsage/updated 会被转成
// 前端可直接消费的 token_usage 事件，而不是只落库后在界面上无从展示。
func Test_codexEventParse_tokenUsage(t *testing.T) {
	// 字段取自线上真实报文结构：total 是 thread 累计，last 是本轮用量
	raw := `{
		"threadId":"th_1",
		"turnId":"tu_1",
		"tokenUsage":{
			"total":{"totalTokens":1000,"inputTokens":800,"cachedInputTokens":300,"cacheWriteInputTokens":20,"outputTokens":200,"reasoningOutputTokens":50},
			"last":{"totalTokens":120,"inputTokens":100,"cachedInputTokens":40,"cacheWriteInputTokens":0,"outputTokens":20,"reasoningOutputTokens":10},
			"modelContextWindow":200000
		}
	}`

	state := types.NewState(&types.InputMessage{ID: "req_1", Session: "ws_user_chat", Channel: "ws", ChatID: "chat"})
	var got *types.Event
	state.EventHandler = func(event *types.Event) { got = event }

	notification := &codex.Event{
		Type:    codex.EventTypeNotification,
		Method:  string(codex.ThreadTokenUsageUpdated),
		RawData: []byte(raw),
	}
	(&Agent{}).codexEventParse(state, notification)

	if got == nil {
		t.Fatal("未产出 token_usage 事件")
	}
	if got.Type != types.EventTokenUsage {
		t.Fatalf("事件类型不符：want %s, got %s", types.EventTokenUsage, got.Type)
	}
	if got.EventID != "tu_1" {
		t.Fatalf("EventID 应为 turnID（前端据此覆盖同一轮的多次推送）：%q", got.EventID)
	}
	if got.Usage == nil {
		t.Fatal("缺少 token 用量载荷")
	}
	usage := got.Usage
	if usage.ModelContextWindow != 200000 {
		t.Fatalf("上下文窗口不符: %d", usage.ModelContextWindow)
	}
	if usage.Total.TotalTokens != 1000 || usage.Total.InputTokens != 800 || usage.Total.OutputTokens != 200 {
		t.Fatalf("累计用量不符: %+v", usage.Total)
	}
	if usage.Last.TotalTokens != 120 || usage.Last.CachedInputTokens != 40 || usage.Last.ReasoningOutputTokens != 10 {
		t.Fatalf("本轮用量不符: %+v", usage.Last)
	}
}

// Test_codexEventParse_tokenUsageEmptyContent 确认该事件不会污染正文：
// 前端只按 usage 更新用量面板，因此 Content 必须为空、Delta 恒为 false。
func Test_codexEventParse_tokenUsageEmptyContent(t *testing.T) {
	raw := `{"threadId":"th_1","turnId":"tu_2","tokenUsage":{"total":{"totalTokens":10},"last":{"totalTokens":10}}}`
	state := types.NewState(&types.InputMessage{ID: "req_1", Session: "ws_user_chat", Channel: "ws"})
	var got *types.Event
	state.EventHandler = func(event *types.Event) { got = event }

	(&Agent{}).codexEventParse(state, &codex.Event{
		Type:    codex.EventTypeNotification,
		Method:  string(codex.ThreadTokenUsageUpdated),
		RawData: []byte(raw),
	})

	if got == nil || got.Message == nil {
		t.Fatalf("事件或载荷缺失: %+v", got)
	}
	if got.Message.Content != "" || got.Message.ReasoningContent != "" || got.Message.IsDetla {
		t.Fatalf("token_usage 不应携带文本或标记为增量: %+v", got.Message)
	}
}
