package proc

import "context"

type ProcessResult struct {
	WsUrl string `description:"codex ws地址"`
	Token string `description:"codex ws连接token"`
}

type Process interface {
	// Start 启动进程
	Start(ctx context.Context) (*ProcessResult, error)
	// Stop 停止进程
	Stop() error
}
