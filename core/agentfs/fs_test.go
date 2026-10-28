package agentfs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/pathmap"
)

// newFS 造一个临时工作区：<tmp>/workspace 与 <tmp>/tmp，
// 映射成 agent 眼里的 /workspace 与 /tmp。
func newFS(t *testing.T) (*FS, string) {
	t.Helper()
	root := t.TempDir()
	wsDir := filepath.Join(root, "workspace")
	tmpDir := filepath.Join(root, "tmp")
	for _, dir := range []string{wsDir, tmpDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("建目录失败: %v", err)
		}
	}
	fs := New([]pathmap.PathMapping{
		{Host: wsDir, Container: "/workspace"},
		{Host: tmpDir, Container: "/tmp"},
	})
	return fs, root
}

func TestRoots(t *testing.T) {
	fs, _ := newFS(t)
	roots := fs.Roots()
	if len(roots) != 2 || roots[0].Container != "/workspace" || roots[1].Container != "/tmp" {
		t.Fatalf("根不符: %+v", roots)
	}
	for _, r := range roots {
		if !r.Exists {
			t.Fatalf("目录已创建，Exists 应为 true: %+v", r)
		}
	}
}

func TestRootNotExist(t *testing.T) {
	// 容器还没启动过：宿主机目录不存在，前端据此提示而不是显示空目录
	missing := filepath.Join(t.TempDir(), "nope")
	fs := New([]pathmap.PathMapping{{Host: missing, Container: "/workspace"}})
	roots := fs.Roots()
	if len(roots) != 1 || roots[0].Exists {
		t.Fatalf("目录不存在时 Exists 应为 false: %+v", roots)
	}
	if _, err := fs.List("/workspace"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("列不存在的目录应报 ErrNotFound，实际 %v", err)
	}
}

func TestListAndStat(t *testing.T) {
	fs, root := newFS(t)
	if err := os.WriteFile(filepath.Join(root, "workspace", "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "workspace", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	items, err := fs.List("/workspace")
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("应有 2 项，实际 %d", len(items))
	}
	for _, item := range items {
		if !strings.HasPrefix(item.Path, "/workspace/") {
			t.Fatalf("条目路径应是容器路径: %+v", item)
		}
		if item.Name == "a.txt" && (item.IsDir || item.Size != 5) {
			t.Fatalf("a.txt 元信息不符: %+v", item)
		}
	}

	entry, err := fs.Stat("/workspace/a.txt")
	if err != nil {
		t.Fatalf("Stat 失败: %v", err)
	}
	if entry.Size != 5 || entry.Mime == "" {
		t.Fatalf("元信息不符: %+v", entry)
	}
}

func TestOutOfScope(t *testing.T) {
	fs, _ := newFS(t)

	cases := []string{
		"/etc/passwd",          // 不在任何根内
		"/workspace/../../etc", // Clean 后跑出根
		"/",                    // 根本不是根
		"/workspace2/a.txt",    // 前缀形似但不是同一层
	}
	for _, p := range cases {
		if _, err := fs.List(p); !errors.Is(err, ErrOutOfScope) {
			t.Errorf("List(%q) 应拒绝，实际 %v", p, err)
		}
	}

	if _, err := fs.Stat("workspace/a.txt"); !errors.Is(err, ErrNotAbsolute) {
		t.Errorf("相对路径应拒绝，实际 %v", err)
	}
}

func TestSymlinkEscape(t *testing.T) {
	fs, root := newFS(t)

	// 工作区里放一个指向外部的软链接，不能成为读取外部文件的跳板
	link := filepath.Join(root, "workspace", "evil")
	if err := os.Symlink("/etc", link); err != nil {
		t.Skipf("当前环境不支持建软链接: %v", err)
	}

	if _, _, err := fs.Open("/workspace/evil/passwd"); !errors.Is(err, ErrOutOfScope) {
		t.Fatalf("软链接逃逸应被拒绝，实际 %v", err)
	}
	// 列表本身可以列出这个链接（不跟随），但要标记出来
	items, err := fs.List("/workspace")
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	for _, item := range items {
		if item.Name == "evil" && !item.IsDir {
			// 软链接不是目录，ReadDir 不跟随，符合预期
			continue
		}
		if item.Name == "evil" && item.IsDir {
			t.Fatal("软链接不应被当成目录")
		}
	}
}

func TestSaveOpenRemove(t *testing.T) {
	fs, root := newFS(t)

	entry, err := fs.Save("/workspace/out.txt", strings.NewReader("数据"), 0)
	if err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	if entry.Size != int64(len("数据")) {
		t.Fatalf("大小不符: %d", entry.Size)
	}

	// 内容能被读回
	file, info, err := fs.Open("/workspace/out.txt")
	if err != nil {
		t.Fatalf("打开失败: %v", err)
	}
	defer file.Close()
	if info.Name != "out.txt" {
		t.Fatalf("名字不符: %s", info.Name)
	}

	// 覆盖写
	if _, err := fs.Save("/workspace/out.txt", strings.NewReader("new"), 0); err != nil {
		t.Fatalf("覆盖失败: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "workspace", "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("覆盖后内容不符: %q", data)
	}

	if err := fs.Remove("/workspace/out.txt", false); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	if _, err := fs.Stat("/workspace/out.txt"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("删除后应查不到，实际 %v", err)
	}
}

func TestSaveTooLarge(t *testing.T) {
	fs, root := newFS(t)
	if _, err := fs.Save("/workspace/big.bin", strings.NewReader(strings.Repeat("x", 2048)), 1024); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("超限应报错，实际 %v", err)
	}
	// 不能留半截文件或 .part
	items, err := os.ReadDir(filepath.Join(root, "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("超限后应清理临时文件，仍残留 %d 个", len(items))
	}
}

func TestRemoveRules(t *testing.T) {
	fs, root := newFS(t)
	if err := os.MkdirAll(filepath.Join(root, "workspace", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "workspace", "sub", "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 目录必须显式递归
	if err := fs.Remove("/workspace/sub", false); !errors.Is(err, ErrIsDir) {
		t.Fatalf("非递归删目录应拒绝，实际 %v", err)
	}
	if err := fs.Remove("/workspace/sub", true); err != nil {
		t.Fatalf("递归删除失败: %v", err)
	}

	// 根不允许删
	if err := fs.Remove("/workspace", true); !errors.Is(err, ErrRootProtected) {
		t.Fatalf("删根应拒绝，实际 %v", err)
	}
}

func TestMkdirAndAutoDirForbidden(t *testing.T) {
	fs, _ := newFS(t)

	entry, err := fs.Mkdir("/workspace/docs")
	if err != nil {
		t.Fatalf("建目录失败: %v", err)
	}
	if !entry.IsDir {
		t.Fatalf("应是目录: %+v", entry)
	}

	// 父目录不存在时写入要报错，而不是偷偷建一串空目录
	if _, err := fs.Save("/workspace/missing/deep/a.txt", strings.NewReader("x"), 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("父目录不存在应报错，实际 %v", err)
	}
}

func TestNoMappingIsEmpty(t *testing.T) {
	fs := New(nil)
	if len(fs.Roots()) != 0 {
		t.Fatal("没有映射时不应有可浏览的根")
	}
	if _, err := fs.List("/workspace"); !errors.Is(err, ErrOutOfScope) {
		t.Fatalf("没有映射时应拒绝，实际 %v", err)
	}
}
