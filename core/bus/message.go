package bus

import (
	"time"
)

type StreamingMode string

// StreamingMode 常量定义（流式输出模式）
const (
	StreamingModeDelta      StreamingMode = "delta"      // 增量模式：Content 返回增量内容
	StreamingModeAccumulate StreamingMode = "accumulate" // 累积模式：Content 返回累积后的完整内容
)

// Log 日志事件（用于系统日志输出）
type Log struct {
	ID        string    `json:"id"`
	Level     string    `json:"level"` // "debug", "info", "warn", "error"
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"` // 来源组件
}

// LogEvent levels
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)
