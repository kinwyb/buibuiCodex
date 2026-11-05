package types

import (
	"encoding/json"
	"strings"
)

const (
	uiEventsOpen  = "<ui_events>"
	uiEventsClose = "</ui_events>"
)

// UIEventNotice 页面主动上报的一条界面变化。
type UIEventNotice struct {
	TS      int64          `json:"ts"`
	View    string         `json:"view"`
	Event   string         `json:"event"`
	Payload map[string]any `json:"payload,omitempty"`
}

// UIEventBatch 多条界面变化的稳定文本载荷。
type UIEventBatch struct {
	Version int             `json:"v"`
	Events  []UIEventNotice `json:"events"`
}

// ParseUIEventContent 解析消息开头的事件批次，并返回其后的真实用户正文。
//
// 标记不完整、版本不支持或没有有效事件时返回 ok=false，调用方按普通用户输入处理。
func ParseUIEventContent(content string) (batch UIEventBatch, userText string, ok bool) {
	text := strings.TrimLeft(content, " \t\r\n")
	if !strings.HasPrefix(text, uiEventsOpen) {
		return UIEventBatch{}, "", false
	}
	closeAt := strings.Index(text[len(uiEventsOpen):], uiEventsClose)
	if closeAt < 0 {
		return UIEventBatch{}, "", false
	}
	jsonEnd := len(uiEventsOpen) + closeAt
	if err := json.Unmarshal([]byte(text[len(uiEventsOpen):jsonEnd]), &batch); err != nil {
		return UIEventBatch{}, "", false
	}
	if batch.Version != 1 || len(batch.Events) == 0 {
		return UIEventBatch{}, "", false
	}
	events := batch.Events[:0]
	for _, event := range batch.Events {
		if event.Event != "" && event.View != "" {
			events = append(events, event)
		}
	}
	if len(events) == 0 {
		return UIEventBatch{}, "", false
	}
	batch.Events = events
	return batch, strings.TrimSpace(text[jsonEnd+len(uiEventsClose):]), true
}

// HasUIEventContent 是否是带界面事件块的消息（纯事件或事件 + 用户正文）。
func HasUIEventContent(content string) bool {
	_, _, ok := ParseUIEventContent(content)
	return ok
}

// UIEventToolParams 给前端工具步骤展示的紧凑参数。
func UIEventToolParams(batch UIEventBatch) string {
	raw, err := json.Marshal(batch)
	if err != nil {
		return ""
	}
	return string(raw)
}
