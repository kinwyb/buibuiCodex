package core

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/types"
)

type steerCommandHandler struct {
	steered      chan *types.State
	messageCalls atomic.Int32
	busyCalls    atomic.Int32
}

func (h *steerCommandHandler) messageHandler(context.Context, *types.State) {
	h.messageCalls.Add(1)
}

func (h *steerCommandHandler) busyNotification(context.Context, *types.InputMessage, bool) {
	h.busyCalls.Add(1)
}

func (h *steerCommandHandler) agentSteer(_ context.Context, state *types.State) {
	h.steered <- state
}

func (h *steerCommandHandler) agentUIEventSteer(context.Context, *types.State) bool {
	return false
}

func (h *steerCommandHandler) log(context.Context, string, string, string) {}

func TestSteerCommandDoesNotDispatchAsNewTurn(t *testing.T) {
	ctx := t.Context()
	msgBus := bus.NewMessageBus(8)
	defer msgBus.Close()

	sub, err := msgBus.SubscribeInput()
	if err != nil {
		t.Fatal(err)
	}
	seq := newSessionSequence(ctx, sub)
	handler := &steerCommandHandler{steered: make(chan *types.State, 1)}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = seq.Start(handler)
	}()

	msg := &types.InputMessage{
		ID:        "req-steer",
		Session:   "ws_u1_c1",
		AccountID: "u1",
		SenderID:  "u1",
		ChatID:    "c1",
		Content:   "/steer 补充要求",
	}
	if err := msgBus.PublishInput(ctx, msg); err != nil {
		t.Fatal(err)
	}

	state := <-handler.steered
	if state.Input.Content != "补充要求" {
		t.Fatalf("steer 内容应去掉命令前缀，实际 %q", state.Input.Content)
	}

	seq.cancel()
	<-done
	seq.wg.Wait()
	if got := handler.messageCalls.Load(); got != 0 {
		t.Fatalf("steer 后不应再作为新一轮投递，实际调用 %d 次", got)
	}
	if got := handler.busyCalls.Load(); got != 0 {
		t.Fatalf("steer 后不应显示排队提示，实际调用 %d 次", got)
	}
}
