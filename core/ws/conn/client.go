// Package conn 管理 WebSocket 连接：单连接的读写与连接集合的路由分发。
package conn

import (
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

// Client 单条 WebSocket 连接。
// 读与写分别由独立的 goroutine 负责：gorilla/websocket 不允许并发写。
type Client struct {
	id      string
	conn    *websocket.Conn
	hub     *Hub
	handler Handler
	cfg     Config

	send      chan []byte
	done      chan struct{}
	closed    atomic.Bool
	closeOnce sync.Once
}

// NewClient 创建连接
func NewClient(cfg Config, ws *websocket.Conn, hub *Hub, handler Handler) *Client {
	if cfg.SendBuffer <= 0 {
		cfg.SendBuffer = 256
	}
	return &Client{
		id:      uuid.NewString(),
		conn:    ws,
		hub:     hub,
		handler: handler,
		cfg:     cfg,
		send:    make(chan []byte, cfg.SendBuffer),
		done:    make(chan struct{}),
	}
}

// ID 返回连接标识
func (c *Client) ID() string {
	return c.id
}

// Outbound 返回出站消息队列，仅用于测试与调试观察，业务代码不应消费
func (c *Client) Outbound() <-chan []byte {
	return c.send
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
// 队列满时：增量消息（delta）直接丢弃，由后续完整消息收敛；关键消息则断开连接，靠重连恢复。
func (c *Client) enqueue(raw []byte, critical bool) bool {
	if len(raw) == 0 || c.closed.Load() {
		return false
	}
	select {
	case c.send <- raw:
		return true
	default:
		if critical {
			c.Close()
		}
		return false
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

// writePump 唯一的写协程，负责业务消息、心跳与关闭帧
func (c *Client) writePump() {
	ticker := time.NewTicker(c.cfg.PingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case <-c.done:
			c.flush()
			_ = c.conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(c.cfg.WriteWait),
			)
			return
		case raw := <-c.send:
			if err := c.write(raw); err != nil {
				c.Close()
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(c.cfg.WriteWait)); err != nil {
				c.Close()
				return
			}
		}
	}
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
		select {
		case raw := <-c.send:
			if err := c.write(raw); err != nil {
				return
			}
		default:
			return
		}
	}
}
