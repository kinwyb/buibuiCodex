package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/kinwyb/buibuiCodex/channel"
	_ "github.com/kinwyb/buibuiCodex/channel/wxcom"
	"github.com/kinwyb/buibuiCodex/core"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/mcp"
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
	sqlite, err := db.NewSQLiteStorage(filepath.Join(cfg.Workspace, "buibui.db"))
	if err != nil {
		panic(err)
	}
	dataStorage := db.NewData(sqlite)
	manager := core.NewManager(msgBus, dataStorage)
	defer manager.Stop()
	err = manager.InitFromConfig(ctx, &cfg.ManagerConfig)
	if err != nil {
		slog.Error("register agents failed ", "error", err)
		return
	}
	mcpConfig := cfg.McpServer
	mcpConfig.SetDefaults()
	ms := mcp.NewMcp(&mcp.Config{
		Name:      mcpConfig.Name,
		WorkSpace: filepath.Join(cfg.Workspace, "mcp"),
		Address:   fmt.Sprintf(":%d", mcpConfig.Port),
		Version:   mcpConfig.Version,
		BaseURL:   mcpConfig.BaseURL,
		UserMap:   mcpConfig.UserMap,
		Host:      mcpConfig.Host,
		Issuer:    mcpConfig.Issuer,
		Audience:  mcpConfig.Audience,
		JWKSURL:   mcpConfig.JWKSURL,
	})
	go ms.Start(ctx)
	defer ms.Stop(ctx)
	go func() {
		echan, err := msgBus.SubscribeEvent()
		if err != nil {
			slog.Error("msgBus.SubscribeEvent failed", "error", err)
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			default:
				event, ok := echan.Consume(ctx)
				if ok {
					switch event.Type {
					case types.EventMessageStart:
						fmt.Println("\n↩️ 回复开始: [" + event.EventID + "]")
					case types.EventMessageDelta:
						//fmt.Print(event.Message.Content)
					case types.EventMessageCompleted:
						fmt.Println("\n✅ 完整回复: [" + event.EventID + "]\n" + event.Message.Content)
					case types.EventReasoningStart:
						fmt.Println("\n🤔思考中...[" + event.EventID + "]")
					case types.EventReasoningDelta:
						//fmt.Print(event.Message.ReasoningContent)
					case types.EventReasoningCompleted:
						fmt.Println("\n😊思考完成：[" + event.EventID + "] \n" + event.Message.ReasoningContent)
					case types.EventToolStart:
						for _, tool := range event.Message.ToolCalls {
							fmt.Printf("🔧工具调用: [%s]%s\n", tool.Type, tool.Params)
						}
					case types.EventToolCompleted:
						for _, tool := range event.Message.ToolCalls {
							fmt.Printf("🔧工具调用结束: [%s]%s => %s\n", tool.Type, tool.Params, tool.Result)
						}
					case types.EventError:
						slog.Error("🙅 error: " + event.Message.Content)
					}
				}
			}
		}
	}()
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
