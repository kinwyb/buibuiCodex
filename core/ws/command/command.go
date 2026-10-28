// Package command 实现 ws 的上行命令。
//
// 本文件只做注册与集成：定义依赖、注册表与会话键工具，具体命令按文件拆分——
// 改哪个命令就看哪个文件，不必在一份长文件里翻找。

package command

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/filestore"
	"github.com/kinwyb/buibuiCodex/core/ws/preflow"
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

// TurnReader 会话数据读取能力，由 db.ISession 实现。
// 层级关系为 session -> thread -> turn：list 按 thread 分页，history 按 turn 展开。
type TurnReader interface {
	ThreadList(ctx context.Context, sessionID string) ([]db.SessionThread, error)
	ThreadPage(ctx context.Context, query db.ThreadQuery) ([]db.SessionThread, error)
	// ThreadQueryByID 按 threadID 查单个线程，历史接口据此补线程级累计用量
	ThreadQueryByID(ctx context.Context, threadID string) *db.SessionThread
	ThreadSummaries(ctx context.Context, threadIDs []string) (map[string]db.ThreadSummary, error)
	TurnList(ctx context.Context, query db.TurnQuery) ([]db.SessionTurn, error)
	TurnQueryByID(ctx context.Context, turnID string) *db.SessionTurn
	TurnEventList(ctx context.Context, query db.TurnEventQuery) ([]db.TurnEvent, error)
}

// Deps 命令执行依赖
type Deps struct {
	Bus          *bus.MessageBus
	Hub          *conn.Hub
	Canceller    Canceller
	Agents       AgentSource
	Turns        TurnReader // 历史消息数据源，为空时 session.history 返回未启用
	Channel      string     // 入站消息的渠道标识
	DefaultAgent string     // 前端未指定时使用的 agent
	Preflow      Preflow    // 界面前置路由，为空表示不启用
	// Files 文件存储，用于把 chat.send 里的 fileID 还原成本地路径。
	// 为空时 fileIDs 不可用（只有内联 media 能走通）。
	Files *filestore.Store
}

// Preflow 用户消息进入 agent 之前的界面路由，由 preflow.Router 实现。
//
// 它只处理「该打开哪个页面、要不要再点开某个入口」这类判断，做完就把消息交回主流程；
// 结果的 Note 为空表示消息原样投递。未配置决策模型时这个依赖为空，整条链路不生效。
type Preflow interface {
	Route(ctx context.Context, req *preflow.Request) *preflow.Result
}

// Handler 命令处理函数
type Handler func(ctx context.Context, c *conn.Client, env *protocol.Envelope)

// Registry 命令注册表
type Registry struct {
	deps     Deps
	mu       sync.RWMutex
	handlers map[string]Handler
	// routeQueue 界面前置路由的执行队列；未配置 Preflow 时为空
	routeQueue *preflowQueue
}

// NewRegistry 创建注册表并注册内置命令。
//
// 一个命令一个文件，这份清单就是目录：
//
//	ping.go             ping
//	chat_send.go        chat.send
//	chat_cancel.go      chat.cancel
//	agent_list.go       agent.list
//	session_list.go     session.list
//	session_history.go  session.history
//	turn_replay.go      session.turn.replay
func NewRegistry(deps Deps) *Registry {
	r := &Registry{deps: deps, handlers: make(map[string]Handler)}
	if deps.Preflow != nil && deps.Bus != nil {
		// 队列持有 deps 的副本：Preflow 与 Bus 在运行期不会被替换
		r.routeQueue = newPreflowQueue(deps)
	}
	r.Register(protocol.TypePing, r.ping)
	r.Register(protocol.TypeChatSend, r.chatSend)
	r.Register(protocol.TypeChatCancel, r.chatCancel)
	r.Register(protocol.TypeAgentList, r.agentList)
	r.Register(protocol.TypeSessionList, r.sessionList)
	r.Register(protocol.TypeSessionHistory, r.sessionHistory)
	r.Register(protocol.TypeTurnReplay, r.turnReplay)
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

// SessionKey 构造会话键：Channel_UserID_ChatID。
// AccountID 承载用户身份，因此不同用户的同名 chatID 落在不同会话。
func (r *Registry) SessionKey(userID, chatID string) string {
	key := fmt.Sprintf("%s_%s_%s", r.deps.Channel, userID, chatID)
	return strings.Trim(strings.TrimSpace(key), "_")
}

// ownsSession 判断会话是否属于该用户。
// userID 不允许包含下划线（配置层已校验），因此前缀匹配没有歧义。
func (r *Registry) ownsSession(userID, sessionID string) bool {
	if userID == "" || sessionID == "" {
		return false
	}
	return strings.HasPrefix(sessionID, r.deps.Channel+"_"+userID+"_")
}

// OwnsSession 导出会话归属判断，供 server 层在协议层复用处校验
// （例如 ui.response 回执必须来自该会话所属的连接）。
func (r *Registry) OwnsSession(userID, sessionID string) bool {
	return r.ownsSession(userID, sessionID)
}

// Shutdown 停止前置路由队列，服务关闭时调用。
// 已排队的消息会投完再退出，只是不再做界面动作。
func (r *Registry) Shutdown() {
	r.routeQueue.Shutdown()
}
