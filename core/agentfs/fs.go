// Package agentfs 以 agent 的容器视角访问它的工作区文件。
//
// agent 跑在容器里，看到的 /workspace、/tmp 分别对应宿主机的
// <WorkSpace>/workspace、<WorkSpace>/tmp（映射关系见 core/manager.go 里的 pathmap 配置）。
// Web 前端只知道容器路径，所以这里统一接收容器路径、自己转成宿主机路径再操作，
// 两个方向都做越界校验——**不能直接复用 pathmap.ToHost**：它匹配不上时原样返回，
// 那个语义适合拼接路径，不适合做安全边界。
//
// 校验有三层：
//
//  1. 容器路径必须是绝对路径，Clean 后命中某个映射根（含根本身）；
//  2. 拼出的宿主机路径 Clean 后仍在对应宿主根内；
//  3. 解析软链接后仍在宿主根内（防止工作区里一个指向 /etc 的链接把外部目录带进来）。
package agentfs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/core/pathmap"
)

var (
	// ErrOutOfScope 路径不在任何映射根内（或试图用 .. 逃逸）
	ErrOutOfScope = errors.New("agentfs: path out of scope")
	// ErrNotAbsolute 容器路径必须是绝对路径
	ErrNotAbsolute = errors.New("agentfs: path must be absolute")
	// ErrNotFound 文件或目录不存在
	ErrNotFound = errors.New("agentfs: not found")
	// ErrIsDir 目标是目录，不能当文件读
	ErrIsDir = errors.New("agentfs: is a directory")
	// ErrNotDir 目标不是目录
	ErrNotDir = errors.New("agentfs: not a directory")
	// ErrTooLarge 超过单文件上限
	ErrTooLarge = errors.New("agentfs: file too large")
	// ErrRootProtected 不允许删除映射根
	ErrRootProtected = errors.New("agentfs: root is protected")
)

// Root 一个可浏览的根（容器视角）
type Root struct {
	// Container 容器内路径，如 /workspace
	Container string `json:"container"`
	// Exists 宿主机目录是否已创建。容器还没启动过时 /workspace 的源目录不存在，
	// 前端据此提示「先让 agent 跑一次」而不是显示一个空目录。
	Exists bool `json:"exists"`
	// Host 宿主机路径，仅用于排查，前端不需要
	Host string `json:"-"`
}

// Entry 目录项 / 文件元信息。Path 始终是容器路径，前端拿它继续发请求。
type Entry struct {
	Name string `json:"name"`
	// Path 容器内完整路径
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
	// ModTime 毫秒时间戳
	ModTime int64 `json:"modTime"`
	// Mime 只对文件有意义，按后缀推断，取不到时留空
	Mime string `json:"mime,omitempty"`
	// Symlink 是否为软链接（不跟随，避免把外部目录误当工作区内容）
	Symlink bool `json:"symlink,omitempty"`
	// Mode 权限位，如 "-rw-r--r--"，前端展示用
	Mode string `json:"mode,omitempty"`
}

// FS 某个 agent 的工作区视图
type FS struct {
	roots []Root
}

// New 按路径映射创建视图。没有映射时（非 Docker 模式）返回空视图，
// 此时没有任何可浏览的根——宁可什么都不给，也不要放开整个文件系统。
func New(mappings []pathmap.PathMapping) *FS {
	roots := make([]Root, 0, len(mappings))
	for _, m := range mappings {
		if m.Host == "" || m.Container == "" {
			continue
		}
		roots = append(roots, Root{
			Container: path.Clean(m.Container),
			Host:      filepath.Clean(m.Host),
		})
	}
	return &FS{roots: roots}
}

// Roots 可浏览的根，带目录是否已创建
func (f *FS) Roots() []Root {
	out := make([]Root, len(f.roots))
	copy(out, f.roots)
	for i := range out {
		if _, err := os.Stat(out[i].Host); err == nil {
			out[i].Exists = true
		}
	}
	return out
}

// List 列目录。containerPath 为空时用第一个根。
func (f *FS) List(containerPath string) ([]Entry, error) {
	host, root, err := f.resolve(containerPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(host)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, ErrNotDir
	}

	items, err := os.ReadDir(host)
	if err != nil {
		return nil, err
	}
	containerDir := f.containerOf(host, root)

	out := make([]Entry, 0, len(items))
	for _, item := range items {
		info, err := item.Info()
		if err != nil {
			// 单个条目读不到（权限、竞态删除）不该让整个列表失败
			continue
		}
		out = append(out, f.entry(item.Name(), path.Join(containerDir, item.Name()), info))
	}
	return out, nil
}

// Stat 取单个路径的元信息
func (f *FS) Stat(containerPath string) (Entry, error) {
	host, root, err := f.resolve(containerPath)
	if err != nil {
		return Entry{}, err
	}
	info, err := os.Stat(host)
	if err != nil {
		if os.IsNotExist(err) {
			return Entry{}, ErrNotFound
		}
		return Entry{}, err
	}
	return f.entry(filepath.Base(host), f.containerOf(host, root), info), nil
}

// Open 打开文件供下载/内联。返回的文件句柄由调用方关闭。
func (f *FS) Open(containerPath string) (*os.File, Entry, error) {
	host, root, err := f.resolve(containerPath)
	if err != nil {
		return nil, Entry{}, err
	}
	info, err := os.Stat(host)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, Entry{}, ErrNotFound
		}
		return nil, Entry{}, err
	}
	if info.IsDir() {
		return nil, Entry{}, ErrIsDir
	}
	file, err := os.Open(host)
	if err != nil {
		return nil, Entry{}, err
	}
	entry := f.entry(filepath.Base(host), f.containerOf(host, root), info)
	return file, entry, nil
}

// Save 写入（覆盖）文件，父目录必须已存在。limit<=0 表示不限制。
//
// 不做「自动建目录」：前端要建目录得显式调 Mkdir，
// 否则一个拼错的路径会在工作区里悄悄长出一串空目录。
func (f *FS) Save(containerPath string, r io.Reader, limit int64) (Entry, error) {
	host, root, err := f.resolve(containerPath)
	if err != nil {
		return Entry{}, err
	}
	if info, err := os.Stat(host); err == nil && info.IsDir() {
		return Entry{}, ErrIsDir
	}
	if _, err := os.Stat(filepath.Dir(host)); err != nil {
		return Entry{}, ErrNotFound
	}

	// 先写临时文件再 rename：中途失败不会把原文件截断成半截
	tmp := filepath.Join(filepath.Dir(host),
		fmt.Sprintf(".%s.part-%d", filepath.Base(host), time.Now().UnixNano()))
	file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return Entry{}, err
	}

	written, copyErr := copyLimited(file, r, limit)
	closeErr := file.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = os.Remove(tmp)
		if errors.Is(copyErr, ErrTooLarge) {
			return Entry{}, copyErr
		}
		return Entry{}, copyErr
	}
	if err := os.Rename(tmp, host); err != nil {
		_ = os.Remove(tmp)
		return Entry{}, err
	}

	info, err := os.Stat(host)
	if err != nil {
		return Entry{}, err
	}
	entry := f.entry(filepath.Base(host), f.containerOf(host, root), info)
	entry.Size = written
	return entry, nil
}

// Mkdir 创建目录（含父目录）
func (f *FS) Mkdir(containerPath string) (Entry, error) {
	host, root, err := f.resolve(containerPath)
	if err != nil {
		return Entry{}, err
	}
	if err := os.MkdirAll(host, 0o755); err != nil {
		return Entry{}, err
	}
	info, err := os.Stat(host)
	if err != nil {
		return Entry{}, err
	}
	return f.entry(filepath.Base(host), f.containerOf(host, root), info), nil
}

// Remove 删除文件或目录。目录必须显式 recursive 才递归删，
// 并且映射根本身永远不允许删。
func (f *FS) Remove(containerPath string, recursive bool) error {
	host, root, err := f.resolve(containerPath)
	if err != nil {
		return err
	}
	if host == root {
		return ErrRootProtected
	}
	info, err := os.Stat(host)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	if info.IsDir() && !recursive {
		return ErrIsDir
	}
	return os.RemoveAll(host)
}

// ---- 内部实现 ----

// resolve 容器路径 → 宿主机路径，并做三层越界校验
func (f *FS) resolve(containerPath string) (host string, root string, err error) {
	if strings.TrimSpace(containerPath) == "" {
		if len(f.roots) == 0 {
			return "", "", ErrOutOfScope
		}
		containerPath = f.roots[0].Container
	}
	// 容器内是 POSIX 路径，用 path 而不是 filepath 做 Clean，
	// 否则在 Windows 宿主上 "../" 的语义会跑偏
	if !path.IsAbs(containerPath) {
		return "", "", ErrNotAbsolute
	}
	cleaned := path.Clean(containerPath)

	for _, r := range f.roots {
		if cleaned != r.Container && !strings.HasPrefix(cleaned, r.Container+"/") {
			continue
		}
		rel := strings.TrimPrefix(cleaned, r.Container)
		target := filepath.Join(r.Host, filepath.FromSlash(rel))
		if !within(r.Host, target) {
			return "", "", ErrOutOfScope
		}
		// 软链接可能把目标指到根外：解析真实路径后再校验一次。
		// 文件还不存在时（上传新文件）解析父目录，父目录也存在不了就直接放行给后续的 Stat 报错。
		if real, ok := evalExisting(target); ok && !within(realRoot(r.Host), real) {
			return "", "", ErrOutOfScope
		}
		return target, r.Host, nil
	}
	return "", "", ErrOutOfScope
}

// containerOf 宿主机路径还原成容器路径
func (f *FS) containerOf(host string, root string) string {
	rel, err := filepath.Rel(root, host)
	if err != nil || rel == "." {
		for _, r := range f.roots {
			if r.Host == root {
				return r.Container
			}
		}
		return root
	}
	for _, r := range f.roots {
		if r.Host == root {
			return path.Join(r.Container, filepath.ToSlash(rel))
		}
	}
	return filepath.ToSlash(host)
}

func (f *FS) entry(name, containerPath string, info os.FileInfo) Entry {
	e := Entry{
		Name:    name,
		Path:    containerPath,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime().UnixMilli(),
		Mode:    info.Mode().String(),
		Symlink: info.Mode()&os.ModeSymlink != 0,
	}
	if !e.IsDir {
		e.Mime = mimeOf(name)
	}
	return e
}

// within 判断 target 是否在 root 内（含 root 本身）
func within(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if target == root {
		return true
	}
	return strings.HasPrefix(target, root+string(filepath.Separator))
}

// evalExisting 尽力解析真实路径：目标存在就解析目标，否则解析父目录
func evalExisting(host string) (string, bool) {
	if real, err := filepath.EvalSymlinks(host); err == nil {
		return real, true
	}
	parent := filepath.Dir(host)
	if real, err := filepath.EvalSymlinks(parent); err == nil {
		return filepath.Join(real, filepath.Base(host)), true
	}
	return "", false
}

// realRoot 解析根的真实路径（根本身可能是软链接）。
//
// 根还不存在时（容器没启动过，<WorkSpace>/tmp 尚未创建）不能直接返回原值：
// EvalSymlinks 对存在的父目录会解析成 /private/var/... 这类真实路径，
// 而原值还是 /var/...，两边基准不一致会把合法路径误判成越界。
// 这里与 evalExisting 用同一条规则，保证比较的基准一致。
func realRoot(root string) string {
	if real, err := filepath.EvalSymlinks(root); err == nil {
		return real
	}
	if real, ok := evalExisting(root); ok {
		return real
	}
	return root
}

func copyLimited(dst io.Writer, src io.Reader, limit int64) (int64, error) {
	if limit <= 0 {
		return io.Copy(dst, src)
	}
	written, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if err != nil {
		return written, err
	}
	if written > limit {
		return written, ErrTooLarge
	}
	return written, nil
}

// mimeOf 按后缀推断，取不到返回空串（下载时再嗅探）
func mimeOf(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return ""
	}
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".json":
		return "application/json"
	case ".md", ".txt", ".log", ".csv", ".tsv":
		return "text/plain; charset=utf-8"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".xml":
		return "application/xml"
	case ".zip":
		return "application/zip"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	}
	// 推断不出来就留空，让下载端点按内容嗅探
	return ""
}
