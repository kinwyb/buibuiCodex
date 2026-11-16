// 本文件测试 *.page 大数据回执的落盘：预览条数、meta、文件内容、失败回退与过期清理。

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/core/pathmap"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// pageResult 构造一与前端 table.ts 一致的 page 回执。
func pageResult(t *testing.T, rows int) json.RawMessage {
	t.Helper()
	items := make([]map[string]any, 0, rows)
	for i := 0; i < rows; i++ {
		items = append(items, map[string]any{
			"index": i,
			"rowId": fmt.Sprintf("R%03d", i),
			"Memo":  strings.Repeat("数据", 120),
		})
	}
	data, err := json.Marshal(map[string]any{
		"name":    "data",
		"offset":  0,
		"limit":   rows,
		"total":   278,
		"count":   rows,
		"hasMore": true,
		"sorts":   []any{},
		"fields":  []string{"Memo"},
		"note":    "示例备注",
		"rows":    items,
	})
	if err != nil {
		t.Fatalf("构造 page 回执失败: %v", err)
	}
	return data
}

// pageTool 组一个带临时 /tmp 映射的界面工具。
func pageTool(resp *protocol.UIResponseData, tmpDir string) *UIActTool {
	bridge := testUIBridge(&fakeUITransport{resp: resp})
	bridge.SetAgentTmpDir(func(agentName string) (string, bool) {
		if agentName != "agentA" {
			return "", false
		}
		return tmpDir, true
	})
	return &UIActTool{bridge: bridge}
}

func pageState() *types.State {
	return &types.State{
		SessionID: "ws_u1_c1",
		Input:     &types.InputMessage{AgentName: "agentA"},
	}
}

func resultOf(t *testing.T, content string) map[string]json.RawMessage {
	t.Helper()
	_, body, ok := strings.Cut(content, " result=")
	if !ok {
		t.Fatalf("回执缺少 result: %s", content)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("解析 result 失败: %v", err)
	}
	return got
}

func TestUIActSpillsLargePageRows(t *testing.T) {
	tmpDir := t.TempDir()
	tool := pageTool(&protocol.UIResponseData{OK: true, Rev: 2, Result: pageResult(t, 30)}, tmpDir)

	msg, err := tool.Execute(context.Background(), pageState(), map[string]any{"cmd": "data.page"})
	if err != nil {
		t.Fatalf("执行 page 命令失败: %v", err)
	}
	if !strings.HasPrefix(msg.Content, "ok rev=2 result=") {
		t.Fatalf("回执头不符合预期: %s", msg.Content)
	}
	got := resultOf(t, msg.Content)

	var rows []json.RawMessage
	if err := json.Unmarshal(got["rows"], &rows); err != nil {
		t.Fatalf("解析预览 rows 失败: %v", err)
	}
	if len(rows) != uiPagePreviewRows {
		t.Fatalf("预览应为 %d 行，实际 %d", uiPagePreviewRows, len(rows))
	}
	if string(got["count"]) != "30" || string(got["total"]) != "278" {
		t.Fatalf("分页字段应原样保留，实际 count=%s total=%s", got["count"], got["total"])
	}

	var meta struct {
		Total    int    `json:"total"`
		Shown    int    `json:"shown"`
		File     string `json:"file"`
		FileHint string `json:"fileHint"`
	}
	if err := json.Unmarshal(got["meta"], &meta); err != nil {
		t.Fatalf("解析 meta 失败: %v", err)
	}
	if meta.Total != 30 || meta.Shown != uiPagePreviewRows {
		t.Fatalf("meta 计数不对: %+v", meta)
	}
	if !strings.HasPrefix(meta.File, uiPageContainerDir+"/") {
		t.Fatalf("文件路径应是容器路径，实际 %s", meta.File)
	}
	if !strings.Contains(meta.FileHint, "data.page") ||
		!strings.Contains(meta.FileHint, "顶层是 JSON 数组") ||
		!strings.Contains(meta.FileHint, "不包含 result 外层") ||
		!strings.Contains(meta.FileHint, "不要一次性读取完整文件") {
		t.Fatalf("fileHint 应说明文件内容与读法，实际 %s", meta.FileHint)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "ui_data", filepath.Base(meta.File)))
	if err != nil {
		t.Fatalf("读取落盘文件失败: %v", err)
	}
	var all []json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		t.Fatalf("落盘内容应是 rows 数组: %v", err)
	}
	if len(all) != 30 {
		t.Fatalf("文件应保留完整 rows，实际 %d 行", len(all))
	}
}

func TestUIActKeepsSmallPageRowsInline(t *testing.T) {
	tool := pageTool(&protocol.UIResponseData{OK: true, Rev: 2, Result: pageResult(t, 2)}, t.TempDir())

	msg, err := tool.Execute(context.Background(), pageState(), map[string]any{"cmd": "data.page"})
	if err != nil {
		t.Fatalf("执行 page 命令失败: %v", err)
	}
	if strings.Contains(msg.Content, `"meta"`) {
		t.Fatalf("小结果不应落盘: %s", msg.Content)
	}
	got := resultOf(t, msg.Content)
	var rows []json.RawMessage
	if err := json.Unmarshal(got["rows"], &rows); err != nil || len(rows) != 2 {
		t.Fatalf("小结果应原样内联，rows=%d err=%v", len(rows), err)
	}
}

func TestUIActDoesNotSpillNonPageCommands(t *testing.T) {
	tool := pageTool(&protocol.UIResponseData{OK: true, Rev: 2, Result: pageResult(t, 30)}, t.TempDir())

	msg, err := tool.Execute(context.Background(), pageState(), map[string]any{"cmd": "data.filter.set"})
	if err != nil {
		t.Fatalf("执行命令失败: %v", err)
	}
	if strings.Contains(msg.Content, `"meta"`) {
		t.Fatalf("非 *.page 命令不应落盘: %s", msg.Content)
	}
}

func TestUIActFallsBackToInlineWhenSpillFails(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	tool := pageTool(&protocol.UIResponseData{OK: true, Rev: 2, Result: pageResult(t, 30)}, blocked)

	msg, err := tool.Execute(context.Background(), pageState(), map[string]any{"cmd": "data.page"})
	if err != nil {
		t.Fatalf("落盘失败不应让工具报错: %v", err)
	}
	if strings.Contains(msg.Content, `"meta"`) {
		t.Fatalf("落盘失败应回退内联: %s", msg.Content)
	}
	got := resultOf(t, msg.Content)
	var rows []json.RawMessage
	if err := json.Unmarshal(got["rows"], &rows); err != nil || len(rows) != 30 {
		t.Fatalf("回退后应完整内联，rows=%d err=%v", len(rows), err)
	}
}

func TestCleanupPageDataRemovesExpiredFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ui_data")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	oldFile := filepath.Join(dir, "old.json")
	freshFile := filepath.Join(dir, "fresh.json")
	for _, name := range []string{oldFile, freshFile} {
		if err := os.WriteFile(name, []byte("[]"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	expired := time.Now().Add(-uiPageDataTTL - time.Minute)
	if err := os.Chtimes(oldFile, expired, expired); err != nil {
		t.Fatal(err)
	}

	cleanupPageData(dir)
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatalf("超过 24 小时的文件应被清理")
	}
	if _, err := os.Stat(freshFile); err != nil {
		t.Fatalf("未过期文件不应被清理: %v", err)
	}
}

func TestManagerAgentTmpDir(t *testing.T) {
	m := NewManager(nil, nil)
	m.pathMapper["agentA"] = pathmap.New(&pathmap.Config{Mappings: []pathmap.PathMapping{
		{Host: "/host/agentA/workspace", Container: "/workspace"},
		{Host: "/host/agentA/tmp", Container: "/tmp"},
	}})

	got, ok := m.AgentTmpDir("agentA")
	if !ok || got != "/host/agentA/tmp" {
		t.Fatalf("应返回 /tmp 的宿主机目录，实际 %q ok=%v", got, ok)
	}
	if _, ok := m.AgentTmpDir("missing"); ok {
		t.Fatalf("未知 agent 应返回 false")
	}
}
