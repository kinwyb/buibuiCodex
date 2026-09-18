package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gookit/rotatefile"
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
	cfgPath, err := getConfigPath()
	//cfgPath := "/Users/wangyingbin/Developer/go/src/bgAgent/buibuiCodex/buibui.json"
	//var err error
	if err != nil {
		slog.Error("config path err", "error", err)
		return
	}

	ctx := context.Background()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Error("load config failed: ", "error", err)
		return
	}
	initLog(cfg.Log)
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
	var dataStorage *db.Data
	if cfg.DataBase != nil {
		ms, err := db.NewMysqlStorage(cfg.DataBase.Username, cfg.DataBase.Password, cfg.DataBase.Server, cfg.DataBase.Database)
		if err != nil {
			panic(err)
		}
		dataStorage = db.NewData(ms)
	} else {
		sqlite, err := db.NewSQLiteStorage(filepath.Join(cfg.Workspace, "buibui.db"))
		if err != nil {
			panic(err)
		}
		dataStorage = db.NewData(sqlite)
	}
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
		TokenMap:  mcpConfig.TokenMap,
	}, dataStorage.Session())
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
					case types.EventUserMessageStart:
						fmt.Println("\n⌛️ 用户提问: [" + event.EventID + "]")
					case types.EventUserMessageCompleted:
						fmt.Println("\n☑️ 提问接受: [" + event.Message.Content + "]")
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
					case types.EventMessageStart:
						fmt.Println("\n↩️ 回复开始: [" + event.EventID + "]")
					case types.EventMessageDelta:
						//fmt.Print(event.Message.Content)
					case types.EventMessageCompleted:
						fmt.Println("\n✅ 完整回复: [" + event.EventID + "]\n" + event.Message.Content)
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

func getConfigPath() (string, error) {
	// 1. 获取当前正在执行的可执行文件的绝对路径
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	// 2. 解析出可执行文件所在的目录
	exeDir := filepath.Dir(exePath)
	// 3. 拼接配置文件路径
	configPath := filepath.Join(exeDir, "buibui.json")
	return configPath, nil
}

func initLog(cfg *config.LogConfig) {
	if cfg == nil {
		cfg = &config.LogConfig{
			Level:      "info",
			MaxBackups: 7,
			MaxSize:    100,
			Compress:   false,
		}
	}
	logLevel := slog.LevelDebug
	switch cfg.Level {
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	opts := &slog.HandlerOptions{
		Level: logLevel,
	}
	if cfg.File != "" {
		// 配置 rotatefile：按天轮转，保留 30 天
		w, _ := rotatefile.NewConfig(cfg.File, func(c *rotatefile.Config) {
			c.MaxSize = uint64(cfg.MaxSize) * rotatefile.OneMByte
			c.RotateTime = rotatefile.EveryDay
			c.BackupNum = uint(cfg.MaxBackups)
			c.Compress = cfg.Compress
		}).Create()

		logger := slog.New(slog.NewJSONHandler(w, opts))
		slog.SetDefault(logger)
	} else {
		// 设置为默认 Logger
		logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
		slog.SetDefault(logger)
	}
}
