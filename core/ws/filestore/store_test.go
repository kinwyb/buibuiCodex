package filestore

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newTestStore 建一个临时目录的存储，并把时钟固定住便于验证过期
func newTestStore(t *testing.T, ttl time.Duration) (*Store, *time.Time) {
	t.Helper()
	now := time.Now()
	s := New(t.TempDir(), ttl)
	s.now = func() time.Time { return now }
	return s, &now
}

func TestSaveAndGet(t *testing.T) {
	s, _ := newTestStore(t, time.Hour)
	content := []byte("hello ws file")

	item, err := s.Save("alice", "报告 (1).txt", bytes.NewReader(content), 0)
	if err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	if item.ID == "" {
		t.Fatal("应生成 fileID")
	}
	if item.Size != int64(len(content)) {
		t.Fatalf("大小不符: %d", item.Size)
	}
	if item.Name != "报告 (1).txt" {
		t.Fatalf("文件名应保留中文与括号，实际 %q", item.Name)
	}

	got, ok := s.Get(item.ID)
	if !ok {
		t.Fatal("应能按 ID 取回")
	}
	data, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatalf("读文件失败: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Fatalf("内容不符: %q", data)
	}

	// 文件必须落在 root 下，避免写到临时目录之外
	if !strings.HasPrefix(got.Path, s.Root()+string(filepath.Separator)) {
		t.Fatalf("路径应在 root 内: %s", got.Path)
	}
}

func TestSaveTooLarge(t *testing.T) {
	s, _ := newTestStore(t, time.Hour)

	_, err := s.Save("alice", "big.bin", bytes.NewReader(make([]byte, 2048)), 1024)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("应返回 ErrTooLarge，实际 %v", err)
	}
	// 超限不留半成品
	entries, _ := os.ReadDir(s.Root())
	for _, day := range entries {
		files, _ := os.ReadDir(filepath.Join(s.Root(), day.Name()))
		if len(files) > 0 {
			t.Fatalf("超限后应清理临时目录，仍残留 %d 个", len(files))
		}
	}
}

func TestSaveExactLimit(t *testing.T) {
	s, _ := newTestStore(t, time.Hour)
	if _, err := s.Save("alice", "exact.bin", bytes.NewReader(make([]byte, 1024)), 1024); err != nil {
		t.Fatalf("恰好等于上限应允许: %v", err)
	}
}

func TestOwnerCheck(t *testing.T) {
	s, _ := newTestStore(t, time.Hour)
	item, err := s.Save("alice", "a.txt", strings.NewReader("x"), 0)
	if err != nil {
		t.Fatalf("保存失败: %v", err)
	}

	if _, err := s.GetOwned(item.ID, "alice"); err != nil {
		t.Fatalf("本人应可取: %v", err)
	}
	if _, err := s.GetOwned(item.ID, "bob"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("他人应被拒绝，实际 %v", err)
	}
	// owner 为空表示内部调用，不做归属校验
	if _, err := s.GetOwned(item.ID, ""); err != nil {
		t.Fatalf("内部调用应放行: %v", err)
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"../../etc/passwd": "passwd",
		`C:\tmp\a.txt`:     "a.txt",
		"":                 "file",
		"..":               "file",
		"正常文件.pdf":         "正常文件.pdf",
	}
	for in, want := range cases {
		if got := sanitizeName(in); got != want {
			t.Errorf("sanitizeName(%q) = %q，期望 %q", in, got, want)
		}
	}

	long := strings.Repeat("a", 200) + ".txt"
	if got := sanitizeName(long); len(got) > 120 || !strings.HasSuffix(got, ".txt") {
		t.Errorf("超长文件名应保留后缀并截断，实际长度 %d: %q", len(got), got)
	}
}

func TestAdoptKeepsOriginal(t *testing.T) {
	s, _ := newTestStore(t, time.Hour)
	src := filepath.Join(t.TempDir(), "agent-out.csv")
	if err := os.WriteFile(src, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatalf("准备文件失败: %v", err)
	}

	item, err := s.Adopt("alice", src, "")
	if err != nil {
		t.Fatalf("登记失败: %v", err)
	}
	if item.Name != "agent-out.csv" {
		t.Fatalf("文件名应取路径末段，实际 %q", item.Name)
	}
	if item.Mime == "" {
		t.Fatal("应按后缀推断出 MIME")
	}

	// 删除登记项不能动原文件：它是 agent 工作目录里的产物
	if err := s.Delete(item.ID); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("原文件不应被删除: %v", err)
	}
}

func TestGetRemovesVanishedFile(t *testing.T) {
	s, _ := newTestStore(t, time.Hour)
	item, err := s.Save("alice", "moved.txt", strings.NewReader("x"), 0)
	if err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	// 模拟 core.Manager 把附件移到 agent 工作区：原路径消失
	if err := os.Remove(item.Path); err != nil {
		t.Fatalf("删除模拟失败: %v", err)
	}
	if _, ok := s.Get(item.ID); ok {
		t.Fatal("文件已不存在时不应返回记录")
	}
}

func TestExpireAndGC(t *testing.T) {
	s, now := newTestStore(t, time.Minute)
	item, err := s.Save("alice", "tmp.txt", strings.NewReader("x"), 0)
	if err != nil {
		t.Fatalf("保存失败: %v", err)
	}

	*now = now.Add(2 * time.Minute)
	if _, ok := s.Get(item.ID); ok {
		t.Fatal("过期后应取不到")
	}

	item2, err := s.Save("alice", "tmp2.txt", strings.NewReader("y"), 0)
	if err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	*now = now.Add(2 * time.Minute)
	if n := s.GC(); n != 1 {
		t.Fatalf("应回收 1 个，实际 %d", n)
	}
	if _, err := os.Stat(item2.Path); !os.IsNotExist(err) {
		t.Fatal("过期文件应被删除")
	}
}

func TestSameNameNoClobber(t *testing.T) {
	s, _ := newTestStore(t, time.Hour)

	a, err := s.Save("alice", "报告.xlsx", strings.NewReader("first"), 0)
	if err != nil {
		t.Fatalf("第一次保存失败: %v", err)
	}
	b, err := s.Save("alice", "报告.xlsx", strings.NewReader("second"), 0)
	if err != nil {
		t.Fatalf("第二次保存失败: %v", err)
	}

	// 展示名相同，落盘路径必须不同：附件随后会被移到 agent 工作区的 /tmp，
	// 那里按文件名铺平，撞名就会互相覆盖
	if a.Name != b.Name {
		t.Fatalf("展示名应保持一致: %q / %q", a.Name, b.Name)
	}
	if a.Path == b.Path {
		t.Fatal("同名文件不能落到同一个路径")
	}
	data, err := os.ReadFile(a.Path)
	if err != nil {
		t.Fatalf("读第一份失败: %v", err)
	}
	if string(data) != "first" {
		t.Fatalf("第一份内容被覆盖: %q", data)
	}
}

func TestList(t *testing.T) {
	s, _ := newTestStore(t, time.Hour)
	if _, err := s.Save("alice", "a.txt", strings.NewReader("a"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save("bob", "b.txt", strings.NewReader("b"), 0); err != nil {
		t.Fatal(err)
	}

	if got := len(s.List("alice")); got != 1 {
		t.Fatalf("alice 应只有 1 个文件，实际 %d", got)
	}
	if got := len(s.List("")); got != 2 {
		t.Fatalf("不传 owner 应返回全部，实际 %d", got)
	}
}
