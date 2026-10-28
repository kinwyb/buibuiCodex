// 本文件存放命令间共用的小工具。

package command

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/core/types"
)

// mediaTypeByName 按扩展名判断媒体类型，图片/音频单独归类，其余按文档处理
func mediaTypeByName(name string) types.MediaType {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico", ".heic":
		return types.MediaTypeImage
	case ".mp3", ".wav", ".m4a", ".aac", ".ogg", ".flac", ".amr", ".silk":
		return types.MediaTypeAudio
	default:
		return types.MediaTypeFile
	}
}

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
