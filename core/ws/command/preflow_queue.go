// 本文件是前置界面路由的执行队列。
//
// 为什么必须离开读循环：ui.response 只能从连接读循环进来，而 chat.send 正是读循环在处理的
// 消息，同步等待就是自己等自己，必然耗到超时。
//
// 为什么不是「每条消息开一个 goroutine」：那样只解了死锁，没解顺序——用户连发两条时，
// 后一条可能先送到 agent，界面也会被两个 worker 交叉操作。这里按会话排队：
// 一个会话最多一个 worker，消息严格先进先出，界面动作不会互相打断。

package command

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/preflow"
)

// maxPendingPerSession 单个会话最多排多少条待处理消息。
// 超出后新消息不再做界面判断，只排队照常投递：宁可少做一步前置，也不让聊天越点越慢。
const maxPendingPerSession = 32

// queuedJob 队列里的一条待投递消息
type queuedJob struct {
	msg *types.InputMessage
	// skipRoute 跳过界面判断直接投递（队列排满、或进程正在退出）
	skipRoute bool
}

// preflowQueue 按会话串行的「决策 + 投递」队列
type preflowQueue struct {
	ctx    context.Context
	cancel context.CancelFunc
	deps   Deps

	mu      sync.Mutex
	running map[string]bool
	pending map[string][]queuedJob
}

func newPreflowQueue(deps Deps) *preflowQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &preflowQueue{
		ctx:     ctx,
		cancel:  cancel,
		deps:    deps,
		running: map[string]bool{},
		pending: map[string][]queuedJob{},
	}
}

// submit 把消息排进所属会话的队列，立即返回，不占用连接读循环
func (q *preflowQueue) submit(msg *types.InputMessage) {
	if q == nil || msg == nil {
		return
	}
	key := msg.Session
	job := queuedJob{msg: msg}

	q.mu.Lock()
	if len(q.pending[key]) >= maxPendingPerSession {
		job.skipRoute = true
	}
	q.pending[key] = append(q.pending[key], job)
	startWorker := !q.running[key]
	if startWorker {
		q.running[key] = true
	}
	q.mu.Unlock()

	if startWorker {
		go q.drain(key)
	}
}

// Pending 各会话当前排队长度，测试与排查用
func (q *preflowQueue) Pending() map[string]int {
	if q == nil {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make(map[string]int, len(q.pending))
	for key, list := range q.pending {
		out[key] = len(list)
	}
	return out
}

// drain 串行处理一个会话的队列：判断并打开界面 -> 投递 -> 下一条
func (q *preflowQueue) drain(key string) {
	defer func() {
		q.mu.Lock()
		delete(q.running, key)
		q.mu.Unlock()
	}()

	for {
		q.mu.Lock()
		list := q.pending[key]
		if len(list) == 0 {
			delete(q.pending, key)
			q.mu.Unlock()
			return
		}
		job := list[0]
		q.pending[key] = list[1:]
		if q.ctx.Err() != nil {
			// 进程退出中：不再做界面动作，把已排队的消息投完就收
			job.skipRoute = true
		}
		q.mu.Unlock()

		if !job.skipRoute {
			q.route(job.msg)
		}
		q.publish(job.msg)
	}
}

// route 让决策模型替用户把页面打开、把入口点上。
// 任何一步不成功都只是什么都不做，不影响这条消息的投递。
func (q *preflowQueue) route(msg *types.InputMessage) {
	if q.deps.Preflow == nil {
		return
	}
	res := q.deps.Preflow.Route(q.ctx, &preflow.Request{
		ReqID:     msg.ID,
		SessionID: msg.Session,
		Content:   msg.Content,
	})
	if res != nil && res.Note != "" {
		msg.Content = strings.TrimSpace(msg.Content) + "\n" + res.Note
	}
	// 调试探针：交给 agent 的到底是哪段文字
	fmt.Println(msg.Content)
}

func (q *preflowQueue) publish(msg *types.InputMessage) {
	ctx := q.ctx
	if ctx.Err() != nil {
		// 已经应答过受理，退出时也别把消息悄悄吞掉
		ctx = context.Background()
	}
	if err := q.deps.Bus.PublishInput(ctx, msg); err != nil {
		slog.Error("ws 前置路由后投递消息失败", "session", msg.Session, "reqID", msg.ID, "error", err.Error())
	}
}

// Shutdown 停止所有 worker，服务关闭时调用
func (q *preflowQueue) Shutdown() {
	if q == nil {
		return
	}
	q.cancel()
}
