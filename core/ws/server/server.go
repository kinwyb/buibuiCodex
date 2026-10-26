// Package server 提供 WebSocket 服务：HTTP 升级、鉴权、生命周期，并组装连接层与命令层。
package server

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/ws/bridge"
	"github.com/kinwyb/buibuiCodex/core/ws/command"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// Options 服务启动参数
type Options struct {
	Config    *config.WebSocketConfig // 服务配置
	Bus       *bus.MessageBus         // 消息总线
	Canceller command.Canceller       // 会话取消能力，可为空
	Agents    command.AgentSource     // agent 列表来源，可为空
	Turns     command.TurnReader      // 历史轮次读取能力，为空时 session.history 不可用
	Channel   string                  // 入站消息的渠道标识，默认 ws
	AccountID string                  // 入站消息的账号标识，默认 web
	Logger    *slog.Logger            // 日志
}

// Server WebSocket 服务
type Server struct {
	opts Options
	cfg  *config.WebSocketConfig
	log  *slog.Logger

	hub      *conn.Hub
	bridge   *bridge.Bridge
	commands *command.Registry

	upgrader  websocket.Upgrader
	clientCfg conn.Config

	httpServer *http.Server
	ctx        context.Context
	cancel     context.CancelFunc

	mu      sync.Mutex
	running bool
}

// New 创建 WebSocket 服务
func New(opts Options) *Server {
	if opts.Config == nil {
		opts.Config = &config.WebSocketConfig{}
	}
	opts.Config.SetDefaults()
	if opts.Channel == "" {
		opts.Channel = "ws"
	}
	if opts.AccountID == "" {
		opts.AccountID = "web"
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
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
		opts: opts,
		cfg:  opts.Config,
		log:  opts.Logger,
		hub:  conn.NewHub(),
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
		AccountID:    opts.AccountID,
		DefaultAgent: opts.Config.DefaultAgent,
	})
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

	s.log.Info("websocket server started", "addr", addr, "path", s.cfg.Path)
	return nil
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
	if msgBridge != nil {
		msgBridge.Close()
	}
	if httpServer != nil {
		_ = httpServer.Shutdown(ctx)
	}
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
	if !s.authorize(r) {
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

	c := conn.NewClient(s.clientCfg, rawConn, s.hub, s)
	s.hub.Add(c)
	c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypeHello).WithData(&protocol.HelloData{
		ClientID:     c.ID(),
		Protocol:     protocol.ProtocolVersion,
		Agents:       command.AgentInfos(s.opts.Agents),
		DefaultAgent: s.cfg.DefaultAgent,
		Heartbeat:    int(s.clientCfg.PingPeriod.Seconds()),
	}))

	c.Run()
}

// authorize 校验访问令牌，未配置令牌时放行
func (s *Server) authorize(r *http.Request) bool {
	token := s.cfg.AuthToken
	if token == "" {
		return true
	}
	got := r.URL.Query().Get("token")
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		got = strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
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
