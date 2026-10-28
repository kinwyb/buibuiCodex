// Package server 提供 WebSocket 服务：HTTP 升级、鉴权、生命周期，并组装连接层与命令层。
package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/ws/bridge"
	"github.com/kinwyb/buibuiCodex/core/ws/command"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/filestore"
	"github.com/kinwyb/buibuiCodex/core/ws/preflow"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// defaultUploadLimit 兜底的单文件上限，与 config.DefaultMaxUploadSize 保持一致
const defaultUploadLimit int64 = 50 << 20

// Authenticator 由 token 解析连接身份。
//
// 返回 (userID, chatID, ok)：
//   - ok=false 表示拒绝连接（HTTP 401）
//   - chatID 为空时由服务端生成，客户端可通过 hello 拿到
//   - 需要「一个 token 固定对应一个对话」时，直接在这里返回 chatID
//
// userID 会拼进会话键 ws_<userID>_<chatID>，必须是合法标识（见 validUserID）。
type Authenticator func(token string) (userID string, chatID string, ok bool)

// DefaultAuthenticator 默认实现：token 即用户标识，对话由服务端生成。
// TODO 接入真实鉴权（查库/缓存/JWT）时替换本函数，可同时返回该用户当前的对话 ID。
func DefaultAuthenticator(token string) (string, string, bool) {
	//if !validUserID(token) {
	//	return "", "", false
	//}
	//return token, "", true
	return "aibAnLYMK4TgdOkkhCCYZabAtOGE64ouXFV", "007928", true
}

// Options 服务启动参数
type Options struct {
	Config        *config.WebSocketConfig // 服务配置
	Bus           *bus.MessageBus         // 消息总线
	Canceller     command.Canceller       // 会话取消能力，可为空
	Agents        command.AgentSource     // agent 列表来源，可为空
	Turns         command.TurnReader      // 历史轮次读取能力，为空时 session.history 不可用
	Authenticator Authenticator           // 身份解析函数，为空时使用 DefaultAuthenticator
	Channel       string                  // 入站消息的渠道标识，默认 ws
	Logger        *slog.Logger            // 日志
	// UIRequestTimeout 等待浏览器回执的超时，为 0 时用 defaultUIRequestTimeout。
	// 界面工具（ui_observe / ui_act）是同步等待的，超时意味着页面没响应或已经关闭。
	UIRequestTimeout time.Duration
	// PreflowDecider 界面前置路由使用的决策模型入口，正常传 *decision.Client。
	// 为空且 websocket.page_router.enabled 时，按配置里的地址与密钥自己建一个；
	// 留出这个注入点是为了测试与自定义网关。
	PreflowDecider preflow.Decider
	// Files 文件存储。为空时按 cfg.FileDir / cfg.FileTTL 自建，
	// 注入的场景是测试，或多个 Server 实例共享同一目录。
	Files *filestore.Store
}

// Server WebSocket 服务
type Server struct {
	opts Options
	cfg  *config.WebSocketConfig
	log  *slog.Logger

	hub      *conn.Hub
	bridge   *bridge.Bridge
	commands *command.Registry
	auth     Authenticator

	upgrader  websocket.Upgrader
	clientCfg conn.Config

	httpServer *http.Server
	ctx        context.Context
	cancel     context.CancelFunc

	mu      sync.Mutex
	running bool

	// 界面操作的反向 RPC：请求 ID -> 等待中的回执
	uiMu      sync.Mutex
	uiPending map[string]*uiPending
	uiTimeout time.Duration

	// 文件收发：存储按需初始化（见 files.go）
	fileStore *filestore.Store
	filesOnce sync.Once
	// syncHandlerRegistered 记录本通道是否已接管 bus 的同步请求，
	// 重复 Start / Stop 时避免反复订阅与误删别人的 handler
	syncHandlerRegistered bool
}

// New 创建 WebSocket 服务
func New(opts Options) *Server {
	if opts.Config == nil {
		opts.Config = &config.WebSocketConfig{}
	}
	opts.Config.SetDefaults()
	if opts.Channel == "" {
		// 渠道标识决定会话键前缀，也决定 agent 的 send_file 请求回流到哪：
		// 用 "ws" 才能把文件回给浏览器；用其它渠道名（例如共用企业微信会话）
		// 时，文件会被对应通道接管，web 端收不到。
		opts.Channel = opts.Config.Channel
	}
	if opts.Channel == "" {
		opts.Channel = "ws"
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Authenticator == nil {
		opts.Authenticator = DefaultAuthenticator
	}

	pingPeriod := time.Duration(opts.Config.HeartbeatInterval) * time.Second
	if pingPeriod <= 0 {
		pingPeriod = 30 * time.Second
	}
	pongWait := time.Duration(opts.Config.ReadTimeout) * time.Second
	if pongWait <= pingPeriod {
		// 读超时必须长于心跳间隔，否则连接会被误判为假死
		pongWait = pingPeriod * 2
	}
	writeWait := time.Duration(opts.Config.WriteTimeout) * time.Second
	if writeWait <= 0 {
		writeWait = 10 * time.Second
	}

	s := &Server{
		opts:      opts,
		cfg:       opts.Config,
		log:       opts.Logger,
		hub:       conn.NewHub(),
		auth:      opts.Authenticator,
		uiPending: make(map[string]*uiPending),
		uiTimeout: opts.UIRequestTimeout,
		clientCfg: conn.Config{
			PongWait:   pongWait,
			PingPeriod: pingPeriod,
			WriteWait:  writeWait,
			SendBuffer: opts.Config.SendBuffer,
		},
	}
	s.commands = command.NewRegistry(command.Deps{
		Bus:          opts.Bus,
		Hub:          s.hub,
		Canceller:    opts.Canceller,
		Agents:       opts.Agents,
		Turns:        opts.Turns,
		Channel:      opts.Channel,
		DefaultAgent: opts.Config.DefaultAgent,
		Preflow:      s.newPreflow(),
		// 与 HTTP 端点共用同一个存储实例：上传拿到的 fileID 必须在 chat.send 里认得
		Files: s.files(),
	})
	// 浏览器回执走 client 域，注册在这里由 server 认领等待中的请求
	s.commands.Register(protocol.TypeUIResponse, s.handleUIResponse)
	s.upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin:     s.checkOrigin,
	}
	return s
}

// Start 启动 HTTP 服务与消息桥接，未启用时直接返回
func (s *Server) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		s.log.Info("websocket server disabled")
		return nil
	}
	if s.opts.Bus == nil {
		return errors.New("websocket: message bus is required")
	}

	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	mux := http.NewServeMux()
	mux.HandleFunc(s.cfg.Path, s.handleWebSocket)
	// 文件端点挂在同一个端口上：字节走 HTTP，ws 只传 fileID（见 files.go）
	s.registerFileRoutes(mux)

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	ctx, cancel := context.WithCancel(ctx)
	s.ctx, s.cancel = ctx, cancel
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	s.bridge = bridge.New(bridge.Options{
		Bus:        s.opts.Bus,
		Dispatcher: s.hub,
		Channel:    s.opts.Channel,
		ForwardAll: s.cfg.ForwardAllChannels,
		Logger:     s.log,
		Coalesce: bridge.CoalesceConfig{
			Enabled:     s.cfg.StreamCoalesceEnabled(),
			Interval:    time.Duration(s.cfg.StreamCoalesceMs) * time.Millisecond,
			MaxBytes:    s.cfg.StreamCoalesceMaxBytes,
			MaxSegments: 128, // 跟踪片段数的兜底上限，正常一轮不会同时开这么多段
		},
	})
	httpServer := s.httpServer
	msgBridge := s.bridge
	s.mu.Unlock()

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("websocket server stopped", "addr", addr, "error", err)
			cancel()
		}
	}()

	go msgBridge.Run(ctx)

	// 接管本通道的同步请求：agent 的 send_file 按 channel 路由到这里，
	// 由 ws 登记文件并把下载地址推回浏览器。换个渠道标识就等于关掉这个能力。
	if err := s.attachSyncHandler(); err != nil {
		s.log.Error("websocket register sync handler failed", "channel", s.opts.Channel, "error", err)
	}
	go s.gcLoop(ctx)

	s.log.Info("websocket server started", "addr", addr, "path", s.cfg.Path,
		"files", s.fileBase(), "channel", s.opts.Channel)
	return nil
}

// attachSyncHandler 把自己注册为当前渠道的同步请求处理器
func (s *Server) attachSyncHandler() error {
	s.mu.Lock()
	if s.syncHandlerRegistered || s.opts.Bus == nil {
		s.mu.Unlock()
		return nil
	}
	s.syncHandlerRegistered = true
	s.mu.Unlock()

	if err := s.opts.Bus.SubscribeSyncHandler(s.opts.Channel, s.handleSyncRequest); err != nil {
		s.mu.Lock()
		s.syncHandlerRegistered = false
		s.mu.Unlock()
		return err
	}
	return nil
}

// detachSyncHandler 撤销订阅，避免服务停了还把请求吸过来
func (s *Server) detachSyncHandler() {
	s.mu.Lock()
	registered := s.syncHandlerRegistered
	s.syncHandlerRegistered = false
	s.mu.Unlock()
	if registered && s.opts.Bus != nil {
		_ = s.opts.Bus.UnsubscribeSyncHandler(s.opts.Channel)
	}
}

// Stop 停止服务并关闭所有连接
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	cancel := s.cancel
	httpServer := s.httpServer
	msgBridge := s.bridge
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	// 先撤同步请求订阅：服务要停了，send_file 不该再往这里路由
	s.detachSyncHandler()
	if msgBridge != nil {
		msgBridge.Close()
	}
	if httpServer != nil {
		_ = httpServer.Shutdown(ctx)
	}
	// 前置路由队列最后停：已排队的消息投完再退，不再发起新的界面动作
	s.commands.Shutdown()
	s.hub.CloseAll("server shutting down")
	s.log.Info("websocket server stopped")
	return nil
}

// HandleCommand 注册上行命令，供业务侧扩展新的消息类型
func (s *Server) HandleCommand(typ string, h command.Handler) {
	s.commands.Register(typ, h)
}

// DispatchCommand 实现 conn.Handler，把上行消息交给命令层
func (s *Server) DispatchCommand(c *conn.Client, env *protocol.Envelope) {
	s.commands.Dispatch(s.commandContext(), c, env)
}

// Broadcast 主动推送一条消息给所有连接，用于非 agent 消息（通知、定时任务等）
func (s *Server) Broadcast(kind protocol.Kind, typ string, data any) int {
	if kind == protocol.KindClient {
		return 0
	}
	return s.hub.Broadcast(protocol.NewEnvelope(kind, typ).WithData(data))
}

// ClientCount 当前连接数
func (s *Server) ClientCount() int {
	return s.hub.Count()
}

// SessionCount 当前被订阅的会话数
func (s *Server) SessionCount() int {
	return s.hub.SessionCount()
}

// handleWebSocket 处理连接升级
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID, chatID, ok := s.authenticate(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		// 前端只带 token 时，由服务端确定当前对话
		chatID = uuid.NewString()
	} else if len(chatID) > maxChatIDLength {
		s.log.Warn("websocket chatID too long", "user", userID, "length", len(chatID))
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.cfg.MaxClients > 0 && s.hub.Count() >= s.cfg.MaxClients {
		http.Error(w, "too many clients", http.StatusServiceUnavailable)
		return
	}
	rawConn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Warn("websocket upgrade failed", "remote", r.RemoteAddr, "error", err)
		return
	}

	c := conn.NewClient(s.clientCfg, rawConn, s.hub, s, conn.Identity{UserID: userID, ChatID: chatID})
	s.hub.Add(c)
	c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypeHello).WithData(&protocol.HelloData{
		ClientID:     c.ID(),
		UserID:       userID,
		ChatID:       chatID,
		Protocol:     protocol.ProtocolVersion,
		Agents:       command.AgentInfos(s.opts.Agents),
		DefaultAgent: s.cfg.DefaultAgent,
		Heartbeat:    int(s.clientCfg.PingPeriod.Seconds()),
	}))

	c.Run()
}

// maxChatIDLength 对话标识最大长度，防止异常值进入会话键
const maxChatIDLength = 128

// authenticate 解析连接身份，失败时返回 401
func (s *Server) authenticate(r *http.Request) (userID string, chatID string, ok bool) {
	return s.auth(s.requestToken(r))
}

// requestToken 从 Header 或 query 中提取 token，Header 优先
func (s *Server) requestToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return r.URL.Query().Get("token")
}

// validUserID userID 会拼进会话键 ws_<userID>_<chatID>，
// 只允许字母、数字和短横线：禁止下划线是为了让归属前缀判断没有歧义。
func validUserID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, ch := range id {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9', ch == '-':
		default:
			return false
		}
	}
	return true
}

// checkOrigin 校验来源，未配置白名单时放行（便于本地联调）
func (s *Server) checkOrigin(r *http.Request) bool {
	allow := s.cfg.AllowOrigins
	if len(allow) == 0 {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	host := origin
	if i := strings.Index(origin, "://"); i >= 0 {
		host = origin[i+3:]
	}
	for _, item := range allow {
		if item == "*" || strings.EqualFold(item, origin) || strings.EqualFold(item, host) {
			return true
		}
	}
	s.log.Warn("websocket origin rejected", "origin", origin)
	return false
}

// commandContext 返回命令执行使用的上下文，未启动时退化为 Background
func (s *Server) commandContext() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}
