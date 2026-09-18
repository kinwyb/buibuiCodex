package core

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kinwyb/buibuiCodex/codex/cdm"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/types"
)

type sessionHandler interface {
	messageHandler(ctx context.Context, state *types.State)
	busyNotification(ctx context.Context, msg *types.InputMessage, isStore bool)
	agentSteer(ctx context.Context, state *types.State)
	log(ctx context.Context, level, source, message string)
}

// MaxConcurrentSessions 最大并发 session 处理数
const MaxConcurrentSessions = 5

// maxInflightGoroutines spawn 出来的 RouteInbound goroutine 总上限，防止瞬时洪流尖刺
const maxInflightGoroutines = 1024

// pending 队列上限：单 session 32、全局 1024，超过即拒收并复用 busy 提示
const (
	maxPendingPerSession = 32
	maxPendingTotal      = 1024
)

// sessionSequence 消息处理序列
type sessionSequence struct {
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	inBoundSub      *bus.InboundSubscription //输入消息订阅
	runningMu       sync.RWMutex
	runningSessions map[string]string // sessionKey -> reqID
	// 并发控制
	semaphore       chan struct{}             // 限制最大并发 session 数（控制 agent.Prompt 实际执行）
	spawnLimit      chan struct{}             // 限制 RouteInbound goroutine 总数，避免洪流尖刺
	pendingMessages map[string][]*types.State // sessionKey -> 等待消息队列
	pendingTotal    int                       // pendingMessages 总数（受 pendingMu 保护）
	pendingMu       sync.Mutex
	handler         sessionHandler //session消息处理器
}

func (s *sessionSequence) messageHandler(ctx context.Context, state *types.State) {
	slog.Debug("[sessionSequence] message handler")
}

func (s *sessionSequence) busyNotification(ctx context.Context, msg *types.InputMessage, isStore bool) {
	slog.Debug("[sessionSequence] message busy notification")
}

func (s *sessionSequence) log(ctx context.Context, level, source, message string) {
	slog.Debug("[sessionSequence] message log")
}

func (s *sessionSequence) agentSteer(ctx context.Context, state *types.State) {
	slog.Debug("[sessionSequence] message agent steer")
}

func newSessionSequence(ctx context.Context, inputSub *bus.InboundSubscription) *sessionSequence {
	nctx, cancel := context.WithCancel(ctx)
	return &sessionSequence{
		ctx:             nctx,
		cancel:          cancel,
		runningSessions: make(map[string]string),
		inBoundSub:      inputSub,
		semaphore:       make(chan struct{}, MaxConcurrentSessions),
		spawnLimit:      make(chan struct{}, maxInflightGoroutines),
		pendingMessages: make(map[string][]*types.State),
	}
}

func (s *sessionSequence) Start(handler sessionHandler) error {
	if handler != nil {
		s.handler = handler
	}
	if s.handler == nil {
		s.handler = s
	}
	// 启动 pending 消息调度器,处理缓存中的消息
	// 因为常规处理方式如果处理结束，但缓冲中还有待处理消息会导致一直积压直到下一次消息来临
	// 该进程用于触发未被激发的待处理消息
	s.wg.Add(1)
	go s.processPendingScheduler()

	defer func() {
		if r := recover(); r != nil {
			slog.Error("manager processMessages recovered from panic", "panic", r)
			go s.Start(nil)
		}
	}()
	for {
		msg, ok := s.inBoundSub.Consume(s.ctx)
		if !ok {
			return nil
		}
		if msg.Content != "" {
			cmd := cdm.CommandParse(msg.Content)
			if cmd != nil && cmd.Steer {
				msg.Content = cmd.Msg
				state := types.NewState(msg)
				s.handler.agentSteer(s.ctx, state)
			}
		}
		select {
		case s.spawnLimit <- struct{}{}:
		default:
			slog.Warn("inflight goroutines saturated, dropping message", "msgID", msg.ID, "limit", maxInflightGoroutines)
			s.handler.busyNotification(s.ctx, msg, false)
			continue
		}
		s.wg.Add(1)
		go func(inMsg *types.InputMessage) {
			defer s.wg.Done()
			defer func() { <-s.spawnLimit }()
			state := types.NewState(msg)
			s.handleInboundMessage(state)
		}(msg)
	}
}

// Stop 停止所有 Agent
func (s *sessionSequence) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.inBoundSub != nil {
		s.inBoundSub.Close()
		s.inBoundSub = nil
	}

	// 清理 pendingMessages（cancel 后不再处理新消息）
	s.pendingMu.Lock()
	s.pendingMessages = make(map[string][]*types.State)
	s.pendingTotal = 0
	s.pendingMu.Unlock()

	// 等待所有 goroutine 退出（handleInboundMessage、processPendingScheduler）
	s.wg.Wait()
	return nil
}

func (s *sessionSequence) handleInboundMessage(state *types.State) {
	msg := state.Input
	sessionKey := state.SessionID + msg.SenderID

	// 首次：检查 session 是否正在运行 + 获取 semaphore
	s.runningMu.Lock()
	if _, running := s.runningSessions[sessionKey]; running {
		s.runningMu.Unlock()
		inSq := s.enqueuePending(sessionKey, state)
		if !inSq {
			slog.Warn("pending queue full, dropping message", "sessionKey", sessionKey, "msgID", msg.ID, "reason", "session running")
		}
		s.handler.busyNotification(s.ctx, msg, inSq)
		return
	}
	select {
	case s.semaphore <- struct{}{}:
		s.runningSessions[sessionKey] = state.ReqID
		s.runningMu.Unlock()
	default:
		s.runningMu.Unlock()
		inSq := s.enqueuePending(sessionKey, state)
		if !inSq {
			slog.Warn("pending queue full, dropping message", "sessionKey", sessionKey, "msgID", msg.ID, "reason", "semaphore saturated")
		}
		s.handler.busyNotification(s.ctx, msg, inSq)
		return
	}

	// 循环处理消息：处理完后尝试复用当前 goroutine 处理不同 session 的 pending 消息
	for {
		s.handler.messageHandler(s.ctx, state)

		// 释放当前 session，尝试获取下一条不同 session 的 pending
		s.runningMu.Lock()
		delete(s.runningSessions, sessionKey)
		s.runningMu.Unlock()

		// 尝试获取下一个不同 session 的 pending 消息（跳过 runningSessions 中已有的）
		next, nextSk := s.popNextPending(sessionKey)
		if next == nil {
			// 没有可复用的 pending 消息，释放 semaphore
			<-s.semaphore
			return
		}

		// 找到可复用的 pending，标记为运行中，复用当前 goroutine
		s.runningMu.Lock()
		s.runningSessions[nextSk] = next.ReqID
		s.runningMu.Unlock()

		state = next
		sessionKey = nextSk
	}
}

// enqueuePending 将消息加入等待队列。返回 false 表示队列已满（单 session 32 / 全局 1024）。
func (s *sessionSequence) enqueuePending(sessionKey string, state *types.State) bool {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if s.pendingTotal >= maxPendingTotal {
		return false
	}
	if len(s.pendingMessages[sessionKey]) >= maxPendingPerSession {
		return false
	}
	s.pendingMessages[sessionKey] = append(s.pendingMessages[sessionKey], state)
	s.pendingTotal++
	return true
}

// popNextPending 从 pendingMessages 中取出一条可处理的 pending 消息
// excludeSession: 当前正在处理的 session key，必须跳过
// 同时跳过 runningSessions 中已有的 session（避免徒劳触发"请稍候"提示）
func (s *sessionSequence) popNextPending(excludeSession string) (*types.State, string) {
	// 先快照 runningSessions，避免与 enqueuePending 等函数的锁顺序冲突导致死锁
	// (enqueuePending: runningMu → pendingMu; popNextPending: pendingMu → runningMu = 死锁)
	s.runningMu.RLock()
	runningCopy := make(map[string]struct{}, len(s.runningSessions))
	for k := range s.runningSessions {
		runningCopy[k] = struct{}{}
	}
	s.runningMu.RUnlock()

	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	for sk, pending := range s.pendingMessages {
		if sk == excludeSession {
			continue
		}
		if _, running := runningCopy[sk]; running {
			continue
		}
		if len(pending) == 0 {
			continue
		}
		next := pending[0]
		if len(pending) == 1 {
			delete(s.pendingMessages, sk)
		} else {
			s.pendingMessages[sk] = pending[1:]
		}
		s.pendingTotal--
		return next, sk
	}
	return nil, ""
}

// processPendingScheduler 独立调度器，负责从 pendingMessages 中按 session 轮询取消息
// 使用 ticker 定期调度，确保不同 session 的消息公平获取 semaphore
func (s *sessionSequence) processPendingScheduler() {
	defer s.wg.Done()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processOnePending()
		}
	}
}

// processOnePending 从 pendingMessages 中按 session 轮询取一条消息处理
// 只负责取消息并启动 goroutine，由 goroutine 走完整 handleInboundMessage 流程
// （semaphore 获取和 runningSessions 设置都在 handleInboundMessage 中统一处理）
func (s *sessionSequence) processOnePending() {
	next, _ := s.popNextPending("")
	if next == nil {
		return
	}

	// pending 消息已经入过队列，spawnLimit 满了也必须放它跑（否则消息丢失）；
	// 只在常规入口（processMessages）做 spawnLimit 拒收。
	s.spawnLimit <- struct{}{}
	s.wg.Go(func() {
		defer func() { <-s.spawnLimit }()
		s.handleInboundMessage(next)
	})
}
