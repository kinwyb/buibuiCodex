// 本文件测试 MCP 临时文件的 24 小时回收：过期文件删除、未过期保留、空日期目录清理。

package mcp

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanTmpFilesRemovesExpiredAndKeepsFresh(t *testing.T) {
	dir := t.TempDir()
	oldDir := filepath.Join(dir, "2026-11-15")
	freshDir := filepath.Join(dir, "2026-11-16")
	for _, sub := range []string{oldDir, freshDir} {
		if err := os.MkdirAll(sub, 0755); err != nil {
			t.Fatal(err)
		}
	}
	oldFile := filepath.Join(oldDir, "1.json")
	freshFile := filepath.Join(freshDir, "2.json")
	for _, name := range []string{oldFile, freshFile} {
		if err := os.WriteFile(name, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	expired := time.Now().Add(-tmpFileTTL - time.Minute)
	if err := os.Chtimes(oldFile, expired, expired); err != nil {
		t.Fatal(err)
	}

	cleanTmpFiles(dir, tmpFileTTL, time.Now())

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatalf("超过 24 小时的文件应被删除")
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("清空后的日期目录应被删除")
	}
	if _, err := os.Stat(freshFile); err != nil {
		t.Fatalf("未过期文件不应被删除: %v", err)
	}
	if _, err := os.Stat(freshDir); err != nil {
		t.Fatalf("仍有文件的日期目录应保留: %v", err)
	}
}
