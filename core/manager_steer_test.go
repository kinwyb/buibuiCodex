package core

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
)

type steerManagerAgent struct {
	steerErr    error
	cancelErr   error
	steerCalls  int
	cancelCalls int
}

func (a *steerManagerAgent) AgentID() string { return "steer" }
func (a *steerManagerAgent) Init() error     { return nil }
func (a *steerManagerAgent) Skills(context.Context) ([]types.Skill, error) {
	return nil, nil
}
func (a *steerManagerAgent) Prompt(context.Context, *types.State) error  { return nil }
func (a *steerManagerAgent) Approve(any, jsonRpc.ApprovalDecision) error { return nil }
func (a *steerManagerAgent) Steer(context.Context, *types.State) error {
	a.steerCalls++
	return a.steerErr
}
func (a *steerManagerAgent) Cancel(context.Context, string) error {
	a.cancelCalls++
	return a.cancelErr
}
func (a *steerManagerAgent) Stop() error { return nil }

func newSteerTestManager(t *testing.T, agent types.Agent) (*Manager, db.IStorage) {
	t.Helper()
	storage, err := db.NewSQLiteStorage(filepath.Join(t.TempDir(), "steer.db"))
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	t.Cleanup(func() { storage.Close() })
	msgBus := bus.NewMessageBus(8)
	t.Cleanup(func() { _ = msgBus.Close() })
	manager := NewManager(msgBus, db.NewData(storage))
	manager.RegisterAgent(agent.AgentID(), agent, true)
	return manager, storage
}

func steerTestTurn(t *testing.T, storage db.IStorage, reqID string) db.SessionTurn {
	t.Helper()
	var turn db.SessionTurn
	if err := storage.GetDB().Where("req_id = ?", reqID).First(&turn).Error; err != nil {
		t.Fatalf("查询 steer turn 失败: %v", err)
	}
	return turn
}

func TestAgentSteerSuccessDoesNotPersistIndependentTurn(t *testing.T) {
	agent := &steerManagerAgent{}
	manager, storage := newSteerTestManager(t, agent)
	state := types.NewState(&types.InputMessage{
		ID:      "req-steer",
		Session: "ws_u1_c1",
		Content: "补充要求",
	})

	manager.agentSteer(context.Background(), state)

	if agent.steerCalls != 1 {
		t.Fatalf("应调用一次 Steer，实际 %d", agent.steerCalls)
	}
	var count int64
	if err := storage.GetDB().Model(&db.SessionTurn{}).Where("req_id = ?", state.ReqID).Count(&count).Error; err != nil {
		t.Fatalf("统计 steer turn 失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("steer 不应落成独立轮次，实际 %d 条", count)
	}
}

func TestAgentSteerFailureDoesNotPersistIndependentTurn(t *testing.T) {
	agent := &steerManagerAgent{steerErr: errors.New("no running turn")}
	manager, storage := newSteerTestManager(t, agent)
	state := types.NewState(&types.InputMessage{
		ID:      "req-steer-fail",
		Session: "ws_u1_c1",
		Content: "补充要求",
	})

	manager.agentSteer(context.Background(), state)

	var count int64
	if err := storage.GetDB().Model(&db.SessionTurn{}).Where("req_id = ?", state.ReqID).Count(&count).Error; err != nil {
		t.Fatalf("统计 steer turn 失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("steer 失败也不应落成独立轮次，实际 %d 条", count)
	}
}

func TestCancelSessionCompletesRunningTurn(t *testing.T) {
	agent := &steerManagerAgent{}
	manager, storage := newSteerTestManager(t, agent)
	ctx := context.Background()
	sessionID := "ws_u1_c1"
	reqID := "req-running"
	if err := manager.dataStorage.Session().RequestSave(ctx, sessionID, reqID, "问题", agent.AgentID()); err != nil {
		t.Fatalf("保存请求失败: %v", err)
	}
	if err := manager.dataStorage.Session().TurnSave(ctx, &db.SessionTurn{
		ReqID:          reqID,
		TurnID:         "turn-1",
		RunTurnID:      "turn-1",
		ThreadID:       "thread-1",
		SessionID:      sessionID,
		Agent:          agent.AgentID(),
		RecoveryStatus: db.RecoveryStatusRunning,
	}); err != nil {
		t.Fatalf("保存 turn 失败: %v", err)
	}
	manager.sequence = &sessionSequence{
		runningSessions: map[string]string{sessionID: reqID},
		pendingMessages: make(map[string][]*types.State),
	}

	if !manager.CancelSession(sessionID) {
		t.Fatal("取消应成功")
	}
	if agent.cancelCalls != 1 {
		t.Fatalf("应调用一次 Cancel，实际 %d", agent.cancelCalls)
	}
	turn := steerTestTurn(t, storage, reqID)
	if !turn.IsCompleted || turn.CompletedAt == nil {
		t.Fatalf("主动取消后逻辑轮次应立即完成: %+v", turn)
	}
}

func TestCancelSessionFailureKeepsTurnRunning(t *testing.T) {
	agent := &steerManagerAgent{cancelErr: errors.New("interrupt failed")}
	manager, storage := newSteerTestManager(t, agent)
	ctx := context.Background()
	sessionID := "ws_u1_c1"
	reqID := "req-running"
	if err := manager.dataStorage.Session().RequestSave(ctx, sessionID, reqID, "问题", agent.AgentID()); err != nil {
		t.Fatalf("保存请求失败: %v", err)
	}
	manager.sequence = &sessionSequence{
		runningSessions: map[string]string{sessionID: reqID},
		pendingMessages: make(map[string][]*types.State),
	}

	if manager.CancelSession(sessionID) {
		t.Fatal("Cancel 失败时不应报告成功")
	}
	turn := steerTestTurn(t, storage, reqID)
	if turn.IsCompleted {
		t.Fatalf("Cancel 失败时不应提前完成轮次: %+v", turn)
	}
}
