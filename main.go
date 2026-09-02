package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kinwyb/buibuiCodex/channel"
	"github.com/kinwyb/buibuiCodex/core"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/config"
)

func main() {
	// 1. 定义 Handler 选项，将日志级别设置为 LevelDebug
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // 开启 Debug 级别（会输出 Debug, Info, Warn, Error）
	}
	// 2. 使用选项创建 Handler（可选 slog.NewTextHandler 或 slog.NewJSONHandler）
	handler := slog.NewTextHandler(os.Stdout, opts)
	// 3. 构建并设置为默认 Logger
	logger := slog.New(handler)
	slog.SetDefault(logger)

	ctx := context.Background()
	cfg, err := config.Load("/Users/wangyingbin/Developer/go/src/bgAgent/buibuiCodex/buibui.json")
	if err != nil {
		slog.Error("load config failed: ", "error", err)
		return
	}
	msgBus := bus.NewMessageBus(100)

	channel := channel.NewManager(msgBus)
	err = channel.InitializeFromConfig(ctx, cfg.Channels)
	if err != nil {
		slog.Error("init channel failed", "error", err)
		return
	}
	err = channel.StartAll(ctx)
	if err != nil {
		slog.Error("start channel failed", "error", err)
		return
	}
	defer channel.StopAll()
	manager := core.NewManager(msgBus)
	defer manager.Stop()
	err = manager.InitFromConfig(ctx, &cfg.ManagerConfig)
	if err != nil {
		slog.Error("register agents failed ", "error", err)
		return
	}
	err = manager.Start(ctx)
	if err != nil {
		slog.Error("manager.Start failed", "error", err)
		return
	}
	slog.Info("manager.Start ok")
	// 5. 监听系统中断信号（Ctrl+C 或 SIGTERM），实现优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n正在安全关闭 Go 服务与 app-server 进程...")

}
