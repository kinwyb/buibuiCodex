package cdm

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/db"
)

func newReconcileTestStore(t *testing.T) db.ISession {
	t.Helper()
	storage, err := db.NewSQLiteStorage(filepath.Join(t.TempDir(), "reconcile.db"))
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	data := db.NewData(storage)
	t.Cleanup(data.Close)
	return data.Session()
}

func reconcileTurnResult(turnStatus string, items ...string) *jsonRpc.Turn {
	rawItems := make([]jsonRpc.ThreadItem, 0, len(items))
	for _, item := range items {
		rawItems = append(rawItems, json.RawMessage(item))
	}
	return &jsonRpc.Turn{
		TurnID: "u1",
		Status: turnStatus,
		Items:  rawItems,
	}
}

func saveOpenCommand(t *testing.T, store db.ISession) {
	t.Helper()
	err := store.TurnEventSave(context.Background(), &db.TurnEvent{
		ID:         "started-i1",
		ThreadID:   "t1",
		TurnID:     "u1",
		Type:       "notification",
		Method:     "item/started",
		RawData:    `{"item":{"id":"i1","type":"commandExecution","status":"inProgress","command":"curl"}}`,
		CreateTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("写入 started 事件失败: %v", err)
	}
}

func TestReconcileTurnFillsCompletedCommand(t *testing.T) {
	store := newReconcileTestStore(t)
	saveOpenCommand(t, store)
	reconciler := newTurnReconciler(store, func(context.Context, string, string) (*jsonRpc.Turn, error) {
		return reconcileTurnResult("completed",
			`{"id":"i1","type":"commandExecution","status":"completed","command":"curl","aggregatedOutput":"ok","exitCode":0}`,
		), nil
	})

	done, err := reconciler.reconcileTurn(context.Background(), "t1", "u1")
	if err != nil {
		t.Fatalf("补全失败: %v", err)
	}
	if !done {
		t.Fatal("终态 item 应补全完成")
	}

	events, err := store.TurnEventList(context.Background(), db.TurnEventQuery{
		TurnIDs: []string{"u1"},
		Methods: []string{"item/started", "item/completed"},
		Limit:   -1,
	})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("应有 started + completed 两条事件，实际 %d", len(events))
	}
	completed := events[1]
	if completed.Method != "item/completed" {
		t.Fatalf("第二条应为 item/completed，实际 %s", completed.Method)
	}
	var payload struct {
		Item struct {
			Status   string `json:"status"`
			Output   string `json:"aggregatedOutput"`
			ExitCode int    `json:"exitCode"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(completed.RawData), &payload); err != nil {
		t.Fatalf("解析补全事件失败: %v", err)
	}
	if payload.Item.Status != "completed" || payload.Item.Output != "ok" || payload.Item.ExitCode != 0 {
		t.Fatalf("补全内容不符: %+v", payload.Item)
	}

	// 再次执行应命中稳定 ID，不产生重复完成帧
	done, err = reconciler.reconcileTurn(context.Background(), "t1", "u1")
	if err != nil || !done {
		t.Fatalf("幂等补全失败: done=%v err=%v", done, err)
	}
	events, err = store.TurnEventList(context.Background(), db.TurnEventQuery{
		TurnIDs: []string{"u1"},
		Methods: []string{"item/started", "item/completed"},
		Limit:   -1,
	})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("重复补全不应新增事件，实际 %d 条", len(events))
	}
}

func TestReconcileTurnWaitsForInProgressCommand(t *testing.T) {
	store := newReconcileTestStore(t)
	saveOpenCommand(t, store)
	reconciler := newTurnReconciler(store, func(context.Context, string, string) (*jsonRpc.Turn, error) {
		return reconcileTurnResult("completed",
			`{"id":"i1","type":"commandExecution","status":"inProgress","command":"curl"}`,
		), nil
	})

	done, err := reconciler.reconcileTurn(context.Background(), "t1", "u1")
	if err != nil {
		t.Fatalf("补全失败: %v", err)
	}
	if done {
		t.Fatal("inProgress item 不应被判定为完成")
	}
	events, err := store.TurnEventList(context.Background(), db.TurnEventQuery{
		TurnIDs: []string{"u1"},
		Methods: []string{"item/started", "item/completed"},
		Limit:   -1,
	})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("inProgress 时不应写完成帧，实际 %d 条", len(events))
	}
}

func TestReconcileTurnMarksInProgressFailedWhenTurnFailed(t *testing.T) {
	store := newReconcileTestStore(t)
	saveOpenCommand(t, store)
	reconciler := newTurnReconciler(store, func(context.Context, string, string) (*jsonRpc.Turn, error) {
		return reconcileTurnResult("failed",
			`{"id":"i1","type":"commandExecution","status":"inProgress","command":"curl"}`,
		), nil
	})

	done, err := reconciler.reconcileTurn(context.Background(), "t1", "u1")
	if err != nil {
		t.Fatalf("补全失败: %v", err)
	}
	if !done {
		t.Fatal("失败 turn 中仍在运行的 item 应收敛为终态")
	}
	events, err := store.TurnEventList(context.Background(), db.TurnEventQuery{
		TurnIDs: []string{"u1"},
		Methods: []string{"item/completed"},
		Limit:   -1,
	})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("应写入一条 failed 完成帧，实际 %d 条", len(events))
	}
	var payload struct {
		Item struct {
			Status string `json:"status"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(events[0].RawData), &payload); err != nil {
		t.Fatalf("解析补全事件失败: %v", err)
	}
	if payload.Item.Status != "failed" {
		t.Fatalf("失败 turn 的 inProgress item 应标记 failed，实际 %s", payload.Item.Status)
	}
}
