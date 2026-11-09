package cdm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/db"
)

const (
	defaultReconcileInterval = 5 * time.Second
	defaultReconcileScanAge  = 7 * 24 * time.Hour
	defaultReconcileReadWait = 20 * time.Second
	defaultReconcileMaxAge   = 1 * time.Hour
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

type reconcileJob struct {
	key        string
	threadID   string
	turnID     string
	attempts   int
	next       time.Time
	deadline   time.Time
	processing bool
}

// turnReconciler 对账 turn_events 与 app-server 的完整 thread 历史。
// turn/completed 只说明模型回合结束，后台命令可能仍在运行，因此这里按退避策略持续复查。
type turnReconciler struct {
	store    db.ISession
	readTurn func(context.Context, string, string) (*jsonRpc.Turn, error)

	mu      sync.Mutex
	jobs    map[string]*reconcileJob
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	interval time.Duration
	scanAge  time.Duration
	readWait time.Duration
	maxAge   time.Duration
	now      func() time.Time
}

func newTurnReconciler(store db.ISession, readTurn func(context.Context, string, string) (*jsonRpc.Turn, error)) *turnReconciler {
	return &turnReconciler{
		store:    store,
		readTurn: readTurn,
		jobs:     make(map[string]*reconcileJob),
		interval: defaultReconcileInterval,
		scanAge:  defaultReconcileScanAge,
		readWait: defaultReconcileReadWait,
		maxAge:   defaultReconcileMaxAge,
		now:      time.Now,
	}
}

func (r *turnReconciler) Start() {
	if r == nil || r.store == nil || r.readTurn == nil {
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
}

func (r *turnReconciler) Enqueue(threadID, turnID string) {
	if r == nil || r.store == nil || r.readTurn == nil || threadID == "" || turnID == "" {
		return
	}
	now := r.now()
	key := reconcileKey(threadID, turnID)

	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running {
		return
	}
	if job := r.jobs[key]; job != nil {
		if job.next.After(now) {
			job.next = now
		}
		return
	}
	r.jobs[key] = &reconcileJob{
		key:      key,
		threadID: threadID,
		turnID:   turnID,
		next:     now,
		deadline: now.Add(r.maxAge),
	}
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
		r.Enqueue(turn.ThreadID, turn.TurnID)
	}
	if len(turns) > 0 {
		slog.Info("发现未完成 turn 事件，已加入补全队列", "count", len(turns))
	}
}

func (r *turnReconciler) processDue() {
	now := r.now()
	r.mu.Lock()
	due := make([]*reconcileJob, 0)
	for _, job := range r.jobs {
		if job.processing || job.next.After(now) {
			continue
		}
		job.processing = true
		due = append(due, job)
	}
	r.mu.Unlock()

	for _, job := range due {
		if r.ctx == nil || r.ctx.Err() != nil {
			return
		}
		done, err := r.reconcileTurn(r.ctx, job.threadID, job.turnID)
		r.finishJob(job, done, err)
	}
}

func (r *turnReconciler) finishJob(job *reconcileJob, done bool, err error) {
	now := r.now()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.jobs[job.key] != job {
		return
	}
	job.processing = false
	if done {
		delete(r.jobs, job.key)
		return
	}
	if err != nil {
		slog.Warn("turn 事件补全失败", "threadID", job.threadID, "turnID", job.turnID, "error", err.Error())
	}
	if now.After(job.deadline) {
		slog.Error("turn 事件补全超时，停止重试",
			"threadID", job.threadID, "turnID", job.turnID, "attempts", job.attempts+1)
		delete(r.jobs, job.key)
		return
	}
	job.attempts++
	job.next = now.Add(reconcileBackoffAt(job.attempts))
}

// reconcileTurn 补全一轮中缺少 item/completed 的终态 item。
// 返回值 done=false 表示还有 item 仍在运行或读取失败，需要稍后重试。
func (r *turnReconciler) reconcileTurn(ctx context.Context, threadID, turnID string) (bool, error) {
	events, err := r.store.TurnEventList(ctx, db.TurnEventQuery{
		TurnIDs: []string{turnID},
		Methods: []string{"item/started", "item/completed"},
		Limit:   -1,
	})
	if err != nil {
		return false, fmt.Errorf("读取 turn 事件失败: %w", err)
	}
	open := openItemIDs(events)
	if len(open) == 0 {
		return true, nil
	}

	readCtx, cancel := context.WithTimeout(ctx, r.readWait)
	defer cancel()
	turn, err := r.readTurn(readCtx, threadID, turnID)
	if err != nil {
		return false, fmt.Errorf("读取 app-server turn 失败: %w", err)
	}

	fixed := 0
	for itemID := range open {
		rawItem, ok := findThreadItem(turn, itemID)
		if !ok {
			continue
		}
		final, normalized, err := finalItemPayload(rawItem, turn.Status)
		if err != nil {
			slog.Warn("解析 app-server item 失败", "turnID", turnID, "itemID", itemID, "error", err.Error())
			continue
		}
		if !final {
			continue
		}
		rawData, err := completionEventRaw(threadID, turnID, normalized)
		if err != nil {
			return false, fmt.Errorf("构造 item/completed 事件失败: %w", err)
		}
		event := &db.TurnEvent{
			ID:         stableLifecycleEventID(turnID, itemID, "item/completed"),
			ThreadID:   threadID,
			TurnID:     turnID,
			Type:       "notification",
			Method:     "item/completed",
			RawData:    string(rawData),
			CreateTime: r.now(),
		}
		if err := r.store.TurnEventUpsert(ctx, event); err != nil {
			return false, fmt.Errorf("写入补全事件失败: %w", err)
		}
		fixed++
	}
	if fixed > 0 {
		slog.Info("已补全 turn 事件", "threadID", threadID, "turnID", turnID, "items", fixed, "remaining", len(open)-fixed)
	}
	return fixed == len(open), nil
}

func reconcileKey(threadID, turnID string) string {
	return threadID + "\x00" + turnID
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

func openItemIDs(events []db.TurnEvent) map[string]struct{} {
	started := make(map[string]struct{})
	completed := make(map[string]struct{})
	for _, event := range events {
		itemID := eventItemID(event.RawData)
		if itemID == "" {
			continue
		}
		switch event.Method {
		case "item/started":
			started[itemID] = struct{}{}
		case "item/completed":
			completed[itemID] = struct{}{}
		}
	}
	open := make(map[string]struct{})
	for itemID := range started {
		if _, ok := completed[itemID]; !ok {
			open[itemID] = struct{}{}
		}
	}
	return open
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
	if payload.Status == "" || payload.Status != "inProgress" {
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
