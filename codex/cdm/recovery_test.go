package cdm

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/db"
)

func seedRecoveryTurn(t *testing.T, store db.ISession) db.SessionTurn {
	t.Helper()
	ctx := context.Background()
	if err := store.SessionSave(ctx, &db.Session{SessionID: "s1", Channel: "ws", ChatID: "chat"}); err != nil {
		t.Fatalf("保存 session 失败: %v", err)
	}
	if err := store.ThreadSave(ctx, &db.SessionThread{ThreadID: "t1", SessionID: "s1", Agent: "codex"}); err != nil {
		t.Fatalf("保存 thread 失败: %v", err)
	}
	if err := store.RequestSave(ctx, "s1", "r1", "原始问题", "codex"); err != nil {
		t.Fatalf("保存请求失败: %v", err)
	}
	row := db.SessionTurn{
		ReqID:          "r1",
		TurnID:         "logical-1",
		RunTurnID:      "run-1",
		ThreadID:       "t1",
		SessionID:      "s1",
		Agent:          "codex",
		RecoveryStatus: db.RecoveryStatusRunning,
	}
	if err := store.TurnSave(ctx, &row); err != nil {
		t.Fatalf("保存 turn 失败: %v", err)
	}
	return row
}

func TestTurnRecoveryFinalizesCompletedRunWithoutContinuation(t *testing.T) {
	store := newReconcileTestStore(t)
	row := seedRecoveryTurn(t, store)
	agent := NewAgent(nil, 0)
	agent.cfg = nil
	r := newTurnRecovery(agent, store)
	r.resumeThread = func(context.Context, string) (*codex.Thread, *jsonRpc.Thread, error) {
		return nil, &jsonRpc.Thread{
			ID: "t1",
			Turns: []jsonRpc.Turn{{
				TurnID: "run-1",
				Status: "completed",
				Items: []jsonRpc.ThreadItem{
					json.RawMessage(`{"type":"agentMessage","text":"已完成回答"}`),
				},
			}},
		}, nil
	}
	r.reconcileRun = func(context.Context, string, string, string) (bool, error) { return true, nil }

	r.recoverOne(row)

	got := store.TurnQueryByID(context.Background(), "logical-1")
	if got == nil || !got.IsCompleted || got.Answer != "已完成回答" || got.RecoveryStatus != db.RecoveryStatusCompleted {
		t.Fatalf("已完成 run 应直接收敛，不应继续执行: %+v", got)
	}
}

func TestTurnRecoveryMarksFailedRunWithoutContinuation(t *testing.T) {
	store := newReconcileTestStore(t)
	row := seedRecoveryTurn(t, store)
	agent := NewAgent(nil, 0)
	r := newTurnRecovery(agent, store)
	r.resumeThread = func(context.Context, string) (*codex.Thread, *jsonRpc.Thread, error) {
		return nil, &jsonRpc.Thread{
			ID: "t1",
			Turns: []jsonRpc.Turn{{
				TurnID: "run-1",
				Status: "failed",
				Error:  &jsonRpc.CodexError{Message: "模型失败"},
			}},
		}, nil
	}
	r.reconcileRun = func(context.Context, string, string, string) (bool, error) { return true, nil }

	r.recoverOne(row)

	got := store.TurnQueryByID(context.Background(), "logical-1")
	if got == nil {
		t.Fatal("应保留原逻辑 turn")
	}
	if got.RecoveryStatus != db.RecoveryStatusFailed || got.RecoveryError != "模型失败" {
		t.Fatalf("失败 run 不应自动重跑: %+v", got)
	}
}

func TestTurnRecoveryStartsContinuationForInterruptedRun(t *testing.T) {
	store := newReconcileTestStore(t)
	row := seedRecoveryTurn(t, store)
	agent := NewAgent(nil, 0)
	r := newTurnRecovery(agent, store)
	r.resumeThread = func(context.Context, string) (*codex.Thread, *jsonRpc.Thread, error) {
		return nil, &jsonRpc.Thread{
			ID: "t1",
			Turns: []jsonRpc.Turn{{
				TurnID: "run-1",
				Status: "interrupted",
			}},
		}, nil
	}
	reconciled := false
	r.reconcileRun = func(_ context.Context, threadID, logicalTurnID, runTurnID string) (bool, error) {
		reconciled = true
		if threadID != "t1" || logicalTurnID != "logical-1" || runTurnID != "run-1" {
			t.Fatalf("旧 run 补全参数不符: %s %s %s", threadID, logicalTurnID, runTurnID)
		}
		return true, nil
	}
	continued := false
	r.continueTurn = func(got db.SessionTurn, _ *codex.Thread, attempt int) {
		continued = true
		if got.TurnID != "logical-1" || attempt != 1 {
			t.Fatalf("恢复 continuation 参数不符: %+v attempt=%d", got, attempt)
		}
	}

	r.recoverOne(row)

	if !reconciled || !continued {
		t.Fatalf("interrupted run 应先补全旧事件再启动 continuation: reconciled=%v continued=%v", reconciled, continued)
	}
}

func TestTurnRecoverySessionWaitIsReleased(t *testing.T) {
	r := newTurnRecovery(&Agent{}, nil)
	ch := r.markSession("s1")
	waitDone := make(chan struct{})
	go func() {
		_ = r.WaitSession(context.Background(), "s1")
		close(waitDone)
	}()
	select {
	case <-waitDone:
		t.Fatal("恢复未完成时不应放行")
	case <-time.After(20 * time.Millisecond):
	}
	r.clearSession("s1", ch)
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("恢复完成后应放行")
	}
}
