package mcp

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/kinwyb/buibuiCodex/mcp/edp"
	"github.com/kinwyb/buibuiCodex/mcp/serv"
	"github.com/kinwyb/buibuiCodex/mcp/serv/finance"
	"github.com/kinwyb/buibuiCodex/mcp/serv/order"
	"github.com/mark3labs/mcp-go/server"
)

type Config struct {
	Name      string            `description:"mcp名称"`
	WorkSpace string            `description:"工作区"`
	Address   string            `description:"监听地址"`
	Version   string            `description:"mcp服务版本"`
	BaseURL   string            `description:"基础信息"`
	UserMap   map[string]string `description:"用户映射"`
}

type Mcp struct {
	cfg        *Config                      `description:"配置"`
	server     *server.MCPServer            `description:"mcp服务器"`
	httpServer *server.StreamableHTTPServer `description:"http服务"`
	tmpPath    string                       `description:"临时文件目录"`
	tokenCache *edp.TokenCache              `description:"token缓存"`
}

func NewMcp(cfg *Config) *Mcp {
	if cfg == nil {
		cfg = &Config{
			Name:      "buibui",
			Version:   "1.0.0",
			WorkSpace: filepath.Dir(os.Args[0]),
			BaseURL:   "http://127.0.0.1:8080",
		}
	}
	return &Mcp{
		cfg: cfg,
	}
}

func (m *Mcp) Start(ctx context.Context) error {
	// 初始化edp
	m.tmpPath = filepath.Join(m.cfg.WorkSpace, "tmp")
	serv.TmpDir = m.tmpPath
	cache, err := edp.NewTokenCache(filepath.Join(m.cfg.WorkSpace, "token"), m.cfg.UserMap)
	if err != nil {
		return err
	}
	m.tokenCache = cache
	edp.NewAPIClient(m.cfg.BaseURL)
	// 初始化 MCP 服务
	s := server.NewMCPServer(m.cfg.Name, m.cfg.Version, server.WithToolCapabilities(true),
		server.WithLogging())
	// 注册工具
	order.InitTool()
	finance.InitTool()
	tools := serv.McpTools()
	if len(tools) > 0 {
		s.AddTools(tools...)
	}
	// 启动 http 服务提供mcp服务
	m.httpServer = server.NewStreamableHTTPServer(s, server.WithDisableLocalhostProtection(true))
	go func() {
		if err := m.httpServer.Start(":9090"); err != nil {
			log.Fatalf("启动失败: %v", err)
		}
	}()
	return nil
}

func (m *Mcp) Stop(ctx context.Context) error {
	if m.httpServer != nil {
		m.httpServer.Shutdown(ctx)
	}
	return nil
}
