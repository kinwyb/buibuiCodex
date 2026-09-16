package mcp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/core/db"
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
	TokenMap  map[string]string `description:"授权映射"`
	Host      string            `description:"公开地址"`
	Issuer    string            `description:"授权服务器issuer"`
	Audience  string            `description:"资源标识(本服务对外地址)"`
	JWKSURL   string            `description:"授权服务器JWKS地址"`
}

// applyOAuthDefaults 为 OAuth 相关字段补默认值。
// Resource、Issuer 与 JWKS 地址必须与实际校验逻辑一致，否则会导致令牌校验失败。
func (c *Config) applyOAuthDefaults() {
	if c.Host == "" {
		c.Host = "http://127.0.0.1:9090"
	}
	if c.Audience == "" {
		c.Audience = "http://localhost:9090"
	}
	if c.Issuer == "" {
		c.Issuer = "https://localhost"
	}
	if c.JWKSURL == "" {
		c.JWKSURL = strings.TrimSuffix(c.Issuer, "/") + "/.well-known/jwks.json"
	}
}

// resourceMetadataURL 返回本服务的受保护资源元数据地址（RFC 9728）。
func (c *Config) resourceMetadataURL() string {
	return strings.TrimSuffix(c.Host, "/") + "/.well-known/oauth-protected-resource"
}

type Mcp struct {
	cfg        *Config                      `description:"配置"`
	server     *server.MCPServer            `description:"mcp服务器"`
	httpServer *server.StreamableHTTPServer `description:"http服务"`
	tmpPath    string                       `description:"临时文件目录"`
	tokenCache *edp.TokenCache              `description:"token缓存"`
	oauthMid   *authMiddleware              `description:"oauth中间认证"`
	sess       db.ISession                  `description:"session信息"`
}

func NewMcp(cfg *Config, sess db.ISession) *Mcp {
	if cfg == nil {
		cfg = &Config{
			Name:      "buibui",
			Version:   "1.0.0",
			WorkSpace: filepath.Dir(os.Args[0]),
			BaseURL:   "http://127.0.0.1:8080",
		}
	}
	cfg.applyOAuthDefaults()
	return &Mcp{
		cfg:  cfg,
		sess: sess,
		// 使用授权服务器公布的 JWKS 公钥验签（RS256），不再使用对称密钥。
		oauthMid: NewAuthMiddleware(cfg.Issuer, cfg.Audience, cfg.JWKSURL, cfg.resourceMetadataURL(), cfg.TokenMap),
	}
}

func (m *Mcp) Start(ctx context.Context) error {
	// 初始化edp
	m.tmpPath = filepath.Join(m.cfg.WorkSpace, "tmp")
	serv.TmpDir = m.tmpPath
	if _, perr := os.Stat(m.tmpPath); os.IsNotExist(perr) {
		_ = os.MkdirAll(m.tmpPath, 0755)
	}
	serv.TmpUrl = m.cfg.Host + "/tmp_file"
	cache, err := edp.NewTokenCache(filepath.Join(m.cfg.WorkSpace, "token"), m.cfg.UserMap)
	if err != nil {
		return err
	}
	m.tokenCache = cache
	edp.NewAPIClient(m.cfg.BaseURL)
	serv.UserQueryFun = m.codexThreadParseUser
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
	httpMux := &http.Server{
		Addr:    m.cfg.Address,
		Handler: mux,
	}
	options := []server.StreamableHTTPOption{server.WithDisableLocalhostProtection(true),
		server.WithStreamableHTTPServer(httpMux)}
	oauth := m.oauth()
	if oauth != nil {
		options = append(options, server.WithProtectedResourceMetadata(*oauth), server.WithStreamableHTTPCORS(
			server.WithCORSAllowedOrigins("*"),
			server.WithCORSAllowCredentials(),
			server.WithCORSMaxAge(3600),
		))
	}
	m.httpServer = server.NewStreamableHTTPServer(s, options...)
	mux.Handle("/tmp_file", m)
	mux.Handle("/mcp", m.oauthMid.Middleware(m.httpServer))
	mux.Handle("/", m.httpServer)
	go func() {
		slog.Info("MCP server started")
		if err := m.httpServer.Start(m.cfg.Address); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("启动失败", "error", err)
		}
	}()
	return nil
}

func (m *Mcp) oauth() *server.ProtectedResourceMetadataConfig {
	return &server.ProtectedResourceMetadataConfig{
		Resource:               m.cfg.Audience,
		AuthorizationServers:   []string{m.cfg.Issuer},
		ScopesSupported:        []string{"openid", "profile", "email", "offline_access"},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Boda MCP Server",
	}
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
	if len(cleanFileName) < 7 {
		http.Error(w, "invalid file name", http.StatusBadRequest)
		return
	}
	date := cleanFileName[:8]
	fileDate, te := time.ParseInLocation("20060102", date, time.Local)
	if te != nil {
		http.Error(w, "invalid file name", http.StatusBadRequest)
		return
	}
	cleanFileName = cleanFileName[8:]

	// 拼接目标文件的完整路径
	filePath := filepath.Join(m.tmpPath, fileDate.Format(time.DateOnly), cleanFileName)

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

func (m *Mcp) codexThreadParseUser(ctx context.Context, threadID string) string {
	if m.sess == nil {
		return ""
	}
	thread := m.sess.ThreadQueryByID(ctx, threadID)
	if thread == nil {
		return ""
	}
	session := m.sess.SessionQueryByID(ctx, thread.SessionID)
	if session == nil {
		return ""
	}
	return session.UserID
}
