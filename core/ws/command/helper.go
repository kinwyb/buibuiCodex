// 本文件存放命令间共用的小工具。

package command

import (
	"time"
)

// truncateRunes 按字符数截断，超出时补省略号
func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "…"
}

// millis 转毫秒时间戳，零值返回 0 而不是负数
func millis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
