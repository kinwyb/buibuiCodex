package channel

import (
	"context"
	"log/slog"
	"sync"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/types"
)

// StreamSender 实际发送回调
type StreamSender func(ctx context.Context, msg *types.Event) error

type StreamDoneFunc func(reqID string)

// StreamBuffer 流式消息缓冲投递（只处理流式消息）
// 每个 ReplyTo 独立 buffer + worker，多路并发互不干扰。
// Push 非阻塞（只写缓存），实际 API 调用由 per-ReplyTo worker 异步完成。
type StreamBuffer struct {
	mu            sync.Mutex
	buffers       map[string]*replyBuffer // key = msg.ReplyTo
	sendFunc      StreamSender
	doneFunc      StreamDoneFunc
	ctx           context.Context
	cancel        context.CancelFunc
	streamingMode bus.StreamingMode
}

// replyBuffer 单个 ReplyTo 的缓冲
type replyBuffer struct {
	key            string // ReplyTo 值（cleanup 用）
	latest         *types.Event
	isFinal        bool
	lastSent       int           // 已发送的最大 ChunkIndex（防止乱序）
	isAccumulateOn bool          // true: 累积模式, false: 增量模式（需自己累加）
	reasoning      string        //累计的思考模式
	content        string        //累计的回复内容
	wake           chan struct{} // buffered 1
}

// NewStreamBuffer 创建 StreamBuffer
func NewStreamBuffer(streamingMode bus.StreamingMode, sendFunc StreamSender) *StreamBuffer {
	return NewStreamBufferWithDone(streamingMode, sendFunc, nil)
}

func NewStreamBufferWithDone(streamingMode bus.StreamingMode, sendFunc StreamSender, doneFunc StreamDoneFunc) *StreamBuffer {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamBuffer{
		buffers:       make(map[string]*replyBuffer),
		sendFunc:      sendFunc,
		doneFunc:      doneFunc,
		ctx:           ctx,
		cancel:        cancel,
		streamingMode: streamingMode,
	}
}

// Push 非阻塞写入缓冲（仅流式消息调用）
func (b *StreamBuffer) Push(event *types.Event) {
	// req_id 为空时无法分组，直接回调发送
	key := event.ReqID
	if key == "" {
		b.sendFunc(b.ctx, event)
		return
	}
	msg := event.Message
	b.mu.Lock()
	rb, ok := b.buffers[key]
	if !ok {
		if !msg.IsDetla { //如果消息通道不存在，并且是最终消息，不需要再开通消息通道，直接发送
			b.sendFunc(b.ctx, event)
			b.mu.Unlock()
			return
		}
		// 首条消息：从 metadata 获取流式模式
		isAccumulate := b.streamingMode == bus.StreamingModeAccumulate
		rb = &replyBuffer{
			key:            key,
			latest:         event,
			isFinal:        false,
			lastSent:       -1,
			isAccumulateOn: isAccumulate,
			wake:           make(chan struct{}, 1),
		}
		b.buffers[key] = rb
		b.mu.Unlock()
		go b.work(rb)
		return
	}

	// 跳过比已发送更旧的 chunk（保证顺序）
	if event.ChunkIndex <= rb.lastSent {
		b.mu.Unlock()
		return
	}

	if rb.isAccumulateOn && msg.IsDetla {
		rb.reasoning = rb.reasoning + msg.ReasoningContent
		msg.ReasoningContent = rb.reasoning
		rb.content = rb.content + msg.Content
		msg.Content = rb.content
	}
	if !msg.IsDetla && msg.Content == "" { //单独思考结束不作为消息发送，合并等有具体回复消息才允许完成
		msg.IsDetla = true
	}
	rb.latest = event
	rb.isFinal = !msg.IsDetla
	if rb.isFinal { //如果是完成，那么清空累计内容
		rb.reasoning = ""
		rb.content = ""
	}
	b.mu.Unlock()
	if !msg.IsDetla { //如果是完整消息，必须处理完才能继续,避免多次回复消息被覆盖
		rb.wake <- struct{}{}
		return
	}
	// 非阻塞唤醒 worker
	select {
	case rb.wake <- struct{}{}:
	default:
	}
}

// Stop 停止所有 worker（channel 关闭时调用）
func (b *StreamBuffer) Stop() {
	b.cancel()
}

// work per-ReplyTo worker goroutine
func (b *StreamBuffer) work(rb *replyBuffer) {
	defer func() {
		b.mu.Lock()
		delete(b.buffers, rb.key)
		b.mu.Unlock()
		if b.doneFunc != nil {
			b.doneFunc(rb.key)
		}
		slog.Info("streamBuffer work end", "reqID", rb.key)
	}()

	for {
		b.mu.Lock()
		event := rb.latest
		rb.latest = nil
		isFinal := rb.isFinal
		b.mu.Unlock()
		if event == nil {
			slog.Info("streamBuffer work end", "reqID", rb.key)
		}
		if event != nil && event.Message != nil {
			msg := event.Message
			// msg 的 Content 已包含累积完整内容（Push 中已处理）
			if msg.Content != "" || msg.ReasoningContent != "" { //有消息内容才发送
				b.sendFunc(b.ctx, event)
			}

			b.mu.Lock()
			if event.ChunkIndex > rb.lastSent {
				rb.lastSent = event.ChunkIndex
			}
			b.mu.Unlock()
		}

		if isFinal {
			b.mu.Lock()
			hasPending := rb.latest != nil
			b.mu.Unlock()
			if !hasPending {
				return
			}
			continue
		}

		// 无数据：阻塞等 wake 或 ctx 取消
		select {
		case <-rb.wake:
		case <-b.ctx.Done():
			return
		}
	}
}
