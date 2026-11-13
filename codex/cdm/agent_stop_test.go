package cdm

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
)

func TestMarkActiveTurnsInterruptedCompletesLogicalTurn(t *testing.T) {
	storage, err := db.NewSQLiteStorage(filepath.Join(t.TempDir(), "agent-stop.db"))
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	data := db.NewData(storage)
	defer data.Close()
	sess := data.Session()
	ctx := context.Background()
	if err := sess.RequestSave(ctx, "s1", "r1", "问题", "codex"); err != nil {
		t.Fatalf("保存请求失败: %v", err)
	}
	if err := sess.TurnSave(ctx, &db.SessionTurn{
		ReqID:          "r1",
		TurnID:         "run-1",
		RunTurnID:      "run-1",
		ThreadID:       "t1",
		SessionID:      "s1",
		Agent:          "codex",
		RecoveryStatus: db.RecoveryStatusRunning,
	}); err != nil {
		t.Fatalf("保存 turn 失败: %v", err)
	}

	turn, err := codex.NewTurnWithID(codex.NewThread(nil), "run-1")
	if err != nil {
		t.Fatalf("构建 turn 失败: %v", err)
	}
	a := NewAgent(&config.AgentConfig{Name: "codex", SessionDB: sess}, 0)
	a.eps["s1"] = &codexTurn{
		state: types.NewState(&types.InputMessage{ID: "r1", Session: "s1"}),
		turn:  turn,
	}

	a.markActiveTurnsInterrupted(ctx)

	got := sess.TurnQueryByID(ctx, "run-1")
	if got == nil || !got.IsCompleted || got.CompletedAt == nil {
		t.Fatalf("强制停止后逻辑轮次应完成: %+v", got)
	}
	if got.RecoveryStatus != db.RecoveryStatusInterrupted {
		t.Fatalf("强制停止的运行状态应为 interrupted: %+v", got)
	}
}
