package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newRecoveryTestStorage(t *testing.T) ISession {
	t.Helper()
	storage, err := NewSQLiteStorage(filepath.Join(t.TempDir(), "recovery.db"))
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	data := NewData(storage)
	t.Cleanup(data.Close)
	return data.Session()
}

func TestTurnRecoveryUsesLogicalAndRunIDs(t *testing.T) {
	s := newRecoveryTestStorage(t)
	ctx := context.Background()
	if err := s.SessionSave(ctx, &Session{SessionID: "s1", Channel: "ws"}); err != nil {
		t.Fatalf("保存 session 失败: %v", err)
	}
	if err := s.ThreadSave(ctx, &SessionThread{ThreadID: "t1", SessionID: "s1", Agent: "codex"}); err != nil {
		t.Fatalf("保存 thread 失败: %v", err)
	}
	if err := s.RequestSave(ctx, "s1", "r1", "问题", "codex"); err != nil {
		t.Fatalf("保存请求失败: %v", err)
	}
	if err := s.TurnSave(ctx, &SessionTurn{
		ReqID:          "r1",
		TurnID:         "logical-1",
		RunTurnID:      "run-1",
		ThreadID:       "t1",
		SessionID:      "s1",
		Agent:          "codex",
		RecoveryStatus: RecoveryStatusRunning,
	}); err != nil {
		t.Fatalf("保存 turn 失败: %v", err)
	}
	if err := s.RequestSave(ctx, "s1", "legacy", "历史失败", "codex"); err != nil {
		t.Fatalf("保存历史请求失败: %v", err)
	}
	if err := s.TurnSave(ctx, &SessionTurn{ReqID: "legacy", TurnID: "legacy-1", RunTurnID: "legacy-1", ThreadID: "t1", SessionID: "s1", Agent: "codex"}); err != nil {
		t.Fatalf("保存历史 turn 失败: %v", err)
	}

	rows, err := s.TurnListRecoverableByAgent(ctx, "codex", time.Now().Add(-time.Hour), 10)
	if err != nil {
		t.Fatalf("查询待恢复 turn 失败: %v", err)
	}
	if len(rows) != 1 || rows[0].TurnID != "logical-1" || rows[0].RunTurnID != "run-1" {
		t.Fatalf("待恢复 turn 不符合预期: %+v", rows)
	}

	if err := s.TurnRecoveryUpdate(ctx, "r1", "run-2", RecoveryStatusRecovering, "", 1); err != nil {
		t.Fatalf("更新恢复状态失败: %v", err)
	}
	got := s.TurnQueryByID(ctx, "logical-1")
	if got == nil || got.RunTurnID != "run-2" || got.RecoveryStatus != RecoveryStatusRecovering || got.RecoveryAttempt != 1 {
		t.Fatalf("逻辑 ID 应保持不变，实际运行 ID 应更新: %+v", got)
	}
}

func TestTurnEventOpenTurnsSeparatesRunsUnderSameLogicalTurn(t *testing.T) {
	s := newRecoveryTestStorage(t)
	ctx := context.Background()
	base := time.Now()
	events := []TurnEvent{
		{ID: "r1-start", ThreadID: "t1", TurnID: "logical-1", RunTurnID: "run-1", Method: "item/started", RawData: `{"item":{"id":"i1"}}`, CreateTime: base},
		{ID: "r2-start", ThreadID: "t1", TurnID: "logical-1", RunTurnID: "run-2", Method: "item/started", RawData: `{"item":{"id":"i2"}}`, CreateTime: base},
		{ID: "r2-done", ThreadID: "t1", TurnID: "logical-1", RunTurnID: "run-2", Method: "item/completed", RawData: `{"item":{"id":"i2"}}`, CreateTime: base},
	}
	for i := range events {
		if err := s.TurnEventSave(ctx, &events[i]); err != nil {
			t.Fatalf("保存事件失败: %v", err)
		}
	}

	open, err := s.TurnEventOpenTurns(ctx, base.Add(-time.Minute), 10)
	if err != nil {
		t.Fatalf("扫描未完成 run 失败: %v", err)
	}
	if len(open) != 1 || open[0].TurnID != "logical-1" || open[0].RunTurnID != "run-1" {
		t.Fatalf("应只返回 run-1 的开放 item，实际 %+v", open)
	}

	all, err := s.TurnEventList(ctx, TurnEventQuery{TurnIDs: []string{"logical-1"}})
	if err != nil {
		t.Fatalf("按逻辑 ID 查询事件失败: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("前端按逻辑 ID 应看到全部 run 的事件，实际 %d 条", len(all))
	}
}

func TestTurnEventDeleteByRunOnlyDeletesCurrentRun(t *testing.T) {
	s := newRecoveryTestStorage(t)
	ctx := context.Background()
	base := time.Now()
	for _, event := range []TurnEvent{
		{ID: "d1", ThreadID: "t1", TurnID: "logical-1", RunTurnID: "run-1", Method: "item/agentMessage/delta", RawData: `{"itemId":"i1","delta":"旧"}`, CreateTime: base},
		{ID: "d2", ThreadID: "t1", TurnID: "logical-1", RunTurnID: "run-2", Method: "item/agentMessage/delta", RawData: `{"itemId":"i2","delta":"新"}`, CreateTime: base},
	} {
		event := event
		if err := s.TurnEventSave(ctx, &event); err != nil {
			t.Fatalf("保存事件失败: %v", err)
		}
	}
	if err := s.TurnEventDeleteByRun(ctx, "run-1", []string{"item/agentMessage/delta"}); err != nil {
		t.Fatalf("按 run 清理失败: %v", err)
	}
	events, err := s.TurnEventList(ctx, TurnEventQuery{TurnIDs: []string{"logical-1"}})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(events) != 1 || events[0].RunTurnID != "run-2" {
		t.Fatalf("只应删除 run-1 的 delta，实际 %+v", events)
	}
}

func TestInitBackfillsLegacyRunTurnID(t *testing.T) {
	storage, err := NewSQLiteStorage(filepath.Join(t.TempDir(), "backfill.db"))
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	data := NewData(storage)
	defer data.Close()
	s := data.Session()
	ctx := context.Background()

	if err := s.RequestSave(ctx, "s1", "r1", "旧问题", "codex"); err != nil {
		t.Fatalf("保存请求失败: %v", err)
	}
	if err := s.TurnSave(ctx, &SessionTurn{
		ReqID: "r1", TurnID: "legacy-turn", ThreadID: "t1", SessionID: "s1", Agent: "codex", IsCompleted: true,
	}); err != nil {
		t.Fatalf("保存旧 turn 失败: %v", err)
	}
	if err := s.TurnEventSave(ctx, &TurnEvent{
		ID: "legacy-event", ThreadID: "t1", TurnID: "legacy-turn", Method: "item/completed", RawData: `{"item":{"id":"i1"}}`,
	}); err != nil {
		t.Fatalf("保存旧事件失败: %v", err)
	}

	// 模拟升级后再次启动，Init 会把两张表的空 RunTurnID 回填为 TurnID。
	s.Init()
	turn := s.TurnQueryByID(ctx, "legacy-turn")
	if turn == nil || turn.RunTurnID != "legacy-turn" {
		t.Fatalf("session_turns 未正确回填: %+v", turn)
	}
	events, err := s.TurnEventList(ctx, TurnEventQuery{TurnIDs: []string{"legacy-turn"}})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(events) != 1 || events[0].RunTurnID != "legacy-turn" {
		t.Fatalf("turn_events 未正确回填: %+v", events)
	}
}
