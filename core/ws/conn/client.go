// Package conn 管理 WebSocket 连接：单连接的读写与连接集合的路由分发。
package conn

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// Config 单连接运行参数
type Config struct {
	PongWait   time.Duration // 读超时，超过未收到 pong 视为连接已死
	PingPeriod time.Duration // 心跳发送间隔
	WriteWait  time.Duration // 单次写超时
	SendBuffer int           // 发送队列长度
}

// Handler 上行命令处理器，由 server 层实现，避免 conn 反向依赖 server
type Handler interface {
	DispatchCommand(c *Client, env *protocol.Envelope)
}

// Identity 连接身份：由握手阶段解析，决定会话归属
type Identity struct {
	// UserID 用户标识，参与会话键构造 ws_<userID>_<chatID>
	UserID string
	// ChatID 当前对话标识，前端不传时后续消息都落在这个对话里
	ChatID string
}

// Client 单条 WebSocket 连接。
// 读与写分别由独立的 goroutine 负责：gorilla/websocket 不允许并发写。
type Client struct {
	id      string
	userID  string
	conn    *websocket.Conn
	hub     *Hub
	handler Handler
	cfg     Config

	chatMu sync.RWMutex
	chatID string

	// out 发送队列：关键帧不退让，增量帧在积压时丢最旧的一条（见 queue.go）
	out *outQueue
	// tap 出站消息的旁路副本，只给测试与调试观察用；生产路径不消费
	tap chan []byte

	done      chan struct{}
	closed    atomic.Bool
	closeOnce sync.Once
}

// NewClient 创建连接。identity 由握手阶段解析，用于会话归属隔离与审计。
func NewClient(cfg Config, ws *websocket.Conn, hub *Hub, handler Handler, identity Identity) *Client {
	if cfg.SendBuffer <= 0 {
		cfg.SendBuffer = 256
	}
	return &Client{
		id:      uuid.NewString(),
		userID:  identity.UserID,
		chatID:  identity.ChatID,
		conn:    ws,
		hub:     hub,
		handler: handler,
		cfg:     cfg,
		out:     newOutQueue(cfg.SendBuffer),
		tap:     make(chan []byte, cfg.SendBuffer),
		done:    make(chan struct{}),
	}
}

// ID 返回连接标识，用于日志与诊断，不代表用户身份
func (c *Client) ID() string {
	return c.id
}

// UserID 返回连接所属用户，会话键为 ws_<userID>_<chatID>
func (c *Client) UserID() string {
	return c.userID
}

// ChatID 返回当前对话标识
func (c *Client) ChatID() string {
	c.chatMu.RLock()
	defer c.chatMu.RUnlock()
	return c.chatID
}

// SetChatID 切换当前对话，后续未显式指定时沿用该值
func (c *Client) SetChatID(chatID string) {
	if chatID == "" {
		return
	}
	c.chatMu.Lock()
	c.chatID = chatID
	c.chatMu.Unlock()
}

// Outbound 返回出站消息的旁路副本，仅用于测试与调试观察，业务代码不应消费。
//
// 副本只收不取：它不参与投递，队列满时也不会被丢弃策略影响，
// 因此观察它不会改变真实发送行为。
func (c *Client) Outbound() <-chan []byte {
	return c.tap
}

// Run 启动读写协程，读协程在当前 goroutine 中阻塞直到连接结束
func (c *Client) Run() {
	go c.writePump()
	c.readPump()
}

// SendEnvelope 发送一条信封消息
func (c *Client) SendEnvelope(env *protocol.Envelope) {
	if env == nil {
		return
	}
	c.enqueue(protocol.EncodeEnvelope(env), !protocol.IsDelta(env.Kind, env.Type))
}

// ReplyAck 回复受理确认，回填请求 ID
func (c *Client) ReplyAck(env *protocol.Envelope, data *protocol.AckData) {
	ack := protocol.NewEnvelope(protocol.KindSystem, protocol.TypeAck).WithData(data)
	if env != nil {
		ack.WithReplyTo(env.ID)
	}
	c.SendEnvelope(ack)
}

// ReplyError 回复错误
func (c *Client) ReplyError(env *protocol.Envelope, body *protocol.ErrorBody) {
	if body == nil {
		body = &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "unknown error"}
	}
	e := protocol.NewEnvelope(protocol.KindSystem, protocol.TypeError).WithData(body)
	if env != nil {
		e.WithReplyTo(env.ID)
		e.SessionID = env.SessionID
	}
	c.SendEnvelope(e)
}

// Close 标记关闭并通知写协程收尾，可重复调用
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		close(c.done)
	})
}

// enqueue 投递一条已序列化的消息。
//
// 背压策略（见 queue.go）：队列满时先丢掉最旧的一条增量帧给新帧让位；
// 关键帧不会被主动丢弃，只有队列里全是关键帧、腾不出位置时才断开连接，
// 靠前端重连 + 历史补发恢复。
func (c *Client) enqueue(raw []byte, critical bool) bool {
	if len(raw) == 0 || c.closed.Load() {
		return false
	}
	ok, dropped := c.out.push(outbound{raw: raw, critical: critical})
	if dropped > 0 {
		c.reportDrop()
	}
	if !ok {
		if critical {
			c.Close()
		}
		return false
	}
	c.tapRaw(raw)
	return true
}

// reportDrop 记录一次「增量帧被丢弃」。
// 限流打日志：这是「界面看起来不流式」唯一的现场证据，全量打会把日志冲垮。
func (c *Client) reportDrop() {
	queued, dropped := c.out.stats()
	if dropped == 1 || dropped%100 == 0 {
		slog.Warn("ws 发送积压，已丢弃最旧的增量帧",
			"client", c.id,
			"dropped", dropped,
			"queued", queued,
			"capacity", c.cfg.SendBuffer)
	}
}

// tapRaw 给旁路副本发一份；满了就跳过，不能影响真实投递
func (c *Client) tapRaw(raw []byte) {
	select {
	case c.tap <- raw:
	default:
	}
}

// readPump 读取上行消息，连接结束时从 Hub 注销
func (c *Client) readPump() {
	_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.PongWait))
	c.conn.SetReadLimit(protocol.MaxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(c.cfg.PongWait))
	})

	defer func() {
		c.hub.Remove(c)
		c.Close()
		_ = c.conn.Close()
	}()

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		env, body := protocol.DecodeEnvelope(raw)
		if body != nil {
			c.ReplyError(env, body)
			continue
		}
		if c.handler != nil {
			c.handler.DispatchCommand(c, env)
		}
	}
}

// writePump 唯一的写协程，负责业务消息、心跳与关闭帧。
//
// 每轮先把队列里排队的写完，再等「有新数据 / 心跳 / 关闭」三者之一：
// 关闭要优先处理，避免积压时迟迟不响应 done。
func (c *Client) writePump() {
	ticker := time.NewTicker(c.cfg.PingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case <-c.done:
			c.shutdown()
			return
		default:
		}

		if item, ok := c.out.pop(); ok {
			if err := c.write(item.raw); err != nil {
				c.Close()
				return
			}
			continue
		}

		select {
		case <-c.done:
			c.shutdown()
			return
		case <-c.out.wake():
			// 有新数据，回到循环顶部取
		case <-ticker.C:
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(c.cfg.WriteWait)); err != nil {
				c.Close()
				return
			}
		}
	}
}

// shutdown 关闭前尽力把队列里剩下的发完，再走正常的关闭帧
func (c *Client) shutdown() {
	c.flush()
	_ = c.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(c.cfg.WriteWait),
	)
}

// write 写入一条文本帧
func (c *Client) write(raw []byte) error {
	if len(raw) == 0 {
		return nil
	}
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteWait)); err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, raw)
}

// flush 关闭前尽力把队列里的消息发完
func (c *Client) flush() {
	deadline := time.Now().Add(c.cfg.WriteWait)
	for time.Now().Before(deadline) {
		item, ok := c.out.pop()
		if !ok {
			return
		}
		if err := c.write(item.raw); err != nil {
			return
		}
	}
}
