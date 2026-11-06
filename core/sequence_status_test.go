package core

import (
	"testing"

	"github.com/kinwyb/buibuiCodex/core/types"
)

func TestSessionSequenceStatusTracksRunningAndPending(t *testing.T) {
	seq := &sessionSequence{
		runningSessions: map[string]string{"ws_u1_c1u1": "req-running"},
		pendingMessages: map[string][]*types.State{
			"ws_u1_c1u1": {
				types.NewState(&types.InputMessage{Session: "ws_u1_c1", SenderID: "u1"}),
			},
		},
	}

	status := seq.SessionStatus("ws_u1_c1u1")
	if !status.Running || status.ReqID != "req-running" {
		t.Fatalf("运行状态不符合预期: %+v", status)
	}
	if status.Pending != 1 || !status.Busy() {
		t.Fatalf("排队状态不符合预期: %+v", status)
	}

	idle := seq.SessionStatus("ws_u1_c2u1")
	if idle.Running || idle.Pending != 0 || idle.Busy() {
		t.Fatalf("空闲会话状态不符合预期: %+v", idle)
	}
}

func TestManagerSessionStatusUsesSessionKey(t *testing.T) {
	seq := &sessionSequence{
		runningSessions: map[string]string{"ws_u1_c1u1": "req-running"},
		pendingMessages: make(map[string][]*types.State),
	}
	m := &Manager{sequence: seq}

	status := m.SessionStatus("ws_u1_c1u1")
	if !status.Running || status.ReqID != "req-running" {
		t.Fatalf("Manager 运行状态不符合预期: %+v", status)
	}
	if !m.IsSessionRunning("ws_u1_c1u1") {
		t.Fatal("Agent 执行中的会话应返回 running")
	}
	if m.IsSessionRunning("ws_u1_c1") {
		t.Fatal("查询必须使用完整 sessionKey，不能用 sessionID 代替")
	}
}
