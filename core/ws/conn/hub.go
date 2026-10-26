package conn

import (
	"sync"
	"sync/atomic"

	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// Hub 维护所有 WebSocket 连接，并按会话把消息路由给订阅者。
type Hub struct {
	mu       sync.RWMutex
	clients  map[string]*Client            // clientID -> Client
	sessions map[string]map[string]*Client // sessionID -> clientID -> Client
	seq      atomic.Int64                  // 全局单调序号
}

// NewHub 创建连接集合
func NewHub() *Hub {
	return &Hub{
		clients:  make(map[string]*Client),
		sessions: make(map[string]map[string]*Client),
	}
}

// Add 注册连接
func (h *Hub) Add(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.id] = c
}

// Remove 注销连接，并清理其所有会话订阅
func (h *Hub) Remove(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c.id)
	for sessionID, set := range h.sessions {
		delete(set, c.id)
		if len(set) == 0 {
			delete(h.sessions, sessionID)
		}
	}
}

// Subscribe 让连接订阅指定会话
func (h *Hub) Subscribe(c *Client, sessionID string) {
	if sessionID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.sessions[sessionID]
	if !ok {
		set = make(map[string]*Client)
		h.sessions[sessionID] = set
	}
	set[c.id] = c
}

// Dispatch 把消息投递给订阅了该会话的连接，返回投递数量
func (h *Hub) Dispatch(sessionID string, env *protocol.Envelope) int {
	if sessionID == "" || env == nil {
		return 0
	}
	h.mu.RLock()
	set := h.sessions[sessionID]
	targets := make([]*Client, 0, len(set))
	for _, c := range set {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	if len(targets) == 0 {
		return 0
	}
	return h.deliver(targets, env)
}

// Broadcast 把消息投递给所有连接，返回投递数量
func (h *Hub) Broadcast(env *protocol.Envelope) int {
	if env == nil {
		return 0
	}
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.clients))
	for _, c := range h.clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	if len(targets) == 0 {
		return 0
	}
	return h.deliver(targets, env)
}

// deliver 统一序列化一次后分发给多个连接
func (h *Hub) deliver(targets []*Client, env *protocol.Envelope) int {
	env.Seq = h.seq.Add(1)
	raw := protocol.EncodeEnvelope(env)
	if raw == nil {
		return 0
	}
	critical := !protocol.IsDelta(env.Kind, env.Type)
	sent := 0
	for _, c := range targets {
		if c.enqueue(raw, critical) {
			sent++
		}
	}
	return sent
}

// Count 当前连接数
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// SessionCount 当前被订阅的会话数
func (h *Hub) SessionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}

// CloseAll 通知所有连接后逐个关闭
func (h *Hub) CloseAll(reason string) {
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.clients))
	for _, c := range h.clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	if len(targets) > 0 {
		env := protocol.NewEnvelope(protocol.KindSystem, protocol.TypeShutdown).
			WithData(&protocol.ShutdownData{Reason: reason})
		h.Broadcast(env)
	}
	for _, c := range targets {
		c.Close()
	}
}
