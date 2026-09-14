package mcp

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	if _, perr := os.Stat(m.tmpPath); os.IsNotExist(perr) {
		_ = os.MkdirAll(m.tmpPath, 0755)
	}
	serv.TmpUrl = "http://10.0.110.80:9090/tmp_file"
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
	// 自定义路由
	mux := http.NewServeMux()
	//if s.protectedResourceMetadataHandler != nil && s.protectedResourceMetadataPath != s.endpointPath {
	//	mux.Handle(s.protectedResourceMetadataPath, s.protectedResourceMetadataHandler)
	//}
	httpMux := &http.Server{
		Addr:    m.cfg.Address,
		Handler: mux,
	}
	m.httpServer = server.NewStreamableHTTPServer(s, server.WithDisableLocalhostProtection(true), server.WithStreamableHTTPServer(httpMux))
	mux.Handle("/mcp", m.httpServer)
	mux.Handle("/tmp_file", m.httpServer)
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

func (m *Mcp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. 提取 URL 查询参数中的 file (例如: /tmp_file?file=abc.txt)
	fileName := r.URL.Query().Get("file")
	if fileName == "" {
		// 如果支持 POST 表单提交，可以尝试从 FormValue 获取
		fileName = r.FormValue("file")
	}

	if fileName == "" {
		http.Error(w, "missing 'file' parameter", http.StatusBadRequest)
		return
	}

	// 2. 防路径穿越安全处理 (Path Traversal Clean)
	// 清理文件名，防止攻击者传入 "../../etc/passwd" 等危险相对路径
	cleanFileName := filepath.Clean(fileName)

	// 拼接目标文件的完整路径
	filePath := filepath.Join(m.tmpPath, cleanFileName)

	// 3. 严格检查：确保解析后的真实路径必须在 baseDir 允许的范围内
	if !strings.HasPrefix(filePath, filepath.Clean(m.tmpPath)) {
		http.Error(w, "access denied: invalid file path", http.StatusForbidden)
		return
	}

	// 4. 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// 5. 获取文件信息（可选，用于设置 Content-Length 或响应头）
	fileInfo, err := file.Stat()
	if err != nil || fileInfo.IsDir() {
		http.Error(w, "invalid file", http.StatusBadRequest)
		return
	}

	// 6. 返回文件内容
	// 方案 A: 适合普通文件/下载（Go 标准库自动处理 Content-Type 与高效传输）
	http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), file)

	// 方案 B（备选）：如果只需要简单地复制原生字节流
	// w.Header().Set("Content-Type", "application/octet-stream")
	// io.Copy(w, file)
}
