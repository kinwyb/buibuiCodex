package bridge

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// collector 收集落地顺序，便于断言「增量在前、关键帧在后」
type collector struct {
	mu     sync.Mutex
	events []*types.Event
}

func (c *collector) emit(event *types.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
}

func (c *collector) snapshot() []*types.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*types.Event(nil), c.events...)
}

func deltaEvent(eventID, text string) *types.Event {
	return &types.Event{
		SessionID: "s1",
		ReqID:     "r1",
		EventID:   eventID,
		Type:      types.EventMessageDelta,
		Channel:   "ws",
		Message:   &types.Message{Role: types.RoleAssistant, Content: text, IsDetla: true},
	}
}

func startedCoalescer(cfg CoalesceConfig, emit func(*types.Event)) (*deltaCoalescer, context.CancelFunc) {
	cfg.Enabled = true
	if cfg.Interval <= 0 {
		cfg.Interval = time.Hour // 测试里不靠定时器驱动，显式 flush
	}
	c := newDeltaCoalescer(cfg, emit)
	ctx, cancel := context.WithCancel(context.Background())
	c.Start(ctx)
	return c, func() { c.Close(); cancel() }
}

func TestCoalesceMergesWindowIntoOneFrame(t *testing.T) {
	c := &collector{}
	co, stop := startedCoalescer(CoalesceConfig{}, c.emit)
	defer stop()

	for _, text := range []string{"你", "好", "啊"} {
		co.Push(deltaEvent("item-1", text))
	}
	if got := c.snapshot(); len(got) != 0 {
		t.Fatalf("窗口内不应落地，实际 %d 帧", len(got))
	}
	co.flushAll()

	got := c.snapshot()
	if len(got) != 1 {
		t.Fatalf("三帧增量应合并成一帧，实际 %d 帧", len(got))
	}
	if got[0].Message.Content != "你好啊" {
		t.Fatalf("合并内容不符: %q", got[0].Message.Content)
	}
	if !got[0].Message.IsDetla {
		t.Fatal("合并帧必须仍是增量：改成全文会让带宽变成 O(n^2)")
	}
}

// 关键帧之前必须先落地窗口里的增量，否则前端会先收到 completed 再收到旧增量
func TestCoalesceFlushesBeforeBoundaryFrame(t *testing.T) {
	c := &collector{}
	co, stop := startedCoalescer(CoalesceConfig{}, c.emit)
	defer stop()

	co.Push(deltaEvent("item-1", "生产"))
	co.Push(&types.Event{
		SessionID: "s1", ReqID: "r1", EventID: "item-1",
		Type:    types.EventMessageCompleted,
		Channel: "ws",
		Message: &types.Message{Role: types.RoleAssistant, Content: "生产进度如何"},
	})
	co.flushAll()

	got := c.snapshot()
	if len(got) != 2 {
		t.Fatalf("应为「合并帧 + completed 帧」两帧，实际 %d", len(got))
	}
	if got[0].Type != types.EventMessageDelta || got[1].Type != types.EventMessageCompleted {
		t.Fatalf("顺序不符: %s -> %s", got[0].Type, got[1].Type)
	}
	if len(co.segs) != 0 {
		t.Fatalf("completed 后应释掉片段状态，实际残留 %d 个", len(co.segs))
	}
}

// 一轮里思考与正文的多个 eventID 会交错，绝不能按 reqID 混成一个片段
func TestCoalesceKeepsSegmentsSeparate(t *testing.T) {
	c := &collector{}
	co, stop := startedCoalescer(CoalesceConfig{}, c.emit)
	defer stop()

	co.Push(deltaEvent("msg-1", "答案"))
	co.Push(&types.Event{
		SessionID: "s1", ReqID: "r1", EventID: "think-1",
		Type:    types.EventReasoningDelta,
		Channel: "ws",
		Message: &types.Message{Role: types.RoleAssistant, ReasoningContent: "先看订单", IsDetla: true},
	})
	co.Push(deltaEvent("msg-1", "A"))
	co.flushAll()

	got := c.snapshot()
	if len(got) != 2 {
		t.Fatalf("两个片段应各出一帧，实际 %d", len(got))
	}
	if got[0].Message.Content != "答案A" || got[0].Message.ReasoningContent != "" {
		t.Fatalf("正文片段不符: %+v", got[0].Message)
	}
	if got[1].Message.ReasoningContent != "先看订单" || got[1].Message.Content != "" {
		t.Fatalf("思考片段不符: %+v", got[1].Message)
	}
}

// 上游偶尔把累积全文当增量发（前端 mergeText 也有同样的兜底识别）
func TestCoalesceSplitsCumulativeFrame(t *testing.T) {
	c := &collector{}
	co, stop := startedCoalescer(CoalesceConfig{}, c.emit)
	defer stop()

	co.Push(deltaEvent("item-1", "生产"))
	co.Push(deltaEvent("item-1", "生产进度"))
	co.Push(deltaEvent("item-1", "进度")) // 重复帧：已经在尾部，丢掉
	co.flushAll()

	got := c.snapshot()
	if len(got) != 1 {
		t.Fatalf("应只落地一帧，实际 %d", len(got))
	}
	if got[0].Message.Content != "生产进度" {
		t.Fatalf("累积全文要切成真正的增量，实际 %q", got[0].Message.Content)
	}
}

func TestCoalesceMaxBytesFlushesImmediately(t *testing.T) {
	c := &collector{}
	co, stop := startedCoalescer(CoalesceConfig{MaxBytes: 4}, c.emit)
	defer stop()

	co.Push(deltaEvent("item-1", "abcd"))
	got := c.snapshot()
	if len(got) != 1 {
		t.Fatalf("攒够字节数应不等窗口立刻落地，实际 %d 帧", len(got))
	}
	if got[0].Message.Content != "abcd" {
		t.Fatalf("内容不符: %q", got[0].Message.Content)
	}
}

// 带工具/媒体的 delta 帧不参与合并：convert 会随帧下发这些字段
func TestCoalescePassesThroughNonTextDelta(t *testing.T) {
	c := &collector{}
	co, stop := startedCoalescer(CoalesceConfig{}, c.emit)
	defer stop()

	co.Push(deltaEvent("item-1", "先"))
	co.Push(&types.Event{
		SessionID: "s1", ReqID: "r1", EventID: "item-1",
		Type:    types.EventMessageDelta,
		Channel: "ws",
		Message: &types.Message{Role: types.RoleAssistant, Content: "后", Media: []types.Media{{Type: "image"}}},
	})
	co.flushAll()

	got := c.snapshot()
	if len(got) != 2 {
		t.Fatalf("媒体帧应直发且先落地积压，实际 %d 帧", len(got))
	}
	if got[0].Message.Content != "先" || got[0].Message.Media != nil {
		t.Fatalf("先落地的应是窗口里的纯文本增量: %+v", got[0].Message)
	}
	if got[1].Message.Media == nil {
		t.Fatal("媒体帧应原样发出")
	}
}

func TestCoalesceInactivePassesThrough(t *testing.T) {
	c := &collector{}
	co := newDeltaCoalescer(CoalesceConfig{Enabled: true}, c.emit) // 没 Start
	for _, text := range []string{"a", "b"} {
		co.Push(deltaEvent("item-1", text))
	}
	if got := c.snapshot(); len(got) != 2 {
		t.Fatalf("窗口未起来时应逐帧直发，实际 %d 帧", len(got))
	}
}

func TestCoalesceDisabledPassesThrough(t *testing.T) {
	c := &collector{}
	co := newDeltaCoalescer(CoalesceConfig{Enabled: false}, c.emit)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	co.Start(ctx)
	for _, text := range []string{"a", "b"} {
		co.Push(deltaEvent("item-1", text))
	}
	if got := c.snapshot(); len(got) != 2 {
		t.Fatalf("关掉合并后应逐帧直发，实际 %d 帧", len(got))
	}
}

func TestCoalesceTickFlushEmitsOncePerWindow(t *testing.T) {
	c := &collector{}
	co, stop := startedCoalescer(CoalesceConfig{Interval: 10 * time.Millisecond}, c.emit)
	defer stop()

	co.Push(deltaEvent("item-1", "生产"))
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && len(c.snapshot()) == 0 {
		time.Sleep(2 * time.Millisecond)
	}
	if len(c.snapshot()) != 1 {
		t.Fatalf("窗口到点应落地一帧，实际 %d", len(c.snapshot()))
	}
	time.Sleep(60 * time.Millisecond)
	if got := len(c.snapshot()); got != 1 {
		t.Fatalf("空窗口不该重复发同一帧，实际 %d 帧", got)
	}
}

func TestCoalesceCloseFlushesPending(t *testing.T) {
	c := &collector{}
	co := newDeltaCoalescer(CoalesceConfig{Enabled: true, Interval: time.Hour}, c.emit)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	co.Start(ctx)
	co.Push(deltaEvent("item-1", "别丢"))
	co.Close()
	if got := c.snapshot(); len(got) != 1 || got[0].Message.Content != "别丢" {
		t.Fatalf("Close 要尽力落地积压，实际 %+v", got)
	}
}

// 合并不得改写上游事件：总线把同一个指针发给所有订阅者（企业微信、日志、落库）
func TestCoalesceDoesNotMutateSourceEvents(t *testing.T) {
	c := &collector{}
	co, stop := startedCoalescer(CoalesceConfig{}, c.emit)
	defer stop()

	first, second := deltaEvent("item-1", "生产"), deltaEvent("item-1", "进度")
	co.Push(first)
	co.Push(second)
	co.flushAll()

	if first.Message.Content != "生产" || second.Message.Content != "进度" {
		t.Fatalf("原事件被改了: %q / %q", first.Message.Content, second.Message.Content)
	}
	if first.Message == c.snapshot()[0].Message {
		t.Fatal("合并帧必须自带 Message，不能复用共享指针")
	}
}

// 端到端接线：forward 走合并，落地的信封仍是 delta=true 的增量帧
type fakeDispatcher struct {
	mu   sync.Mutex
	envs []*protocol.Envelope
}

func (d *fakeDispatcher) Dispatch(_ string, env *protocol.Envelope) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.envs = append(d.envs, env)
	return 1
}

func TestForwardUsesCoalescer(t *testing.T) {
	d := &fakeDispatcher{}
	b := New(Options{Channel: "ws", Dispatcher: d, Coalesce: CoalesceConfig{Enabled: true, Interval: time.Hour}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.coalescer.Start(ctx)
	defer b.coalescer.Close()

	b.forward(deltaEvent("item-1", "生产"))
	b.forward(deltaEvent("item-1", "进度"))
	b.coalescer.flushAll()

	if len(d.envs) != 1 {
		t.Fatalf("两帧增量应合并成一帧信封，实际 %d", len(d.envs))
	}
	var data protocol.AgentData
	if err := d.envs[0].DecodeData(&data); err != nil {
		t.Fatalf("解析载荷失败: %v", err)
	}
	if data.Content != "生产进度" || !data.Delta {
		t.Fatalf("合并帧语义不符: content=%q delta=%v", data.Content, data.Delta)
	}
}
