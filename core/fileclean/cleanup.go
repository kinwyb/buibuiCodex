// Package fileclean 统一回收临时目录中的历史文件。
//
// 各模块落盘目录不同，但规则一致：删除超过 TTL 的文件（递归子目录），
// 再清掉因此变空的子目录；传入的根目录始终保留。
package fileclean

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultTTL 历史文件保留时长。
	DefaultTTL = 24 * time.Hour
	// DefaultInterval 后台回收扫描间隔。
	DefaultInterval = time.Hour
)

// Run 阻塞运行回收循环：启动时先清一次，之后每隔 DefaultInterval 清理 dirs，
// ctx 结束时返回。需要后台执行时由调用方自行启动 goroutine。
func Run(ctx context.Context, dirs ...string) {
	cleanAll(dirs, DefaultTTL, time.Now())
	ticker := time.NewTicker(DefaultInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			cleanAll(dirs, DefaultTTL, now)
		}
	}
}

// Clean 删除 dir 下修改时间超过 ttl 的文件，返回删除数量。
// 子目录递归处理；清空后的子目录一并删除，根目录保留。
func Clean(dir string, ttl time.Duration, now time.Time) int {
	deadline := now.Add(-ttl)
	removed := 0
	var subdirs []string

	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != dir {
				subdirs = append(subdirs, path)
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(deadline) {
			return nil
		}
		if os.Remove(path) == nil {
			removed++
		}
		return nil
	})

	// WalkDir 是前序遍历，倒序即为先子后父，父目录才能看到子目录已删空。
	for i := len(subdirs) - 1; i >= 0; i-- {
		if entries, err := os.ReadDir(subdirs[i]); err == nil && len(entries) == 0 {
			_ = os.Remove(subdirs[i])
		}
	}
	return removed
}

func cleanAll(dirs []string, ttl time.Duration, now time.Time) {
	for _, dir := range dirs {
		Clean(dir, ttl, now)
	}
}
