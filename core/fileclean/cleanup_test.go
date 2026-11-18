package fileclean

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanRemovesExpiredRecursively(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, "2026-11-15")
	oldSub := filepath.Join(oldDir, "nested")
	freshDir := filepath.Join(root, "2026-11-16")
	for _, dir := range []string{oldSub, freshDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	oldFile := filepath.Join(oldSub, "old.json")
	oldRootFile := filepath.Join(root, "upload.pdf")
	freshFile := filepath.Join(freshDir, "fresh.json")
	for _, name := range []string{oldFile, oldRootFile, freshFile} {
		if err := os.WriteFile(name, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	expired := time.Now().Add(-DefaultTTL - time.Minute)
	for _, name := range []string{oldFile, oldRootFile} {
		if err := os.Chtimes(name, expired, expired); err != nil {
			t.Fatal(err)
		}
	}

	if removed := Clean(root, DefaultTTL, time.Now()); removed != 2 {
		t.Fatalf("应删除 2 个过期文件，实际 %d", removed)
	}
	for _, name := range []string{oldFile, oldRootFile} {
		if _, err := os.Stat(name); !os.IsNotExist(err) {
			t.Fatalf("超过 24 小时的文件应被删除: %s", name)
		}
	}
	if _, err := os.Stat(oldSub); !os.IsNotExist(err) {
		t.Fatalf("清空后的子目录应被删除")
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("清空后的日期目录应被删除")
	}
	if _, err := os.Stat(freshFile); err != nil {
		t.Fatalf("未过期文件不应被删除: %v", err)
	}
	if _, err := os.Stat(freshDir); err != nil {
		t.Fatalf("仍有文件的目录应保留: %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("根目录应保留: %v", err)
	}
}
