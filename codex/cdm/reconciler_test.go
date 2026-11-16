package cdm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
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
	return reconcileTurnResultFor("u1", turnStatus, items...)
}

func reconcileTurnResultFor(turnID, turnStatus string, items ...string) *jsonRpc.Turn {
	rawItems := make([]jsonRpc.ThreadItem, 0, len(items))
	for _, item := range items {
		rawItems = append(rawItems, json.RawMessage(item))
	}
	return &jsonRpc.Turn{
		TurnID: turnID,
		Status: turnStatus,
		Items:  rawItems,
	}
}

func saveOpenCommand(t *testing.T, store db.ISession) {
	saveOpenCommandFor(t, store, "u1", "i1", "started-i1")
}

func saveOpenCommandFor(t *testing.T, store db.ISession, turnID, itemID, eventID string) {
	t.Helper()
	err := store.TurnEventSave(context.Background(), &db.TurnEvent{
		ID:         eventID,
		ThreadID:   "t1",
		TurnID:     turnID,
		RunTurnID:  turnID,
		Type:       "notification",
		Method:     "item/started",
		RawData:    fmt.Sprintf(`{"item":{"id":%q,"type":"commandExecution","status":"inProgress","command":"curl"}}`, itemID),
		CreateTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("写入 started 事件失败: %v", err)
	}
}

func saveOpenReasoning(t *testing.T, store db.ISession, itemID string) {
	t.Helper()
	err := store.TurnEventSave(context.Background(), &db.TurnEvent{
		ID:         "started-" + itemID,
		ThreadID:   "t1",
		TurnID:     "u1",
		RunTurnID:  "u1",
		Type:       "notification",
		Method:     "item/started",
		RawData:    fmt.Sprintf(`{"item":{"id":%q,"type":"reasoning","content":[],"summary":[],"startedAtMs":1794555369944,"completedAtMs":null}}`, itemID),
		CreateTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("写入 reasoning started 事件失败: %v", err)
	}
}

func TestReconcileTurnFillsCompletedCommand(t *testing.T) {
	store := newReconcileTestStore(t)
	saveOpenCommand(t, store)
	reconciler := newTurnReconciler(store, func(context.Context, string, []string) (map[string]*jsonRpc.Turn, error) {
		return map[string]*jsonRpc.Turn{
			"u1": reconcileTurnResult("completed",
				`{"id":"i1","type":"commandExecution","status":"completed","command":"curl","aggregatedOutput":"ok","exitCode":0}`,
			),
		}, nil
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

func TestReconcileTurnFallsBackToStartedReasoning(t *testing.T) {
	for _, tc := range []struct {
		name        string
		withDeltas  bool
		wantSummary []string
		wantContent []string
	}{
		{name: "有delta", withDeltas: true, wantSummary: []string{"想一下"}, wantContent: []string{"思考"}},
		{name: "无delta", wantSummary: []string{}, wantContent: []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newReconcileTestStore(t)
			saveOpenReasoning(t, store, "msg_1")
			if tc.withDeltas {
				for _, event := range []*db.TurnEvent{
					{ID: "d1", ThreadID: "t1", TurnID: "u1", RunTurnID: "u1", Method: "item/reasoning/summaryTextDelta", RawData: `{"itemId":"msg_1","delta":"想","summaryIndex":0}`, CreateTime: time.Now()},
					{ID: "d2", ThreadID: "t1", TurnID: "u1", RunTurnID: "u1", Method: "item/reasoning/summaryTextDelta", RawData: `{"itemId":"msg_1","delta":"一下","summaryIndex":0}`, CreateTime: time.Now()},
					{ID: "d3", ThreadID: "t1", TurnID: "u1", RunTurnID: "u1", Method: "item/reasoning/textDelta", RawData: `{"itemId":"msg_1","delta":"思考"}`, CreateTime: time.Now()},
				} {
					if err := store.TurnEventSave(context.Background(), event); err != nil {
						t.Fatalf("写入 reasoning delta 失败: %v", err)
					}
				}
			}
			reconciler := newTurnReconciler(store, func(context.Context, string, []string) (map[string]*jsonRpc.Turn, error) {
				return map[string]*jsonRpc.Turn{
					"u1": reconcileTurnResult("completed"),
				}, nil
			})

			done, err := reconciler.reconcileTurn(context.Background(), "t1", "u1")
			if err != nil {
				t.Fatalf("补全 reasoning 失败: %v", err)
			}
			if !done {
				t.Fatal("app-server 缺少 reasoning item 时应按本地事件收尾")
			}

			events, err := store.TurnEventList(context.Background(), db.TurnEventQuery{
				TurnIDs: []string{"u1"},
				Methods: []string{"item/completed"},
			})
			if err != nil {
				t.Fatalf("查询完成事件失败: %v", err)
			}
			if len(events) != 1 {
				t.Fatalf("应补一条 reasoning 完成帧，实际 %d 条", len(events))
			}
			var payload struct {
				Item struct {
					Type        string   `json:"type"`
					Summary     []string `json:"summary"`
					Content     []string `json:"content"`
					CompletedAt int64    `json:"completedAtMs"`
				} `json:"item"`
			}
			if err := json.Unmarshal([]byte(events[0].RawData), &payload); err != nil {
				t.Fatalf("解析补全事件失败: %v", err)
			}
			if payload.Item.Type != "reasoning" || payload.Item.CompletedAt == 0 {
				t.Fatalf("补全 item 不完整: %+v", payload.Item)
			}
			if fmt.Sprint(payload.Item.Summary) != fmt.Sprint(tc.wantSummary) {
				t.Fatalf("summary 不符: want %v, got %v", tc.wantSummary, payload.Item.Summary)
			}
			if fmt.Sprint(payload.Item.Content) != fmt.Sprint(tc.wantContent) {
				t.Fatalf("content 不符: want %v, got %v", tc.wantContent, payload.Item.Content)
			}
		})
	}
}

func TestReconcileTurnDoesNotFallbackWhileTurnInProgress(t *testing.T) {
	store := newReconcileTestStore(t)
	saveOpenReasoning(t, store, "msg_1")
	reconciler := newTurnReconciler(store, func(context.Context, string, []string) (map[string]*jsonRpc.Turn, error) {
		return map[string]*jsonRpc.Turn{
			"u1": reconcileTurnResult("inProgress"),
		}, nil
	})

	done, err := reconciler.reconcileTurn(context.Background(), "t1", "u1")
	if err != nil {
		t.Fatalf("检查运行中的 turn 失败: %v", err)
	}
	if done {
		t.Fatal("turn 仍在运行时不能提前补 reasoning 完成帧")
	}
	events, err := store.TurnEventList(context.Background(), db.TurnEventQuery{
		TurnIDs: []string{"u1"},
		Methods: []string{"item/completed"},
	})
	if err != nil {
		t.Fatalf("查询完成事件失败: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("turn 仍在运行时不应写完成帧，实际 %d 条", len(events))
	}
}

func TestReconcileTurnWaitsForInProgressCommand(t *testing.T) {
	store := newReconcileTestStore(t)
	saveOpenCommand(t, store)
	reconciler := newTurnReconciler(store, func(context.Context, string, []string) (map[string]*jsonRpc.Turn, error) {
		return map[string]*jsonRpc.Turn{
			"u1": reconcileTurnResult("completed",
				`{"id":"i1","type":"commandExecution","status":"inProgress","command":"curl"}`,
			),
		}, nil
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
	reconciler := newTurnReconciler(store, func(context.Context, string, []string) (map[string]*jsonRpc.Turn, error) {
		return map[string]*jsonRpc.Turn{
			"u1": reconcileTurnResult("failed",
				`{"id":"i1","type":"commandExecution","status":"inProgress","command":"curl"}`,
			),
		}, nil
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

func TestReconcileRunWritesLogicalAndRunIDs(t *testing.T) {
	store := newReconcileTestStore(t)
	if err := store.TurnEventSave(context.Background(), &db.TurnEvent{
		ID:         "started-run-1",
		ThreadID:   "t1",
		TurnID:     "logical-1",
		RunTurnID:  "run-1",
		Method:     "item/started",
		RawData:    `{"item":{"id":"i1","type":"commandExecution","status":"inProgress","command":"curl"}}`,
		CreateTime: time.Now(),
	}); err != nil {
		t.Fatalf("写入 started 事件失败: %v", err)
	}
	reconciler := newTurnReconciler(store, func(_ context.Context, threadID string, turnIDs []string) (map[string]*jsonRpc.Turn, error) {
		if threadID != "t1" || len(turnIDs) != 1 || turnIDs[0] != "run-1" {
			t.Fatalf("reconciler 应读取实际 RunTurnID: thread=%s turns=%v", threadID, turnIDs)
		}
		return map[string]*jsonRpc.Turn{
			"run-1": reconcileTurnResultFor("run-1", "interrupted",
				`{"id":"i1","type":"commandExecution","status":"inProgress","command":"curl"}`,
			),
		}, nil
	})

	done, err := reconciler.reconcileRun(context.Background(), "t1", "logical-1", "run-1")
	if err != nil || !done {
		t.Fatalf("补全 interrupted run 失败: done=%v err=%v", done, err)
	}
	events, err := store.TurnEventList(context.Background(), db.TurnEventQuery{
		TurnIDs: []string{"logical-1"},
		Methods: []string{"item/completed"},
	})
	if err != nil {
		t.Fatalf("查询补全事件失败: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("应补一条完成帧，实际 %d 条", len(events))
	}
	if events[0].TurnID != "logical-1" || events[0].RunTurnID != "run-1" {
		t.Fatalf("补全事件必须同时保留逻辑 ID 和实际运行 ID: %+v", events[0])
	}
}

func TestProcessThreadJobBatchesTurns(t *testing.T) {
	store := newReconcileTestStore(t)
	saveOpenCommandFor(t, store, "u1", "i1", "started-u1")
	saveOpenCommandFor(t, store, "u2", "i2", "started-u2")

	readCalls := 0
	reconciler := newTurnReconciler(store, func(_ context.Context, threadID string, turnIDs []string) (map[string]*jsonRpc.Turn, error) {
		readCalls++
		if threadID != "t1" {
			t.Fatalf("threadID 不符: %s", threadID)
		}
		if len(turnIDs) != 2 {
			t.Fatalf("同一 thread 的两个 turn 应一次读取，实际 %v", turnIDs)
		}
		return map[string]*jsonRpc.Turn{
			"u1": reconcileTurnResultFor("u1", "completed",
				`{"id":"i1","type":"commandExecution","status":"completed","aggregatedOutput":"one"}`,
			),
			"u2": reconcileTurnResultFor("u2", "completed",
				`{"id":"i2","type":"commandExecution","status":"completed","aggregatedOutput":"two"}`,
			),
		}, nil
	})

	outcomes := reconciler.processThreadJob(
		context.Background(),
		"t1",
		[]string{"u1", "u2"},
		map[string]string{"u1": "u1", "u2": "u2"},
	)
	if readCalls != 1 {
		t.Fatalf("同一 thread 应只调用一次 app-server 读取，实际 %d 次", readCalls)
	}
	if !outcomes["u1"].done || !outcomes["u2"].done {
		t.Fatalf("两个 turn 都应补全完成: %+v", outcomes)
	}
	if outcomes["u1"].fixed != 1 || outcomes["u2"].fixed != 1 {
		t.Fatalf("两个 turn 各应补一条完成帧: %+v", outcomes)
	}

	events, err := store.TurnEventList(context.Background(), db.TurnEventQuery{
		TurnIDs: []string{"u1", "u2"},
		Methods: []string{"item/completed"},
		Limit:   -1,
	})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("两个 turn 应各补一条完成帧，实际 %d 条", len(events))
	}
}

func TestEnqueueGroupsByThread(t *testing.T) {
	store := newReconcileTestStore(t)
	reconciler := newTurnReconciler(store, func(context.Context, string, []string) (map[string]*jsonRpc.Turn, error) {
		return nil, nil
	})
	now := time.Now()
	reconciler.now = func() time.Time { return now }
	reconciler.running = true

	reconciler.Enqueue("t1", "u1", "u1")
	reconciler.Enqueue("t1", "u2", "u2")
	reconciler.Enqueue("t2", "u3", "u3")

	if len(reconciler.jobs) != 2 {
		t.Fatalf("应按 thread 聚合为 2 个任务，实际 %d 个", len(reconciler.jobs))
	}
	job := reconciler.jobs["t1"]
	if job == nil || len(job.turns) != 2 {
		t.Fatalf("t1 任务应包含两个 turn: %+v", job)
	}
	if want := now.Add(reconciler.debounce); !job.next.Equal(want) {
		t.Fatalf("首次检查应延迟 %s，实际 next=%s", reconciler.debounce, job.next)
	}
}

func TestFinishThreadJobReschedulesPerTurn(t *testing.T) {
	store := newReconcileTestStore(t)
	reconciler := newTurnReconciler(store, func(context.Context, string, []string) (map[string]*jsonRpc.Turn, error) {
		return nil, nil
	})
	now := time.Now()
	reconciler.now = func() time.Time { return now }

	job := &threadReconcileJob{
		threadID: "t1",
		turns: map[string]*turnReconcileState{
			"u1": {
				logicalTurnID: "u1",
				runTurnID:     "u1",
				next:          now,
				deadline:      now.Add(time.Hour),
			},
		},
		next: now,
	}
	reconciler.jobs["t1"] = job

	reconciler.finishThreadJob(job, []string{"u1"}, map[string]turnOutcome{
		"u1": {done: false},
	})
	state := job.turns["u1"]
	if state == nil {
		t.Fatal("未完成 turn 应保留在任务中")
	}
	if state.attempts != 1 {
		t.Fatalf("重试次数应为 1，实际 %d", state.attempts)
	}
	if want := now.Add(reconcileBackoffAt(1)); !state.next.Equal(want) {
		t.Fatalf("下次重试时间不符: want %s, got %s", want, state.next)
	}
	if !job.next.Equal(state.next) {
		t.Fatalf("thread 任务应按最近 turn 重试时间调度: job=%s turn=%s", job.next, state.next)
	}

	reconciler.finishThreadJob(job, []string{"u1"}, map[string]turnOutcome{
		"u1": {done: true},
	})
	if reconciler.jobs["t1"] != nil {
		t.Fatal("所有 turn 完成后应删除 thread 任务")
	}
}

func TestFinishThreadJobLogsCompletion(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })

	reconciler := newTurnReconciler(nil, nil)
	now := time.Now()
	reconciler.now = func() time.Time { return now }
	job := &threadReconcileJob{
		threadID: "t1",
		turns: map[string]*turnReconcileState{
			"run-1": {
				logicalTurnID: "logical-1",
				runTurnID:     "run-1",
				next:          now,
				deadline:      now.Add(time.Hour),
			},
		},
		next: now,
	}
	reconciler.jobs["t1"] = job

	reconciler.finishThreadJob(job, []string{"run-1"}, map[string]turnOutcome{
		"run-1": {done: true, fixed: 1},
	})

	output := buf.String()
	for _, want := range []string{"turn 事件补全任务结束", "threadID=t1", "logicalTurnID=logical-1", "runTurnID=run-1", "items=1", "attempts=1"} {
		if !strings.Contains(output, want) {
			t.Fatalf("补全完成日志缺少 %q: %s", want, output)
		}
	}
}

func TestFinishThreadJobLogsPendingRetry(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })

	reconciler := newTurnReconciler(nil, nil)
	now := time.Now()
	reconciler.now = func() time.Time { return now }
	job := &threadReconcileJob{
		threadID: "t1",
		turns: map[string]*turnReconcileState{
			"run-1": {
				logicalTurnID: "logical-1",
				runTurnID:     "run-1",
				next:          now,
				deadline:      now.Add(time.Hour),
			},
		},
		next: now,
	}
	reconciler.jobs["t1"] = job

	reconciler.finishThreadJob(job, []string{"run-1"}, map[string]turnOutcome{
		"run-1": {open: 1, fixed: 0},
	})

	output := buf.String()
	for _, want := range []string{"turn 事件补全未完成，稍后重试", "threadID=t1", "runTurnID=run-1", "open=1", "items=0", "attempts=1"} {
		if !strings.Contains(output, want) {
			t.Fatalf("待重试日志缺少 %q: %s", want, output)
		}
	}
}
