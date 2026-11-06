package core

import (
	"context"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/types"
)

type uiEventSequenceHandler struct {
	steered bool
	busy    int
}

func (h *uiEventSequenceHandler) messageHandler(context.Context, *types.State) {}

func (h *uiEventSequenceHandler) busyNotification(context.Context, *types.InputMessage, bool) {
	h.busy++
}

func (h *uiEventSequenceHandler) log(context.Context, string, string, string) {}

func (h *uiEventSequenceHandler) agentSteer(context.Context, *types.State) {}

func (h *uiEventSequenceHandler) agentUIEventSteer(context.Context, *types.State) bool {
	return h.steered
}

func newUIEventState() *types.State {
	return types.NewState(&types.InputMessage{
		ID:        "req-ui",
		Session:   "ws_u1_c1",
		AccountID: "u1",
		SenderID:  "u1",
		Content:   `<ui_events>{"v":1,"events":[{"ts":1,"view":"V1","event":"selection.change"}]}</ui_events>`,
	})
}

func TestUIEventSteersRunningTurn(t *testing.T) {
	handler := &uiEventSequenceHandler{steered: true}
	seq := &sessionSequence{
		ctx:             context.Background(),
		handler:         handler,
		runningSessions: map[string]string{"ws_u1_c1": "req-1"},
		pendingMessages: make(map[string][]*types.State),
	}
	seq.handleInboundMessage(newUIEventState())
	if len(seq.pendingMessages) != 0 {
		t.Fatalf("steer 成功后不应排队: %+v", seq.pendingMessages)
	}
}

func TestUIEventSteerFailureQueuesWithoutBusyNotice(t *testing.T) {
	handler := &uiEventSequenceHandler{steered: false}
	seq := &sessionSequence{
		ctx:             context.Background(),
		handler:         handler,
		runningSessions: map[string]string{"ws_u1_c1": "req-1"},
		pendingMessages: make(map[string][]*types.State),
	}
	seq.handleInboundMessage(newUIEventState())
	if len(seq.pendingMessages["ws_u1_c1"]) != 1 {
		t.Fatalf("steer 失败后事件应排队: %+v", seq.pendingMessages)
	}
	if handler.busy != 0 {
		t.Fatalf("UI 事件排队不应显示请求繁忙，实际 %d 次", handler.busy)
	}
}
