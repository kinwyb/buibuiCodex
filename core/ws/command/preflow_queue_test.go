// 本文件验证前置路由队列的两条底线：同一会话严格串行且保序，队列绝不阻塞投递本身。

package command

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/preflow"
)

// tracePreflow 记录调用顺序，并用 release 模拟「界面动作要等浏览器回执」的耗时：
// Route 一进入就在 entered 上打个招呼，然后卡在 release 上，直到测试 close(release)。
type tracePreflow struct {
	mu      sync.Mutex
	calls   []string
	inside  int
	peak    int
	entered chan struct{}
	release chan struct{}
}

func (f *tracePreflow) Route(_ context.Context, req *preflow.Request) *preflow.Result {
	f.mu.Lock()
	f.calls = append(f.calls, req.Content)
	f.inside++
	if f.inside > f.peak {
		f.peak = f.inside
	}
	f.mu.Unlock()

	if f.entered != nil {
		select {
		case f.entered <- struct{}{}:
		default:
		}
	}
	if f.release != nil {
		<-f.release
	}

	f.mu.Lock()
	f.inside--
	f.mu.Unlock()
	return &preflow.Result{}
}

func (f *tracePreflow) snapshot() (calls []string, peak int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...), f.peak
}

func newMsg(session, content string) *types.InputMessage {
	return &types.InputMessage{ID: uuid.NewString(), Session: session, Channel: "ws", Content: content}
}

func drainInputs(t *testing.T, sub *bus.InboundSubscription, want int) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var got []string
	for len(got) < want {
		in, ok := sub.Consume(ctx)
		if !ok {
			t.Fatalf("只收到 %d 条，期望 %d：%v", len(got), want, got)
		}
		got = append(got, in.Content)
	}
	return got
}

func TestQueueSerializesAndKeepsOrder(t *testing.T) {
	msgBus := bus.NewMessageBus(16)
	defer msgBus.Close()
	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}
	defer sub.Close()

	pf := &tracePreflow{entered: make(chan struct{}, 1), release: make(chan struct{})}
	q := newPreflowQueue(Deps{Bus: msgBus, Preflow: pf, Channel: "ws"})
	defer q.Shutdown()

	q.submit(newMsg("ws_web_c1", "第一条"))
	<-pf.entered // 第一条正在做界面动作，卡在等浏览器回执
	q.submit(newMsg("ws_web_c1", "第二条"))

	// 第二条排在队列里，既没投递也没进入界面判断：会话内不交叉
	time.Sleep(50 * time.Millisecond)
	if got := drainSoon(sub); got != nil {
		t.Fatalf("第二条不该抢跑: %+v", got)
	}
	if calls, _ := pf.snapshot(); len(calls) != 1 {
		t.Fatalf("会话内不应并发做界面判断: %v", calls)
	}

	close(pf.release)
	got := drainInputs(t, sub, 2)
	if strings.Join(got, ",") != "第一条,第二条" {
		t.Errorf("投递顺序应保序，实际 %v", got)
	}
	if _, peak := pf.snapshot(); peak != 1 {
		t.Errorf("同一会话最多只有一个界面判断在跑，峰值 %d", peak)
	}
}

func TestQueueSessionsAreIndependent(t *testing.T) {
	msgBus := bus.NewMessageBus(16)
	defer msgBus.Close()
	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}
	defer sub.Close()

	pf := &tracePreflow{}
	q := newPreflowQueue(Deps{Bus: msgBus, Preflow: pf, Channel: "ws"})
	defer q.Shutdown()

	q.submit(newMsg("ws_web_c1", "甲会话"))
	q.submit(newMsg("ws_web_c2", "乙会话"))
	got := drainInputs(t, sub, 2)
	if len(got) != 2 {
		t.Fatalf("两条都要投递: %v", got)
	}
	if calls, _ := pf.snapshot(); len(calls) != 2 {
		t.Errorf("两个会话都应做界面判断: %v", calls)
	}
}

// 队列积压到上限后，新消息不再做界面判断，但仍然照常投递（保序）
func TestQueueSkipsRouteWhenBacklogged(t *testing.T) {
	msgBus := bus.NewMessageBus(64)
	defer msgBus.Close()
	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}
	defer sub.Close()

	pf := &tracePreflow{entered: make(chan struct{}, 1), release: make(chan struct{})}
	q := newPreflowQueue(Deps{Bus: msgBus, Preflow: pf, Channel: "ws"})
	defer q.Shutdown()

	q.submit(newMsg("ws_web_c1", "堵塞中的第一条"))
	<-pf.entered
	total := maxPendingPerSession + 5
	for i := 1; i < total; i++ {
		q.submit(newMsg("ws_web_c1", fmt.Sprintf("第%d条", i+1)))
	}
	if q.Pending()["ws_web_c1"] != total-1 {
		t.Fatalf("排队长度 = %d, want %d", q.Pending()["ws_web_c1"], total-1)
	}

	close(pf.release)
	got := drainInputs(t, sub, total)
	if got[0] != "堵塞中的第一条" || got[len(got)-1] != fmt.Sprintf("第%d条", total) {
		t.Errorf("积压后仍要保序: %v … %v", got[:2], got[len(got)-2:])
	}
	calls, _ := pf.snapshot()
	if len(calls) >= total {
		t.Errorf("积压的消息应跳过界面判断，实际做了 %d 次 / 共 %d 条", len(calls), total)
	}
}

// 进程退出时不再做界面动作，但已排队的消息要投完
func TestQueueShutdownPublishesBacklog(t *testing.T) {
	msgBus := bus.NewMessageBus(16)
	defer msgBus.Close()
	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}
	defer sub.Close()

	pf := &tracePreflow{}
	q := newPreflowQueue(Deps{Bus: msgBus, Preflow: pf, Channel: "ws"})

	q.submit(newMsg("ws_web_c1", "退出前排队"))
	q.Shutdown()
	// 再投一条：此时 ctx 已取消，队列应跳过界面判断直接投递
	q.submit(newMsg("ws_web_c1", "退出后到达"))

	got := drainInputs(t, sub, 1)
	if got[0] != "退出前排队" {
		t.Errorf("已排队的消息必须投出去: %v", got)
	}
}

func TestQueueNilSafe(t *testing.T) {
	var q *preflowQueue
	q.submit(newMsg("s", "x"))
	q.Shutdown()
	if len(q.Pending()) != 0 {
		t.Error("空队列应有零个待处理项")
	}
}

// drainSoon 非阻塞地看一眼有没有消息到达
func drainSoon(sub *bus.InboundSubscription) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	var got []string
	for {
		in, ok := sub.Consume(ctx)
		if !ok || in == nil {
			return got
		}
		got = append(got, in.Content)
	}
}
