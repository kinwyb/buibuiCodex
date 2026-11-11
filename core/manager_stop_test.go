package core

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
)

type blockingManagerAgent struct {
	prompts atomic.Int32
	started chan struct{}
	release chan struct{}
}

func (a *blockingManagerAgent) AgentID() string { return "blocking" }
func (a *blockingManagerAgent) Init() error     { return nil }
func (a *blockingManagerAgent) Skills(context.Context) ([]types.Skill, error) {
	return nil, nil
}
func (a *blockingManagerAgent) Prompt(context.Context, *types.State) error {
	if a.prompts.Add(1) == 1 {
		close(a.started)
	}
	<-a.release
	return nil
}
func (a *blockingManagerAgent) Approve(any, jsonRpc.ApprovalDecision) error { return nil }
func (a *blockingManagerAgent) Steer(context.Context, *types.State) error   { return nil }
func (a *blockingManagerAgent) Cancel(context.Context, string) error        { return nil }
func (a *blockingManagerAgent) Stop() error {
	// 模拟真实 Agent.Stop：先等待活跃 Prompt 结束。
	<-a.release
	return nil
}

func TestManagerStopRejectsNewMessagesWhileDraining(t *testing.T) {
	msgBus := bus.NewMessageBus(10)
	manager := NewManager(msgBus, db.NewData(nil))
	agent := &blockingManagerAgent{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	manager.RegisterAgent("blocking", agent, true)
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("启动 Manager 失败: %v", err)
	}
	if err := msgBus.PublishInput(context.Background(), &types.InputMessage{
		ID: "first", Channel: "probe", AccountID: "a", SenderID: "u", ChatID: "c", Content: "hello",
	}); err != nil {
		t.Fatalf("发布首条消息失败: %v", err)
	}
	select {
	case <-agent.started:
	case <-time.After(time.Second):
		t.Fatal("首条消息未进入 Prompt")
	}

	stopDone := make(chan struct{})
	go func() {
		_ = manager.Stop()
		close(stopDone)
	}()
	time.Sleep(30 * time.Millisecond)

	if err := msgBus.PublishInput(context.Background(), &types.InputMessage{
		ID: "second", Channel: "probe", AccountID: "a", SenderID: "u", ChatID: "c", Content: "again",
	}); err != nil {
		t.Fatalf("总线仍可发布消息，但不应被处理: %v", err)
	}
	time.Sleep(80 * time.Millisecond)
	if got := agent.prompts.Load(); got != 1 {
		t.Fatalf("Stop 后不应再处理新消息，实际 Prompt 次数 %d", got)
	}
	select {
	case <-stopDone:
		t.Fatal("活跃 Prompt 未结束前 Manager.Stop 不应返回")
	default:
	}

	close(agent.release)
	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("释放活跃 Prompt 后 Manager.Stop 应返回")
	}
}
