// Package filestore 管理 ws 会话里流转的文件：落地、索引、按归属取用与过期回收。
//
// 它只为 ws 服务：上行把浏览器上传的字节写到磁盘并返回 fileID，
// 下行把 agent 产出的文件登记成可下载资源。与 channel/wxcom 的 media 不同，
// 这里没有第三方存储、没有 AES 密钥、没有 media_id —— 文件就在本机，
// 浏览器凭 fileID + 身份从 HTTP 端点取回。
//
// 磁盘布局（root 默认 ./tmp/ws）：
//
//	<root>/<YYYYMMDD>/<fileID>/<sanitized name>
//
// 用 fileID 建目录而不是直接用文件名，重名不会互相覆盖，过期回收时整目录删掉即可。
package filestore

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

var (
	// ErrNotFound 文件不存在或已过期
	ErrNotFound = errors.New("filestore: file not found")
	// ErrTooLarge 超过单文件上限
	ErrTooLarge = errors.New("filestore: file too large")
	// ErrForbidden 文件不属于该用户
	ErrForbidden = errors.New("filestore: file not owned by user")
)

// DefaultTTL 文件默认保留时长。上传的文件是「待消费」的中间产物，
// 下行文件是「已投递」的附件，都不需要长期留存。
const DefaultTTL = 24 * time.Hour

// Item 一份文件的元信息。Path 属于服务端内部状态，不对外序列化。
type Item struct {
	// ID 对外标识，随机 16 字节十六进制，同时是磁盘目录名
	ID string `json:"id"`
	// Name 用户看到的文件名
	Name string `json:"name"`
	// Size 字节数
	Size int64 `json:"size"`
	// Mime 由文件名后缀推断，取不到时留空
	Mime string `json:"mime,omitempty"`
	// Created / Expires 毫秒时间戳
	CreatedAt int64 `json:"createdAt"`
	ExpiresAt int64 `json:"expiresAt"`

	// Owner 上传者 / 接收者的用户标识，用于归属校验
	Owner string `json:"-"`
	// Path 落盘的绝对路径
	Path string `json:"-"`
}

// Store 文件索引与磁盘根目录。
//
// 索引在内存里，进程重启即失效——这是有意的：ttl 短、文件是临时产物，
// 重启后靠磁盘清理兜底，不需要为它引入持久化存储。
type Store struct {
	root string
	ttl  time.Duration

	mu    sync.RWMutex
	items map[string]*Item

	// now 便于测试注入时钟
	now func() time.Time
}

// New 创建存储。root 为空时用 ./tmp/ws。
func New(root string, ttl time.Duration) *Store {
	if strings.TrimSpace(root) == "" {
		root = filepath.Join(".", "tmp", "ws")
	}
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	return &Store{
		root:  abs,
		ttl:   ttl,
		items: make(map[string]*Item),
		now:   time.Now,
	}
}

// Root 返回磁盘根目录
func (s *Store) Root() string { return s.root }

// TTL 返回保留时长
func (s *Store) TTL() time.Duration { return s.ttl }

// Save 把 r 的内容写成一份新文件。limit 为单文件字节上限，<=0 表示不限制。
//
// 边读边写，不在内存里攒整个文件；超过 limit 立刻中止并清理半成品。
func (s *Store) Save(owner, filename string, r io.Reader, limit int64) (*Item, error) {
	if r == nil {
		return nil, errors.New("filestore: reader is nil")
	}
	item, file, err := s.create(owner, filename)
	if err != nil {
		return nil, err
	}

	written, err := copyLimited(file, r, limit)
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		// 半成品不留：目录里只有这一个文件，整体删掉
		_ = os.RemoveAll(filepath.Dir(item.Path))
		if errors.Is(err, ErrTooLarge) {
			return nil, err
		}
		return nil, fmt.Errorf("filestore: write file failed: %w", err)
	}
	if written == 0 {
		_ = os.RemoveAll(filepath.Dir(item.Path))
		return nil, errors.New("filestore: empty file")
	}

	item.Size = written
	s.index(item)
	return item, nil
}

// Adopt 把磁盘上已有的文件登记为可下载资源，不复制内容。
//
// 用于 agent 的 send_file：文件本来就在容器映射过来的工作目录里，
// 复制一份纯属浪费，登记后由下载端点直接读原路径。
func (s *Store) Adopt(owner, path, filename string) (*Item, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("filestore: resolve path failed: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("filestore: stat file failed: %w", err)
	}
	if info.IsDir() {
		return nil, errors.New("filestore: cannot adopt a directory")
	}
	if filename == "" {
		filename = filepath.Base(abs)
	}

	now := s.now()
	item := &Item{
		ID:        newID(),
		Name:      sanitizeName(filename),
		Size:      info.Size(),
		Mime:      mimeOf(filename),
		Owner:     owner,
		Path:      abs,
		CreatedAt: now.UnixMilli(),
		ExpiresAt: now.Add(s.ttl).UnixMilli(),
	}
	s.index(item)
	return item, nil
}

// Get 按 ID 取文件。已过期、或磁盘上已经没有这个文件时视为不存在。
//
// 磁盘校验不能省：上行附件在投递给 agent 时会被 core.Manager 的
// mapMediaToContainer 移动到该 agent 的工作区（pathmap.MoveFileToContainer），
// 原路径随即消失。索引里留着记录但文件已经不在，此时必须当作不存在，
// 否则会把一个死路径交给下游。
func (s *Store) Get(id string) (*Item, bool) {
	s.mu.RLock()
	item, ok := s.items[id]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if s.expired(item) {
		_ = s.Delete(id)
		return nil, false
	}
	if _, err := os.Stat(item.Path); err != nil {
		s.mu.Lock()
		delete(s.items, id)
		s.mu.Unlock()
		return nil, false
	}
	return item, true
}

// GetOwned 取文件并校验归属：owner 为空表示不校验（服务端内部调用）
func (s *Store) GetOwned(id, owner string) (*Item, error) {
	item, ok := s.Get(id)
	if !ok {
		return nil, ErrNotFound
	}
	if owner != "" && item.Owner != owner {
		// 不返回「无权限」以外的信息，避免探测他人 fileID
		return nil, ErrForbidden
	}
	return item, nil
}

// List 返回某用户名下的有效文件，按创建时间倒序。
// owner 为空时返回全部（仅服务端内部使用）。
func (s *Store) List(owner string) []*Item {
	s.mu.RLock()
	items := make([]*Item, 0, len(s.items))
	for _, item := range s.items {
		if owner != "" && item.Owner != owner {
			continue
		}
		if s.expired(item) {
			continue
		}
		if _, err := os.Stat(item.Path); err != nil {
			continue
		}
		items = append(items, item)
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items
}

// Delete 删除文件并清理索引。Adopt 的资源只丢索引，不动原文件——
// 那份文件属于 agent 的工作目录，不归这里管。
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	item, ok := s.items[id]
	delete(s.items, id)
	s.mu.Unlock()
	if !ok {
		return ErrNotFound
	}
	if s.ownedByStore(item) {
		if err := os.RemoveAll(filepath.Dir(item.Path)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// GC 清理过期文件，返回清理条数
func (s *Store) GC() int {
	s.mu.RLock()
	ids := make([]string, 0, len(s.items))
	for id := range s.items {
		ids = append(ids, id)
	}
	s.mu.RUnlock()

	removed := 0
	for _, id := range ids {
		s.mu.RLock()
		item, ok := s.items[id]
		s.mu.RUnlock()
		if !ok || !s.expired(item) {
			continue
		}
		if err := s.Delete(id); err == nil {
			removed++
		}
	}
	s.gcDisk()
	return removed
}

// ---- 内部实现 ----

// create 建好目录与空文件，返回待写入的 Item（尚未进索引）
func (s *Store) create(owner, filename string) (*Item, *os.File, error) {
	id := newID()
	name := sanitizeName(filename)
	dir := filepath.Join(s.root, s.now().Format("20060102"), id)

	// 落盘名带 fileID 前缀：附件随后会被 core.Manager 移到 agent 工作区的 tmp，
	// 那里是按文件名铺平的，用户连着传两个「报告.xlsx」就会互相覆盖。
	// 展示名仍然是原始文件名（下载、metadata 里用的都是 Name）。
	stored := id[:8] + "_" + name

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("filestore: create dir failed: %w", err)
	}
	path := filepath.Join(dir, stored)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, nil, fmt.Errorf("filestore: create file failed: %w", err)
	}

	now := s.now()
	return &Item{
		ID:        id,
		Name:      name,
		Mime:      mimeOf(name),
		Owner:     owner,
		Path:      path,
		CreatedAt: now.UnixMilli(),
		ExpiresAt: now.Add(s.ttl).UnixMilli(),
	}, file, nil
}

func (s *Store) index(item *Item) {
	s.mu.Lock()
	s.items[item.ID] = item
	s.mu.Unlock()
}

func (s *Store) expired(item *Item) bool {
	return item.ExpiresAt > 0 && s.now().UnixMilli() >= item.ExpiresAt
}

// ownedByStore 判断文件是否由本存储创建（而不是 Adopt 进来的外部文件）
func (s *Store) ownedByStore(item *Item) bool {
	return strings.HasPrefix(item.Path, s.root+string(filepath.Separator))
}

// gcDisk 清理磁盘上已无索引引用的日期目录，兜住进程重启后索引丢失的情况
func (s *Store) gcDisk() {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return
	}
	s.mu.RLock()
	live := make(map[string]struct{}, len(s.items))
	for _, item := range s.items {
		live[item.Path] = struct{}{}
	}
	s.mu.RUnlock()

	for _, day := range entries {
		if !day.IsDir() {
			continue
		}
		dayPath := filepath.Join(s.root, day.Name())
		files, err := os.ReadDir(dayPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !f.IsDir() {
				continue
			}
			dir := filepath.Join(dayPath, f.Name())
			if s.hasLiveFile(dir, live) {
				continue
			}
			// 目录里没有任何在用文件：先看是否已过期（靠 mtime 兜底），
			// 避免把正在上传、尚未进索引的目录删掉
			info, err := f.Info()
			if err != nil || s.now().Sub(info.ModTime()) < s.ttl {
				continue
			}
			_ = os.RemoveAll(dir)
		}
	}
}

func (s *Store) hasLiveFile(dir string, live map[string]struct{}) bool {
	for path := range live {
		if strings.HasPrefix(path, dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// copyLimited 带上限的拷贝，超限返回 ErrTooLarge
func copyLimited(dst io.Writer, src io.Reader, limit int64) (int64, error) {
	if limit <= 0 {
		return io.Copy(dst, src)
	}
	// 多读一个字节用于判定「恰好等于上限」还是「超了」
	written, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if err != nil {
		return written, err
	}
	if written > limit {
		return written, ErrTooLarge
	}
	return written, nil
}

// sanitizeName 只保留文件名本身，剔除路径分隔符与控制字符，避免目录穿越和非法文件名。
//
// 这里不用 filepath.Base：它按当前平台的分隔符切分，而上传方可能来自 Windows，
// 一个 "C:\tmp\a.txt" 在 Linux/macOS 上会被原样当成文件名。
func sanitizeName(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "file"
	}
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		switch {
		case r == '/' || r == '\\' || r == 0:
			b.WriteRune('_')
		case unicode.IsControl(r):
			// 丢弃控制字符
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" || out == "." || out == ".." {
		return "file"
	}
	// 文件名过长会撑爆部分文件系统的单段限制，保留后缀截断主干
	const maxName = 120
	if len(out) > maxName {
		ext := filepath.Ext(out)
		if len(ext) > 20 {
			ext = ""
		}
		keep := maxName - len(ext)
		if keep > 0 {
			out = out[:keep] + ext
		} else {
			out = out[:maxName]
		}
	}
	return out
}

// mimeOf 按后缀推断 MIME。取不到时返回空串，由下载端点嗅探内容兜底
// （mime.TypeByExtension 依赖系统的 mime.types，容器里未必装全）。
func mimeOf(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return ""
	}
	return mime.TypeByExtension(ext)
}

// newID 生成 16 字节随机标识
func newID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// 熵源异常时退化成时间戳，宁可弱一点也不要拿不到 ID
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf[:])
}
