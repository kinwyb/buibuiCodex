package cdm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/db"
)

const (
	defaultReconcileInterval = time.Second
	defaultReconcileDebounce = 3 * time.Second
	defaultReconcileScanAge  = 7 * 24 * time.Hour
	defaultReconcileReadWait = 20 * time.Second
	defaultReconcileMaxAge   = time.Hour
	defaultReconcileScanSize = 500
)

var reconcileBackoff = []time.Duration{
	0,
	5 * time.Second,
	15 * time.Second,
	30 * time.Second,
	time.Minute,
	2 * time.Minute,
	5 * time.Minute,
	10 * time.Minute,
	15 * time.Minute,
	30 * time.Minute,
}

type turnReconcileState struct {
	logicalTurnID string
	runTurnID     string
	attempts      int
	next          time.Time
	deadline      time.Time
}

type threadReconcileJob struct {
	threadID   string
	turns      map[string]*turnReconcileState
	next       time.Time
	processing bool
}

type turnOutcome struct {
	done  bool
	open  int
	fixed int
	err   error
}

// turnReconciler 以 thread 为调度单位、以 turn 为重试单位，对账 turn_events
// 与 app-server 的完整 thread 历史。同一 thread 下的多个异常 turn 合并为一次读取。
type turnReconciler struct {
	store     db.ISession
	readTurns func(context.Context, string, []string) (map[string]*jsonRpc.Turn, error)

	mu      sync.Mutex
	jobs    map[string]*threadReconcileJob
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	interval time.Duration
	debounce time.Duration
	scanAge  time.Duration
	readWait time.Duration
	maxAge   time.Duration
	now      func() time.Time
}

func newTurnReconciler(store db.ISession, readTurns func(context.Context, string, []string) (map[string]*jsonRpc.Turn, error)) *turnReconciler {
	return &turnReconciler{
		store:     store,
		readTurns: readTurns,
		jobs:      make(map[string]*threadReconcileJob),
		interval:  defaultReconcileInterval,
		debounce:  defaultReconcileDebounce,
		scanAge:   defaultReconcileScanAge,
		readWait:  defaultReconcileReadWait,
		maxAge:    defaultReconcileMaxAge,
		now:       time.Now,
	}
}

func (r *turnReconciler) Start() {
	if r == nil || r.store == nil || r.readTurns == nil {
		return
	}
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	r.running = true
	r.wg.Add(2)
	r.mu.Unlock()

	go r.loop()
	go r.scanOpenTurns()
}

func (r *turnReconciler) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}
	r.running = false
	cancel := r.cancel
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	r.wg.Wait()

	r.mu.Lock()
	for _, job := range r.jobs {
		job.processing = false
	}
	r.mu.Unlock()
}

func (r *turnReconciler) Enqueue(threadID, logicalTurnID, runTurnID string) {
	if r == nil || r.store == nil || r.readTurns == nil || threadID == "" || runTurnID == "" {
		return
	}
	if logicalTurnID == "" {
		logicalTurnID = runTurnID
	}
	now := r.now()

	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running {
		return
	}
	job := r.jobs[threadID]
	if job == nil {
		job = &threadReconcileJob{
			threadID: threadID,
			turns:    make(map[string]*turnReconcileState),
		}
		r.jobs[threadID] = job
	}
	firstCheck := now.Add(r.debounce)
	if state := job.turns[runTurnID]; state != nil {
		if state.next.After(firstCheck) {
			state.next = firstCheck
		}
	} else {
		job.turns[runTurnID] = &turnReconcileState{
			logicalTurnID: logicalTurnID,
			runTurnID:     runTurnID,
			next:          firstCheck,
			deadline:      now.Add(r.maxAge),
		}
	}
	job.recalcNext()
}

func (r *turnReconciler) loop() {
	defer r.wg.Done()
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.processDue()
		}
	}
}

func (r *turnReconciler) scanOpenTurns() {
	defer r.wg.Done()
	if r.ctx == nil {
		return
	}
	since := r.now().Add(-r.scanAge)
	turns, err := r.store.TurnEventOpenTurns(r.ctx, since, defaultReconcileScanSize)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("扫描未完成 turn 事件失败", "error", err.Error())
		}
		return
	}
	for _, turn := range turns {
		r.Enqueue(turn.ThreadID, turn.TurnID, turn.RunTurnID)
	}
	if len(turns) > 0 {
		ids := make([]string, 0, len(turns))
		for _, turn := range turns {
			ids = append(ids, turn.ThreadID+"/"+turn.TurnID+"/"+turn.RunTurnID)
		}
		slog.Info("发现未完成 turn 事件，已按 thread 加入补全队列", "count", len(turns), "turns", ids)
	}
}

func (r *turnReconciler) processDue() {
	now := r.now()
	type workItem struct {
		job     *threadReconcileJob
		turnIDs []string
	}
	work := make([]workItem, 0)

	r.mu.Lock()
	for _, job := range r.jobs {
		if job.processing || job.next.After(now) {
			continue
		}
		turnIDs := make([]string, 0, len(job.turns))
		for turnID, state := range job.turns {
			if !state.next.After(now) {
				turnIDs = append(turnIDs, turnID)
			}
		}
		if len(turnIDs) == 0 {
			job.recalcNext()
			continue
		}
		job.processing = true
		work = append(work, workItem{job: job, turnIDs: turnIDs})
	}
	r.mu.Unlock()

	for _, item := range work {
		if r.ctx == nil || r.ctx.Err() != nil {
			return
		}
		logicalByRun := make(map[string]string, len(item.turnIDs))
		for _, runTurnID := range item.turnIDs {
			if state := item.job.turns[runTurnID]; state != nil {
				logicalByRun[runTurnID] = state.logicalTurnID
			}
		}
		outcomes := r.processThreadJob(r.ctx, item.job.threadID, item.turnIDs, logicalByRun)
		r.finishThreadJob(item.job, item.turnIDs, outcomes)
	}
}

func (r *turnReconciler) processThreadJob(ctx context.Context, threadID string, turnIDs []string, logicalByRun map[string]string) map[string]turnOutcome {
	outcomes := make(map[string]turnOutcome, len(turnIDs))
	if len(turnIDs) == 0 {
		return outcomes
	}
	slog.Info("开始补全 turn 事件", "threadID", threadID, "turns", turnIDs)

	events, err := r.store.TurnEventList(ctx, db.TurnEventQuery{
		RunTurnIDs: turnIDs,
		Methods:    []string{"item/started", "item/completed"},
		Limit:      -1,
	})
	if err != nil {
		for _, turnID := range turnIDs {
			outcomes[turnID] = turnOutcome{err: fmt.Errorf("读取 turn 事件失败: %w", err)}
		}
		return outcomes
	}

	byTurn := make(map[string][]db.TurnEvent, len(turnIDs))
	for _, event := range events {
		runTurnID := event.RunTurnID
		if runTurnID == "" {
			runTurnID = event.TurnID
		}
		byTurn[runTurnID] = append(byTurn[runTurnID], event)
	}

	openByTurn := make(map[string]map[string]db.TurnEvent, len(turnIDs))
	needRead := make([]string, 0, len(turnIDs))
	for _, turnID := range turnIDs {
		open := openLifecycleItems(byTurn[turnID])
		if len(open) == 0 {
			outcomes[turnID] = turnOutcome{done: true, open: 0}
			continue
		}
		openByTurn[turnID] = open
		needRead = append(needRead, turnID)
	}
	if len(needRead) == 0 {
		return outcomes
	}

	readCtx, cancel := context.WithTimeout(ctx, r.readWait)
	defer cancel()
	turns, err := r.readTurns(readCtx, threadID, needRead)
	if err != nil {
		for _, turnID := range needRead {
			outcomes[turnID] = turnOutcome{err: fmt.Errorf("读取 app-server turn 失败: %w", err)}
		}
		return outcomes
	}

	for _, runTurnID := range needRead {
		turn := turns[runTurnID]
		if turn == nil {
			outcomes[runTurnID] = turnOutcome{err: fmt.Errorf("app-server 中未找到 turn")}
			continue
		}
		logicalTurnID := logicalByRun[runTurnID]
		if logicalTurnID == "" {
			logicalTurnID = runTurnID
		}
		done, fixed, err := r.reconcileOpenItems(ctx, threadID, logicalTurnID, runTurnID, turn, openByTurn[runTurnID])
		outcomes[runTurnID] = turnOutcome{done: done, open: len(openByTurn[runTurnID]), fixed: fixed, err: err}
	}
	return outcomes
}

func (r *turnReconciler) finishThreadJob(job *threadReconcileJob, turnIDs []string, outcomes map[string]turnOutcome) {
	now := r.now()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.jobs[job.threadID] != job {
		return
	}
	job.processing = false

	for _, turnID := range turnIDs {
		state := job.turns[turnID]
		if state == nil {
			continue
		}
		outcome := outcomes[turnID]
		if outcome.done {
			slog.Info("turn 事件补全任务结束",
				"threadID", job.threadID,
				"logicalTurnID", state.logicalTurnID,
				"runTurnID", turnID,
				"items", outcome.fixed,
				"attempts", state.attempts+1,
			)
			delete(job.turns, turnID)
			continue
		}
		switch {
		case outcome.err != nil:
			slog.Warn("turn 事件补全失败", "threadID", job.threadID, "turnID", turnID, "error", outcome.err.Error())
		default:
			slog.Warn("turn 事件补全未完成，稍后重试",
				"threadID", job.threadID,
				"logicalTurnID", state.logicalTurnID,
				"runTurnID", turnID,
				"open", outcome.open,
				"items", outcome.fixed,
				"attempts", state.attempts+1,
			)
		}
		if now.After(state.deadline) {
			slog.Error("turn 事件补全超时，停止重试",
				"threadID", job.threadID, "turnID", turnID, "attempts", state.attempts+1)
			delete(job.turns, turnID)
			continue
		}
		state.attempts++
		state.next = now.Add(reconcileBackoffAt(state.attempts))
	}

	job.recalcNext()
	if len(job.turns) == 0 {
		delete(r.jobs, job.threadID)
	}
}

// reconcileTurn 保留单 turn 入口，供单元测试和手动补全使用。
func (r *turnReconciler) reconcileTurn(ctx context.Context, threadID, turnID string) (bool, error) {
	// 兼容旧调用：首次运行时逻辑 ID 与实际 ID 相同。
	return r.reconcileRun(ctx, threadID, turnID, turnID)
}

// reconcileRun 只补某一次实际运行的事件。逻辑 TurnID 用于落库，
// RunTurnID 用于从 app-server 读取对应 turn，二者不能混用。
func (r *turnReconciler) reconcileRun(ctx context.Context, threadID, logicalTurnID, runTurnID string) (bool, error) {
	if logicalTurnID == "" {
		logicalTurnID = runTurnID
	}
	events, err := r.store.TurnEventList(ctx, db.TurnEventQuery{
		RunTurnIDs: []string{runTurnID},
		Methods:    []string{"item/started", "item/completed"},
		Limit:      -1,
	})
	if err != nil {
		return false, fmt.Errorf("读取 turn 事件失败: %w", err)
	}
	open := openLifecycleItems(events)
	if len(open) == 0 {
		return true, nil
	}

	readCtx, cancel := context.WithTimeout(ctx, r.readWait)
	defer cancel()
	turns, err := r.readTurns(readCtx, threadID, []string{runTurnID})
	if err != nil {
		return false, fmt.Errorf("读取 app-server turn 失败: %w", err)
	}
	turn := turns[runTurnID]
	if turn == nil {
		return false, fmt.Errorf("app-server 中未找到 turn")
	}
	done, _, err := r.reconcileOpenItems(ctx, threadID, logicalTurnID, runTurnID, turn, open)
	return done, err
}

func (r *turnReconciler) reconcileOpenItems(ctx context.Context, threadID, logicalTurnID, runTurnID string, turn *jsonRpc.Turn, open map[string]db.TurnEvent) (bool, int, error) {
	fixed := 0
	for itemID, started := range open {
		rawItem, ok := findThreadItem(turn, itemID)
		if !ok {
			if turn.Status == "inProgress" {
				slog.Info("turn 仍在运行，等待 app-server 持久化 item",
					"threadID", threadID,
					"logicalTurnID", logicalTurnID,
					"runTurnID", runTurnID,
					"itemID", itemID,
				)
				continue
			}
			fallback, fallbackOK, err := r.reasoningCompletionFromStarted(ctx, runTurnID, itemID, started.RawData)
			if err != nil {
				return false, fixed, fmt.Errorf("根据本地事件补全 reasoning item 失败: %w", err)
			}
			if !fallbackOK {
				slog.Warn("app-server 中未找到 item",
					"threadID", threadID,
					"logicalTurnID", logicalTurnID,
					"runTurnID", runTurnID,
					"itemID", itemID,
				)
				continue
			}
			rawItem = fallback
			slog.Info("app-server 缺少 reasoning item，已按本地事件补全",
				"threadID", threadID,
				"logicalTurnID", logicalTurnID,
				"runTurnID", runTurnID,
				"itemID", itemID,
			)
		}
		final, normalized, err := finalItemPayload(rawItem, turn.Status)
		if err != nil {
			slog.Warn("解析 app-server item 失败", "logicalTurnID", logicalTurnID, "runTurnID", runTurnID, "itemID", itemID, "error", err.Error())
			continue
		}
		if !final {
			slog.Info("turn 事件补全等待 item 进入终态",
				"threadID", threadID,
				"logicalTurnID", logicalTurnID,
				"runTurnID", runTurnID,
				"itemID", itemID,
				"turnStatus", turn.Status,
			)
			continue
		}
		rawData, err := completionEventRaw(threadID, runTurnID, normalized)
		if err != nil {
			return false, fixed, fmt.Errorf("构造 item/completed 事件失败: %w", err)
		}
		event := &db.TurnEvent{
			ID:         stableLifecycleEventID(runTurnID, itemID, "item/completed"),
			ThreadID:   threadID,
			TurnID:     logicalTurnID,
			RunTurnID:  runTurnID,
			Type:       "notification",
			Method:     "item/completed",
			RawData:    string(rawData),
			CreateTime: r.now(),
		}
		if err := r.store.TurnEventUpsert(ctx, event); err != nil {
			return false, fixed, fmt.Errorf("写入补全事件失败: %w", err)
		}
		fixed++
	}
	if fixed > 0 {
		slog.Info("已补全 turn 事件", "threadID", threadID, "logicalTurnID", logicalTurnID, "runTurnID", runTurnID, "items", fixed, "remaining", len(open)-fixed)
	}
	return fixed == len(open), fixed, nil
}

// reasoningCompletionFromStarted 处理 app-server 未持久化的空 reasoning item。
// reasoning 的 started 载荷只有 item ID，完整文本在 delta 事件里；若上游没有
// 保存该 item，就按本地 delta 重建，连 delta 都没有时补一个空终态即可收尾。
func (r *turnReconciler) reasoningCompletionFromStarted(ctx context.Context, runTurnID, itemID, startedRaw string) (json.RawMessage, bool, error) {
	var started struct {
		Item struct {
			ID          string   `json:"id"`
			Type        string   `json:"type"`
			StartedAtMs int64    `json:"startedAtMs"`
			Summary     []string `json:"summary"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(startedRaw), &started); err != nil {
		return nil, false, err
	}
	if started.Item.Type != string(jsonRpc.Reasoning) || started.Item.ID != itemID {
		return nil, false, nil
	}

	events, err := r.store.TurnEventList(ctx, db.TurnEventQuery{
		RunTurnIDs: []string{runTurnID},
		Methods: []string{
			string(codex.ItemReasoningSummaryTextDelta),
			string(codex.ItemReasoningTextDelta),
		},
		Limit: -1,
	})
	if err != nil {
		return nil, false, fmt.Errorf("读取 reasoning delta 失败: %w", err)
	}

	summaryByIndex := make(map[int]string)
	var contentParts []string
	for _, event := range events {
		var delta jsonRpc.TextDeltaEvent
		if json.Unmarshal([]byte(event.RawData), &delta) != nil || delta.ItemID != itemID {
			continue
		}
		switch event.Method {
		case string(codex.ItemReasoningSummaryTextDelta):
			summaryByIndex[delta.SummaryIndex] += delta.Delta
		case string(codex.ItemReasoningTextDelta):
			contentParts = append(contentParts, delta.Delta)
		}
	}

	maxSummaryIndex := -1
	for index := range summaryByIndex {
		if index > maxSummaryIndex {
			maxSummaryIndex = index
		}
	}
	summary := make([]string, maxSummaryIndex+1)
	for index, text := range summaryByIndex {
		summary[index] = text
	}
	content := make([]string, 0, 1)
	if text := strings.Join(contentParts, ""); text != "" {
		content = append(content, text)
	}

	normalized, err := json.Marshal(map[string]any{
		"id":            itemID,
		"type":          string(jsonRpc.Reasoning),
		"summary":       summary,
		"content":       content,
		"startedAtMs":   started.Item.StartedAtMs,
		"completedAtMs": r.now().UnixMilli(),
	})
	if err != nil {
		return nil, false, err
	}
	return normalized, true, nil
}

func (j *threadReconcileJob) recalcNext() {
	var next time.Time
	for _, state := range j.turns {
		if next.IsZero() || state.next.Before(next) {
			next = state.next
		}
	}
	j.next = next
}

func reconcileBackoffAt(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if attempt >= len(reconcileBackoff) {
		return reconcileBackoff[len(reconcileBackoff)-1]
	}
	return reconcileBackoff[attempt]
}

func stableLifecycleEventID(turnID, itemID, method string) string {
	sum := sha256.Sum256([]byte(method + "\x00" + turnID + "\x00" + itemID))
	return "evt:" + hex.EncodeToString(sum[:16])
}

func openLifecycleItems(events []db.TurnEvent) map[string]db.TurnEvent {
	started := make(map[string]db.TurnEvent)
	completed := make(map[string]struct{})
	for _, event := range events {
		itemID := eventItemID(event.RawData)
		if itemID == "" {
			continue
		}
		switch event.Method {
		case "item/started":
			started[itemID] = event
		case "item/completed":
			completed[itemID] = struct{}{}
		}
	}
	for itemID := range completed {
		delete(started, itemID)
	}
	return started
}

func eventItemID(raw string) string {
	var payload struct {
		Item struct {
			ID string `json:"id"`
		} `json:"item"`
	}
	if json.Unmarshal([]byte(raw), &payload) != nil {
		return ""
	}
	return payload.Item.ID
}

func findThreadItem(turn *jsonRpc.Turn, itemID string) (json.RawMessage, bool) {
	if turn == nil {
		return nil, false
	}
	for _, raw := range turn.Items {
		var payload struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &payload) != nil {
			continue
		}
		if payload.ID == itemID {
			return raw, true
		}
	}
	return nil, false
}

func finalItemPayload(raw json.RawMessage, turnStatus string) (bool, json.RawMessage, error) {
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false, nil, err
	}
	// agentMessage / reasoning 等 item 没有 status，出现在已存储历史里即视为终态。
	if payload.Status != "inProgress" {
		return true, raw, nil
	}
	if turnStatus != "failed" && turnStatus != "interrupted" {
		return false, raw, nil
	}

	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return false, nil, err
	}
	object["status"] = "failed"
	normalized, err := json.Marshal(object)
	if err != nil {
		return false, nil, err
	}
	return true, normalized, nil
}

func completionEventRaw(threadID, turnID string, item json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"item":     item,
		"threadId": threadID,
		"turnId":   turnID,
	})
}
