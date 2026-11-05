package types

import (
	"strings"
	"testing"
)

func TestParseUIEventContent(t *testing.T) {
	content := `<ui_events>{"v":1,"events":[{"ts":1,"view":"V1","event":"selection.change","payload":{"table":"data","rowId":"9"}}]}</ui_events>
请分析当前选中行`
	batch, userText, ok := ParseUIEventContent(content)
	if !ok {
		t.Fatal("合法事件批次应解析成功")
	}
	if len(batch.Events) != 1 || batch.Events[0].Event != "selection.change" {
		t.Fatalf("事件内容不符合预期: %+v", batch)
	}
	if userText != "请分析当前选中行" {
		t.Fatalf("用户正文提取错误: %q", userText)
	}
}

func TestParseUIEventContentRejectsBrokenMarker(t *testing.T) {
	for _, content := range []string{
		"",
		"普通用户消息",
		`<ui_events>{"v":1,"events":[]}</ui_events>`,
		`<ui_events>{"v":2,"events":[{"view":"V1","event":"x"}]}</ui_events>`,
		`<ui_events>not-json</ui_events>`,
	} {
		if _, _, ok := ParseUIEventContent(content); ok {
			t.Fatalf("不应把 %q 识别为事件批次", content)
		}
	}
}

func TestUIEventToolParamsKeepsEscapedPayload(t *testing.T) {
	content := `<ui_events>{"v":1,"events":[{"ts":1,"view":"V1","event":"x","payload":{"text":"\u003c/ui_events\u003e"}}]}</ui_events>`
	batch, _, ok := ParseUIEventContent(content)
	if !ok {
		t.Fatal("转义后的标记不应截断事件块")
	}
	params := UIEventToolParams(batch)
	if !strings.Contains(params, "ui_events") {
		t.Fatalf("工具参数应保留事件内容: %s", params)
	}
}
