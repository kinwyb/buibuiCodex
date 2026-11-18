package jsonRpc

import (
	"encoding/json"
	"testing"
)

// 真实 codex 回显：userMessage 没有顶层 text，正文与附件行都在 content 数组里。
// 正文拼回 Text，文件行要解析成结构化附件，不能再混在文本里。
func TestItemLifecycleUserMessageContentArray(t *testing.T) {
	raw := `{"item":{"type":"userMessage","id":"u1","clientId":null,"content":[{"type":"text","text":"分析这个文件内容","text_elements":[]},{"type":"text","text":"文件[报价汇总.xlsx]地址:/tmp/a1759e3e_报价汇总.xlsx","text_elements":[]}],"startedAtMs":1794902406526,"completedAtMs":null},"threadId":"th","turnId":"tu","startedAtMs":1794902406526}`
	var evt ItemLifecycleEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if evt.Item.Text != "分析这个文件内容" {
		t.Fatalf("Text 应只留用户正文: %q", evt.Item.Text)
	}
	if len(evt.Item.UserAttachments) != 1 {
		t.Fatalf("文件行应解析成一个附件: %#v", evt.Item.UserAttachments)
	}
	att := evt.Item.UserAttachments[0]
	if att.Type != "file" || att.Name != "报价汇总.xlsx" || att.URL != "/tmp/a1759e3e_报价汇总.xlsx" {
		t.Fatalf("附件文件名与地址不符: %#v", att)
	}
}

// 旧形态（顶层 text）保持原样，不被 content 解析覆盖
func TestItemLifecycleUserMessageTopLevelTextKept(t *testing.T) {
	raw := `{"item":{"id":"u1","type":"userMessage","text":"旧形态正文"}}`
	var evt ItemLifecycleEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if evt.Item.Text != "旧形态正文" {
		t.Fatalf("顶层 text 应保留: %q", evt.Item.Text)
	}
}

// 图片段解析成图片附件，空 text 段忽略，正文不受影响
func TestItemLifecycleUserMessageImageAttachment(t *testing.T) {
	raw := `{"item":{"id":"u1","type":"userMessage","content":[{"type":"localImage","path":"/tmp/a.png"},{"type":"text","text":"看下这张图"},{"type":"text","text":""}]}}`
	var evt ItemLifecycleEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if evt.Item.Text != "看下这张图" {
		t.Fatalf("只应拼接非空 text 段: %q", evt.Item.Text)
	}
	if len(evt.Item.UserAttachments) != 1 {
		t.Fatalf("localImage 应解析成一个附件: %#v", evt.Item.UserAttachments)
	}
	att := evt.Item.UserAttachments[0]
	if att.Type != "localImage" || att.Path != "/tmp/a.png" {
		t.Fatalf("图片附件路径不符: %#v", att)
	}
}

// reasoning 的 content 仍走默认映射，不受 userMessage 补齐逻辑影响
func TestItemLifecycleReasoningContentUntouched(t *testing.T) {
	raw := `{"item":{"id":"r1","type":"reasoning","summary":["摘要"],"content":[{"type":"text","text":"思考分段"}]}}`
	var evt ItemLifecycleEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if evt.Item.Text != "" {
		t.Fatalf("reasoning 不应填充 Text: %q", evt.Item.Text)
	}
	if len(evt.Item.Content) != 1 {
		t.Fatalf("reasoning content 应原样保留: %#v", evt.Item.Content)
	}
}
