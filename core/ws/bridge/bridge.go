// Package bridge 把消息总线上的 agent 事件转换为 ws 信封并按会话分发。
//
// 订阅与投递必须串行且非阻塞：总线 fanout 会阻塞推送，一旦卡住会拖慢整个事件分发。
package bridge

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// Dispatcher 按会话投递信封，由 conn.Hub 实现
type Dispatcher interface {
	Dispatch(sessionID string, env *protocol.Envelope) int
}

// Options 桥接参数
type Options struct {
	Bus        *bus.MessageBus
	Dispatcher Dispatcher
	Channel    string // 需要转发的来源渠道
	ForwardAll bool   // 是否转发所有渠道的事件
	Logger     *slog.Logger
}

// Bridge 事件总线到 ws 连接的转换器
type Bridge struct {
	opts Options

	mu     sync.Mutex
	sub    *bus.EventSubscription
	closed bool
}

// New 创建桥接器
func New(opts Options) *Bridge {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Bridge{opts: opts}
}

// Run 订阅事件总线并持续转发，直到 ctx 结束或订阅被关闭
// Run 订阅事件总线并持续转发，直到 ctx 结束或订阅被关闭。
//
// 必须保证退出时一定注销订阅：总线 fanout 是单协程顺序 push 且不带超时，
// 若订阅残留且无人消费，队列写满后会阻塞整个 fanout，连带拖死其他渠道。
func (b *Bridge) Run(ctx context.Context) {
	defer b.Close()
	defer func() {
		if r := recover(); r != nil {
			b.opts.Logger.Error("websocket bridge panic recovered", "error", r)
		}
	}()

	if b.opts.Bus == nil {
		b.opts.Logger.Error("websocket bridge: message bus is required")
		return
	}
	sub, err := b.opts.Bus.SubscribeEvent()
	if err != nil {
		b.opts.Logger.Error("websocket subscribe events failed", "error", err)
		return
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		sub.Close()
		return
	}
	b.sub = sub
	b.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-sub.Channel:
			if !ok {
				return
			}
			b.forward(event)
		}
	}
}

// Close 释放总线订阅，可重复调用
func (b *Bridge) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	if b.sub != nil {
		b.sub.Close()
		b.sub = nil
	}
}

// forward 过滤来源渠道后转发单个事件。
// 单条消息处理异常不应中断整条转发链路。
func (b *Bridge) forward(event *types.Event) {
	defer func() {
		if r := recover(); r != nil {
			b.opts.Logger.Error("websocket forward panic recovered", "error", r)
		}
	}()

	if event == nil {
		return
	}
	if !b.accept(event.Channel) {
		return
	}
	env := b.convert(event)
	if env == nil {
		return
	}
	if b.opts.Dispatcher != nil {
		b.opts.Dispatcher.Dispatch(event.SessionID, env)
	}
}

// accept 只转发由 ws 渠道产生的会话事件，避免其他渠道的回复串到 Web 端
func (b *Bridge) accept(channel string) bool {
	if b.opts.ForwardAll {
		return true
	}
	return channel == b.opts.Channel
}

// convert 把 types.Event 转换为对前端友好的信封，不直接暴露内部字段
func (b *Bridge) convert(event *types.Event) *protocol.Envelope {
	env := protocol.NewEnvelope(protocol.KindAgent, string(event.Type)).
		WithSession(event.SessionID, event.ReqID).
		WithChunk(event.ChunkIndex)

	data := &protocol.AgentData{
		Event: string(event.Type),
		Agent: agentOf(event),
	}

	if msg := event.Message; msg != nil {
		data.Role = string(msg.Role)
		data.Content = msg.Content
		data.Reasoning = msg.ReasoningContent
		data.ToolCalls = msg.ToolCalls
		data.Media = msg.Media
		// Delta 决定前端是追加还是覆盖，必须与内容语义严格对应：
		// 上游 delta 事件带 IsDetla=true；completed 事件类型不属于增量，
		// 其 Content 是 item 的完整文本，因此 Delta=false 触发前端覆盖收敛。
		data.Delta = msg.IsDetla || protocol.IsDelta(protocol.KindAgent, string(event.Type))
		if msg.ApprovalID != nil {
			data.Approval = &protocol.ApprovalData{
				ID:      fmt.Sprint(msg.ApprovalID),
				Content: msg.Content,
			}
		}
	} else {
		data.Delta = protocol.IsDelta(protocol.KindAgent, string(event.Type))
	}

	if event.Type == types.EventError {
		message := "agent 执行失败"
		if data.Content != "" {
			message = data.Content
		}
		data.Error = &protocol.ErrorBody{Code: "agent_error", Message: message}
	}
	if len(event.Metadata) > 0 {
		data.Meta = event.Metadata
	}

	return env.WithData(data)
}

// agentOf 从事件元数据中提取 agent 名称
func agentOf(event *types.Event) string {
	if event.Metadata == nil {
		return ""
	}
	for _, key := range []string{"agent", "agentName", "agent_name"} {
		if v, ok := event.Metadata[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}
