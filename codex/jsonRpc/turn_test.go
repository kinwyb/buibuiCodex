package jsonRpc

import "testing"

// 文件行格式正反解析要闭环：buildInput 写进去的，回显解析要原样还原
func TestFileInputTextRoundTrip(t *testing.T) {
	name, url := "2026-07-01_2026-07-30报价汇总.xlsx", "/tmp/a1759e3e_2026-07-01_2026-07-30报价汇总.xlsx"
	gotName, gotURL, ok := ParseFileInputText(FileInputText(name, url))
	if !ok || gotName != name || gotURL != url {
		t.Fatalf("往返解析不符: name=%q url=%q ok=%v", gotName, gotURL, ok)
	}
}

// 普通正文不能被误认成文件行
func TestParseFileInputTextRejectsPlainText(t *testing.T) {
	for _, text := range []string{"分析这个文件内容", "文件[没有地址", "文件[]地址:/tmp/a", "文件[a]地址:"} {
		if _, _, ok := ParseFileInputText(text); ok {
			t.Fatalf("不应识别为文件行: %q", text)
		}
	}
}
