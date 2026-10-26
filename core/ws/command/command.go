// Package command 实现 ws 的上行命令：命令注册表与内置命令。
package command

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// Canceller 会话取消能力，由 core.Manager 实现
type Canceller interface {
	Cancel(sessionKey string)
}

// AgentSource agent 列表来源，由 core.Manager 实现
type AgentSource interface {
	ListAgents() []string
}

// TurnReader 历史轮次读取能力，由 db.ISession 实现。
// 层级关系为 session -> thread -> turn。
type TurnReader interface {
	ThreadList(ctx context.Context, sessionID string) ([]db.SessionThread, error)
	TurnList(ctx context.Context, query db.TurnQuery) ([]db.SessionTurn, error)
}

// Deps 命令执行依赖
type Deps struct {
	Bus          *bus.MessageBus
	Hub          *conn.Hub
	Canceller    Canceller
	Agents       AgentSource
	Turns        TurnReader // 历史消息数据源，为空时 session.history 返回未启用
	Channel      string     // 入站消息的渠道标识
	AccountID    string     // 入站消息的账号标识
	DefaultAgent string     // 前端未指定时使用的 agent
}

// Handler 命令处理函数
type Handler func(ctx context.Context, c *conn.Client, env *protocol.Envelope)

// Registry 命令注册表
type Registry struct {
	deps     Deps
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewRegistry 创建注册表并注册内置命令
func NewRegistry(deps Deps) *Registry {
	r := &Registry{deps: deps, handlers: make(map[string]Handler)}
	r.Register(protocol.TypePing, r.ping)
	r.Register(protocol.TypeChatSend, r.chatSend)
	r.Register(protocol.TypeChatCancel, r.chatCancel)
	r.Register(protocol.TypeAgentList, r.agentList)
	r.Register(protocol.TypeSessionHistory, r.sessionHistory)
	return r
}

// Register 注册或覆盖一条上行命令
func (r *Registry) Register(typ string, h Handler) {
	if typ == "" || h == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[typ] = h
}

// Dispatch 校验并分发上行消息
func (r *Registry) Dispatch(ctx context.Context, c *conn.Client, env *protocol.Envelope) {
	if env == nil {
		return
	}
	if env.Kind != protocol.KindClient {
		c.ReplyError(env, &protocol.ErrorBody{
			Code:    protocol.CodeUnsupported,
			Message: "服务端只接受 kind=client 的上行消息",
		})
		return
	}
	r.mu.RLock()
	h := r.handlers[env.Type]
	r.mu.RUnlock()
	if h == nil {
		c.ReplyError(env, &protocol.ErrorBody{
			Code:    protocol.CodeUnsupported,
			Message: "未支持的消息类型: " + env.Type,
		})
		return
	}
	h(ctx, c, env)
}

// AgentInfos 汇总可用的 agent 列表
func AgentInfos(src AgentSource) []protocol.AgentInfo {
	if src == nil {
		return nil
	}
	names := src.ListAgents()
	infos := make([]protocol.AgentInfo, 0, len(names))
	for _, name := range names {
		infos = append(infos, protocol.AgentInfo{Name: name})
	}
	return infos
}

// SessionKey 与 types.InputMessage.SessionID 保持一致的会话键
func (r *Registry) SessionKey(chatID string) string {
	key := fmt.Sprintf("%s_%s_%s", r.deps.Channel, r.deps.AccountID, chatID)
	return strings.Trim(strings.TrimSpace(key), "_")
}

// ping 应用层心跳，前端可借此测量往返延迟
func (r *Registry) ping(_ context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.PingData
	_ = env.DecodeData(&data)
	c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypePong).
		WithReplyTo(env.ID).
		WithData(&protocol.PongData{ClientTS: data.TS, ServerTS: time.Now().UnixMilli()}))
}

// chatSend 把用户消息投递到消息总线，由 core.Manager 交给 agent 处理
func (r *Registry) chatSend(ctx context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.ChatSendData
	if err := env.DecodeData(&data); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败", Detail: err.Error()})
		return
	}
	if strings.TrimSpace(data.Content) == "" && len(data.Media) == 0 {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "content 与 media 不能同时为空"})
		return
	}
	if data.Agent != "" && r.deps.Agents != nil {
		if !slices.Contains(r.deps.Agents.ListAgents(), data.Agent) {
			c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeAgentNotFound, Message: "指定的 agent 不存在", Detail: data.Agent})
			return
		}
	}

	chatID := strings.TrimSpace(data.ChatID)
	if chatID == "" {
		chatID = uuid.NewString()
	}
	msg := &types.InputMessage{
		ID:        uuid.NewString(),
		Session:   data.SessionID,
		Channel:   r.deps.Channel,
		AccountID: r.deps.AccountID,
		SenderID:  c.ID(),
		ChatID:    chatID,
		Content:   data.Content,
		Media:     data.Media,
		Metadata:  data.Metadata,
		AgentName: data.Agent,
		Timestamp: time.Now(),
	}
	sessionID := msg.SessionID()

	// 先订阅再发送，避免 agent 首帧早于订阅导致丢失
	if r.deps.Hub != nil {
		r.deps.Hub.Subscribe(c, sessionID)
	}

	if err := r.deps.Bus.PublishInput(ctx, msg); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "消息投递失败", Detail: err.Error()})
		return
	}
	c.ReplyAck(env, &protocol.AckData{OK: true, SessionID: sessionID, ReqID: msg.ID})
}

// chatCancel 中断指定会话当前运行中的 agent
func (r *Registry) chatCancel(_ context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.ChatCancelData
	if err := env.DecodeData(&data); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败", Detail: err.Error()})
		return
	}
	sessionID := strings.TrimSpace(data.SessionID)
	if sessionID == "" && strings.TrimSpace(data.ChatID) != "" {
		sessionID = r.SessionKey(data.ChatID)
	}
	if sessionID == "" {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "缺少 sessionID 或 chatID"})
		return
	}
	if r.deps.Canceller == nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "服务未提供会话取消能力"})
		return
	}
	r.deps.Canceller.Cancel(sessionID)
	c.ReplyAck(env, &protocol.AckData{OK: true, SessionID: sessionID})
}

// agentList 返回可用 agent 列表
func (r *Registry) agentList(_ context.Context, c *conn.Client, env *protocol.Envelope) {
	c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypeAgentList).
		WithReplyTo(env.ID).
		WithData(&protocol.AgentListData{
			Agents:       AgentInfos(r.deps.Agents),
			DefaultAgent: r.deps.DefaultAgent,
		}))
}

// sessionHistory 拉取会话历史，按 session -> thread -> turn 的层级返回。
// 三种用法：
//   - 打开会话：只传 sessionID，返回全部线程与轮次；
//   - 向上翻页：额外传 beforeSeq（取上一页第一条的 seq）；
//   - 断线补发：额外传 since（本地最后一条轮次的 createdAt）。
func (r *Registry) sessionHistory(ctx context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.SessionHistoryData
	if err := env.DecodeData(&data); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败", Detail: err.Error()})
		return
	}

	sessionID := strings.TrimSpace(data.SessionID)
	if sessionID == "" && strings.TrimSpace(data.ChatID) != "" {
		sessionID = r.SessionKey(data.ChatID)
	}
	if sessionID == "" {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "缺少 sessionID 或 chatID"})
		return
	}
	if r.deps.Turns == nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "服务未启用历史消息"})
		return
	}

	// 打开会话即订阅，保证断线重连后能继续收到该会话的实时事件
	if r.deps.Hub != nil {
		r.deps.Hub.Subscribe(c, sessionID)
	}

	result, err := r.loadHistory(ctx, sessionID, data)
	if err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "查询历史消息失败", Detail: err.Error()})
		return
	}

	c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypeSessionHistory).
		WithReplyTo(env.ID).
		WithSession(sessionID, "").
		WithData(result))
}

// loadHistory 按 thread 分组查询轮次
func (r *Registry) loadHistory(ctx context.Context, sessionID string, data protocol.SessionHistoryData) (*protocol.SessionHistoryResult, error) {
	query := db.TurnQuery{
		SessionID: sessionID,
		ThreadID:  strings.TrimSpace(data.ThreadID),
		BeforeSeq: data.BeforeSeq,
		Limit:     data.Limit,
	}
	if data.Since > 0 {
		query.Since = time.UnixMilli(data.Since)
	}
	limit := query.NormalizeLimit()
	result := &protocol.SessionHistoryResult{SessionID: sessionID}

	// 指定线程：只返回该线程
	if query.ThreadID != "" {
		turns, err := r.deps.Turns.TurnList(ctx, query)
		if err != nil {
			return nil, err
		}
		result.Threads = append(result.Threads, protocol.HistoryThread{
			ThreadID: query.ThreadID,
			Turns:    toHistoryTurns(turns),
			HasMore:  len(turns) >= limit,
		})
		return result, nil
	}

	// 未指定线程：session -> thread -> turn
	threads, err := r.deps.Turns.ThreadList(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	for _, thread := range threads {
		q := query
		q.ThreadID = thread.ThreadID
		turns, err := r.deps.Turns.TurnList(ctx, q)
		if err != nil {
			return nil, err
		}
		if len(turns) == 0 {
			continue
		}
		result.Threads = append(result.Threads, protocol.HistoryThread{
			ThreadID:   thread.ThreadID,
			Agent:      thread.Agent,
			CreatedAt:  millis(thread.CreateTime),
			LastUpdate: millis(thread.LastUpdate),
			Turns:      toHistoryTurns(turns),
			HasMore:    len(turns) >= limit,
		})
	}

	// 兼容没有 thread 记录的历史数据
	if len(result.Threads) == 0 {
		q := query
		q.ThreadID = ""
		turns, err := r.deps.Turns.TurnList(ctx, q)
		if err != nil {
			return nil, err
		}
		if len(turns) > 0 {
			result.Threads = append(result.Threads, protocol.HistoryThread{
				Turns:   toHistoryTurns(turns),
				HasMore: len(turns) >= limit,
			})
		}
	}

	return result, nil
}

// toHistoryTurns 转换数据库轮次为协议结构
func toHistoryTurns(list []db.SessionTurn) []protocol.HistoryTurn {
	out := make([]protocol.HistoryTurn, 0, len(list))
	for _, t := range list {
		item := protocol.HistoryTurn{
			Seq:       t.Seq,
			ReqID:     t.ReqID,
			ThreadID:  t.ThreadID,
			TurnID:    t.TurnID,
			Agent:     t.Agent,
			Question:  t.Question,
			Answer:    t.Answer,
			Completed: t.IsCompleted,
			CreatedAt: millis(t.CreatedAt),
		}
		if t.CompletedAt != nil {
			item.CompletedAt = millis(*t.CompletedAt)
		}
		out = append(out, item)
	}
	return out
}

// millis 转毫秒时间戳，零值返回 0 而不是负数
func millis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
