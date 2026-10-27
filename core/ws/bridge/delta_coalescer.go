// Package bridge 本文件实现「增量帧合并」：把同一轮、同一片段的 message_delta / reasoning_delta
// 在一个时间窗口内拼成更大的增量帧再发出去，只降帧数，不升带宽。
//
// 为什么需要（现象在浏览器，根因在帧数）：
//
//	上游逐 token 发 delta，一轮回答动辄上百帧，而前端每帧都要重新渲染整段 Markdown。
//	渲染慢过帧间隔后浏览器不再从 socket 读，conn 的发送队列（默认 256）填满，
//	按背压策略丢掉最旧的增量帧（见 conn/queue.go），界面就退化成
//	「等 message_completed 的全量文本才一次性刷出来」——现象就是没有流式。
//	把帧数压一个量级，比让浏览器渲染得更快便宜得多，队列也就基本不再填满。
//
// 三条硬约束（也是与 channel.StreamBuffer 的区别，见 README §8.4）：
//  1. 合并的是**增量**，不是累积全文。一帧的内容仍然是「要追加的那一段」，
//     带宽保持 O(n)；改成每帧发全文会退化成 O(n²)，长回答下反而更贵。
//  2. 按**片段**分组（sessionID + reqID + eventID + 类型），不按 reqID。
//     一轮里思考与正文的多个 eventID 会交错，按 reqID 合并会把两段混成一个。
//  3. 不改写上游事件对象。总线 fanout 会把同一个 *types.Event（以及它的 *types.Message）
//     指针发给所有订阅者（ws、企业微信、日志），就地改 Content 会污染别的渠道与落库。
//
// 什么时候不生效：没有 Start 起窗口、或配置里关掉合并时，一律逐帧直发，与改造前完全一致。
package bridge

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// 合并参数默认值
const (
	defaultCoalesceInterval = 50 * time.Millisecond
	defaultCoalesceMaxBytes = 4096
	defaultCoalesceMaxSegs  = 128
)

// 片段类型，对应两种可合并的 delta 事件
const (
	coalesceKindMessage   = "message"
	coalesceKindReasoning = "reasoning"
)

// CoalesceConfig 增量帧合并参数，由 websocket 配置传进来
type CoalesceConfig struct {
	// Enabled 关掉后逐帧直发，行为与合并前一致
	Enabled bool
	// Interval 合并窗口；<=0 用默认 50ms
	Interval time.Duration
	// MaxBytes 单片段攒到多少字节就立刻落地（长帧与延迟保护）；<=0 用默认 4096
	MaxBytes int
	// MaxSegments 同时跟踪的片段数上限，超了就整体落地一次；<=0 用默认 128
	MaxSegments int
	// Logger 只在「提前落地」这类少见路径上记一行
	Logger *slog.Logger
}

// withDefaults 补齐未配置的字段（值接收者：调用方传的是副本）
func (c CoalesceConfig) withDefaults() CoalesceConfig {
	if c.Interval <= 0 {
		c.Interval = defaultCoalesceInterval
	}
	if c.MaxBytes <= 0 {
		c.MaxBytes = defaultCoalesceMaxBytes
	}
	if c.MaxSegments <= 0 {
		c.MaxSegments = defaultCoalesceMaxSegs
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	return c
}

// coalesceKey 一个流式片段：合并只发生在同一片段内部
type coalesceKey struct {
	sessionID string
	reqID     string
	eventID   string
	kind      string
}

// segmentState 单片段的合并状态
type segmentState struct {
	// accum 该片段已经对外呈现的全部文本（已发出的 + 窗口里还没发的）。
	// 留着它是为了识别「上游把累积全文当增量发」，切出真正的尾巴再拼接。
	accum string
	// pending 窗口里攒着、尚未发出的增量
	pending strings.Builder
	// last 最近一条事件，作为合并帧的元数据来源（agent / metadata / chunk 等）
	last *types.Event
}

// advance 并入一条增量，返回真正需要追加的文本。
// keep=false 表示这条是重复帧，直接丢掉（与前端 mergeText 用同一套识别规则）。
func (s *segmentState) advance(text string) (inc string, keep bool) {
	if text == "" {
		return "", false
	}
	if s.accum != "" && len(text) > len(s.accum) && strings.HasPrefix(text, s.accum) {
		inc, s.accum = text[len(s.accum):], text
		return inc, true
	}
	if strings.HasSuffix(s.accum, text) {
		return "", false
	}
	s.accum += text
	return text, true
}

// deltaCoalescer 增量帧合并器。可合并的文本增量进窗口，其余事件一律先落地再直发。
type deltaCoalescer struct {
	cfg  CoalesceConfig
	emit func(*types.Event)

	mu    sync.Mutex
	segs  map[coalesceKey]*segmentState
	order []coalesceKey
	// active 窗口定时器起来之后才真正合并：漏起窗口就把帧吞掉是不可接受的，宁可逐帧直发
	active bool
	closed bool

	stopCh   chan struct{}
	stopOnce sync.Once
}

// newDeltaCoalescer 创建合并器；emit 必须是「非阻塞」的出口（bridge.send → hub.Dispatch → 入队）
func newDeltaCoalescer(cfg CoalesceConfig, emit func(*types.Event)) *deltaCoalescer {
	return &deltaCoalescer{
		cfg:    cfg.withDefaults(),
		emit:   emit,
		segs:   make(map[coalesceKey]*segmentState),
		stopCh: make(chan struct{}),
	}
}

// Start 启动合并窗口，可重复调用（只生效一次）
func (c *deltaCoalescer) Start(ctx context.Context) {
	if c == nil || !c.cfg.Enabled || ctx == nil {
		return
	}
	c.mu.Lock()
	if c.active || c.closed {
		c.mu.Unlock()
		return
	}
	c.active = true
	c.mu.Unlock()

	go c.loop(ctx)
}

func (c *deltaCoalescer) loop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			// 到点：窗口里每个片段各自拼成一帧落地
			c.flushAll()
		}
	}
}

// Close 停窗口并把积压落地（尽力而为，此时连接可能已经断了），可重复调用
func (c *deltaCoalescer) Close() {
	if c == nil {
		return
	}
	c.stopOnce.Do(func() { close(c.stopCh) })

	c.mu.Lock()
	defer c.mu.Unlock()
	c.active = false
	c.closed = true
	c.flushAllLocked()
	c.segs = make(map[coalesceKey]*segmentState)
	c.order = nil
}

// Push 收一条事件：
//   - 可合并的文本增量进窗口，同片段拼接，攒够字节数立刻落地；
//   - 其余事件先把该会话窗口里的增量落地再原样发出，保证「增量在前、关键帧在后」。
func (c *deltaCoalescer) Push(event *types.Event) {
	if event == nil {
		return
	}

	kind, text, mergeable := mergeableDelta(event)
	useWindow := mergeable && c.cfg.Enabled

	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.active || !useWindow {
		c.flushSessionLocked(event.SessionID)
		c.dropCompletedLocked(event)
		c.emit(event)
		return
	}

	key := coalesceKey{event.SessionID, event.ReqID, event.EventID, kind}
	seg := c.segs[key]
	if seg == nil {
		if len(c.segs) >= c.cfg.MaxSegments {
			// 少见路径：只靠 completed 收尾没清干净（例如轮次被中断），先整体落地腾地方
			c.cfg.Logger.Debug("增量合并片段数达到上限，整体落地一次", "segments", len(c.segs))
			c.flushAllLocked()
			c.segs = make(map[coalesceKey]*segmentState)
			c.order = nil
		}
		seg = &segmentState{}
		c.segs[key] = seg
		c.order = append(c.order, key)
	}

	inc, keep := seg.advance(text)
	if !keep {
		return
	}
	seg.pending.WriteString(inc)
	seg.last = event
	if seg.pending.Len() >= c.cfg.MaxBytes {
		c.emitLocked(key, seg)
	}
}

// flushAll 落地所有片段的待发增量
func (c *deltaCoalescer) flushAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flushAllLocked()
}

func (c *deltaCoalescer) flushAllLocked() {
	for _, key := range c.order {
		if seg := c.segs[key]; seg != nil {
			c.emitLocked(key, seg)
		}
	}
}

// flushSessionLocked 只落地某个会话的待发增量（关键帧前的顺序保证）
func (c *deltaCoalescer) flushSessionLocked(sessionID string) {
	for _, key := range c.order {
		if key.sessionID != sessionID {
			continue
		}
		if seg := c.segs[key]; seg != nil {
			c.emitLocked(key, seg)
		}
	}
}

// emitLocked 把窗口里的增量作为一帧发出：内容仍是要追加的那一段，delta 语义不变
func (c *deltaCoalescer) emitLocked(key coalesceKey, seg *segmentState) {
	text := seg.pending.String()
	seg.pending.Reset()
	if text == "" || seg.last == nil {
		return
	}
	c.emit(mergedEvent(seg.last, key.kind, text))
}

// dropCompletedLocked 收尾帧之后该片段不会再有增量，顺手释掉 accum，
// 避免长会话下状态越攒越多
func (c *deltaCoalescer) dropCompletedLocked(event *types.Event) {
	var kind string
	switch event.Type {
	case types.EventMessageCompleted:
		kind = coalesceKindMessage
	case types.EventReasoningCompleted:
		kind = coalesceKindReasoning
	default:
		return
	}

	key := coalesceKey{event.SessionID, event.ReqID, event.EventID, kind}
	if _, ok := c.segs[key]; !ok {
		return
	}
	delete(c.segs, key)
	kept := c.order[:0]
	for _, item := range c.order {
		if item != key {
			kept = append(kept, item)
		}
	}
	c.order = kept
}

// mergeableDelta 判断这条事件是不是可合并的纯文本增量。
// 带工具、媒体、审批、extra 的 delta 帧不合并：convert 会把这些字段随帧下发，
// 拼进文本帧里就丢了。
func mergeableDelta(event *types.Event) (kind string, text string, ok bool) {
	if event == nil || event.Message == nil {
		return "", "", false
	}
	if !protocol.IsDelta(protocol.KindAgent, string(event.Type)) {
		return "", "", false
	}
	msg := event.Message
	if msg.Media != nil || len(msg.ToolCalls) > 0 || msg.ApprovalID != nil || len(msg.Extra) > 0 {
		return "", "", false
	}

	switch event.Type {
	case types.EventMessageDelta:
		if msg.Content == "" || msg.ReasoningContent != "" {
			return "", "", false
		}
		return coalesceKindMessage, msg.Content, true
	case types.EventReasoningDelta:
		if msg.ReasoningContent == "" || msg.Content != "" {
			return "", "", false
		}
		return coalesceKindReasoning, msg.ReasoningContent, true
	}
	return "", "", false
}

// mergedEvent 用最近一条事件的元数据造一条合并帧。
// 值拷贝事件与 Message：这两个指针在总线上被多个订阅者共享，就地改会互相污染。
func mergedEvent(latest *types.Event, kind, text string) *types.Event {
	ev := *latest
	msg := *latest.Message
	msg.IsDetla = true
	msg.Content = ""
	msg.ReasoningContent = ""
	if kind == coalesceKindReasoning {
		msg.ReasoningContent = text
	} else {
		msg.Content = text
	}
	ev.Message = &msg
	return &ev
}
