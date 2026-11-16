// 本文件实现 MCP 临时文件回收。
//
// 大结果落盘后没有删除路径，长期运行会把 tmp 堆满：服务启动时先清一次，
// 之后每小时删除超过 24 小时的文件。目录布局是 <TmpDir>/<YYYY-MM-DD>/<nano>.json，
// 由 bigDataHandler 与 schema-data 落盘共同使用。

package mcp

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

const (
	// tmpFileTTL 临时文件保留时长。
	tmpFileTTL = 24 * time.Hour
	// tmpCleanupInterval 回收扫描间隔。
	tmpCleanupInterval = time.Hour
)

// startTmpCleanup 启动临时文件回收，ctx 结束时退出。
func startTmpCleanup(ctx context.Context, dir string) {
	go func() {
		cleanTmpFiles(dir, tmpFileTTL, time.Now())
		ticker := time.NewTicker(tmpCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				cleanTmpFiles(dir, tmpFileTTL, now)
			}
		}
	}()
}

// cleanTmpFiles 删除 dir 下超过 ttl 的文件，并清掉清空后的日期目录。
// 日期目录里还有未过期文件时整目录保留。
func cleanTmpFiles(dir string, ttl time.Duration, now time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	deadline := now.Add(-ttl)
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if !entry.IsDir() {
			removeTmpFile(path, deadline)
			continue
		}
		files, err := os.ReadDir(path)
		if err != nil {
			continue
		}
		for _, file := range files {
			if !file.IsDir() {
				removeTmpFile(filepath.Join(path, file.Name()), deadline)
			}
		}
		if rest, err := os.ReadDir(path); err == nil && len(rest) == 0 {
			_ = os.Remove(path)
		}
	}
}

func removeTmpFile(path string, deadline time.Time) {
	info, err := os.Stat(path)
	if err != nil || info.ModTime().After(deadline) {
		return
	}
	_ = os.Remove(path)
}
