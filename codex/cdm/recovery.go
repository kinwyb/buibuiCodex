package cdm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
)

const (
	// recoveryPrompt 与 Codex 自身的 daemon recovery 提示保持一致：
	// 要求模型先检查当前状态，避免把已经完成的工具操作再执行一遍。
	recoveryPrompt = "The server restarted and interrupted the previous turn. " +
		"Continue the unfinished work from the saved conversation. " +
		"Check the current state before repeating actions that may already have completed."

	defaultRecoveryScanAge     = 7 * 24 * time.Hour
	defaultRecoveryMaxAttempts = 3
)

// turnRecovery 与 turnReconciler 的职责严格分开：
//   - reconciler 只修复某一次 run 缺失的 TurnEvent；
//   - recovery 只决定是否启动下一次 recovery continuation turn。
//
// 两者都通过 RunTurnID 定位实际运行，因此旧 run 的 interrupted 状态不会
// 被误用于新 run，事件补全完成也不会被误判成对话执行完成。
type turnRecovery struct {
	agent  *Agent
	store  db.ISession
	notify func(*types.Event)

	// resumeThread/reconcileRun 做成函数依赖，便于单测在不启动 Docker 和
	// app-server 的情况下覆盖“已完成/失败/中断”三种恢复决策。
	resumeThread func(context.Context, string) (*codex.Thread, *jsonRpc.Thread, error)
	reconcileRun func(context.Context, string, string, string) (bool, error)
	continueTurn func(db.SessionTurn, *codex.Thread, int)

	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	running   bool
	wg        sync.WaitGroup
	sessions  map[string]chan struct{}
	ready     chan struct{}
	readyDone bool

	scanAge     time.Duration
	maxAttempts int
	now         func() time.Time
}

func newTurnRecovery(agent *Agent, store db.ISession) *turnRecovery {
	r := &turnRecovery{
		agent:       agent,
		store:       store,
		sessions:    make(map[string]chan struct{}),
		scanAge:     defaultRecoveryScanAge,
		maxAttempts: defaultRecoveryMaxAttempts,
		now:         time.Now,
	}
	r.resumeThread = func(ctx context.Context, threadID string) (*codex.Thread, *jsonRpc.Thread, error) {
		thread := agent.client.Thread()
		thread.SubUnknowTurnEvent(agent.unknowThreadEventHandler)
		resumed, err := thread.ResumeWithThread(ctx, threadID)
		return thread, resumed, err
	}
	r.reconcileRun = func(ctx context.Context, threadID, logicalTurnID, runTurnID string) (bool, error) {
		return agent.reconciler.reconcileRun(ctx, threadID, logicalTurnID, runTurnID)
	}
	r.continueTurn = r.startContinuation
	return r
}

func (r *turnRecovery) SetNotifier(notify func(*types.Event)) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.notify = notify
	r.mu.Unlock()
}

func (r *turnRecovery) Start() {
	if r == nil || r.agent == nil || r.store == nil {
		return
	}
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	r.running = true
	r.ready = make(chan struct{})
	r.readyDone = false
	r.wg.Add(1)
	r.mu.Unlock()
	go r.loop()
}

// BeginStop 停止扫描新任务，但不会取消已经启动的 continuation。
// 已经启动的 run 由 Agent 的 activeTurns 计数和 stopSignal 统一收口。
func (r *turnRecovery) BeginStop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.running = false
	cancel := r.cancel
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (r *turnRecovery) Wait() {
	if r == nil {
		return
	}
	r.wg.Wait()
}

func (r *turnRecovery) Stop() {
	r.BeginStop()
	r.Wait()
}

// WaitSession 让普通 Prompt 等到同一 session 的恢复完成后继续，
// 避免恢复 turn 和用户新 turn 同时争用同一个 thread。
func (r *turnRecovery) WaitSession(ctx context.Context, sessionID string) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	ready := r.ready
	r.mu.Unlock()
	if ready != nil {
		select {
		case <-ready:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	r.mu.Lock()
	ch := r.sessions[sessionID]
	r.mu.Unlock()
	if ch == nil {
		return nil
	}
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *turnRecovery) markSession(sessionID string) chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ch := r.sessions[sessionID]; ch != nil {
		return ch
	}
	ch := make(chan struct{})
	r.sessions[sessionID] = ch
	return ch
}

func (r *turnRecovery) clearSession(sessionID string, ch chan struct{}) {
	if ch == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sessions[sessionID] == ch {
		delete(r.sessions, sessionID)
		close(ch)
	}
}

func (r *turnRecovery) loop() {
	defer r.wg.Done()
	defer r.markReady()
	since := r.now().Add(-r.scanAge)
	rows, err := r.store.TurnListRecoverableByAgent(r.ctx, r.agent.AgentID(), since, 200)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("扫描待恢复对话失败", "error", err.Error())
		}
		return
	}

	// 同一 thread 可能因为多次崩溃留下多条 running 记录。只恢复最新一条，
	// 旧记录标记为 superseded，避免对同一 thread 并发启动多个 turn。
	byThread := make(map[string]db.SessionTurn)
	order := make([]string, 0)
	for _, row := range rows {
		if old, ok := byThread[row.ThreadID]; ok {
			if old.Seq > row.Seq {
				continue
			}
			if err := r.store.TurnRecoveryUpdate(context.Background(), old.ReqID, "", db.RecoveryStatusFailed, "superseded by newer turn", old.RecoveryAttempt); err != nil {
				slog.Warn("标记旧恢复记录失败", "reqID", old.ReqID, "error", err.Error())
			}
		} else {
			order = append(order, row.ThreadID)
		}
		byThread[row.ThreadID] = row
	}

	// 先把所有待恢复 session 标记出来，再逐个执行。这样新 Prompt 在
	// WaitSession 里能看到全部恢复占用，不会因为前一个恢复耗时较长而抢跑。
	tasks := make([]db.SessionTurn, 0, len(order))
	for _, threadID := range order {
		row := byThread[threadID]
		r.markSession(row.SessionID)
		tasks = append(tasks, row)
	}
	r.markReady()

	for _, row := range tasks {
		if r.ctx == nil || r.ctx.Err() != nil {
			return
		}
		r.recoverOne(row)
	}
}

func (r *turnRecovery) markReady() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ready != nil && !r.readyDone {
		r.readyDone = true
		close(r.ready)
	}
}

func (r *turnRecovery) recoverOne(row db.SessionTurn) {
	r.mu.Lock()
	ch := r.sessions[row.SessionID]
	r.mu.Unlock()
	defer r.clearSession(row.SessionID, ch)

	if row.RecoveryAttempt >= r.maxAttempts {
		_ = r.store.TurnRecoveryUpdate(context.Background(), row.ReqID, row.RunTurnID, db.RecoveryStatusFailed,
			fmt.Sprintf("recovery attempts exceeded: %d", row.RecoveryAttempt), row.RecoveryAttempt)
		return
	}
	// 恢复 continuation 也计入 Agent 的活跃任务，Stop 才会等待或中断它。
	if err := r.agent.beginTurn(); err != nil {
		return
	}
	defer r.agent.endTurn()

	attempt := row.RecoveryAttempt + 1
	if err := r.store.TurnRecoveryUpdate(context.Background(), row.ReqID, row.RunTurnID, db.RecoveryStatusRecovering, "", attempt); err != nil {
		slog.Error("标记恢复中失败", "reqID", row.ReqID, "error", err.Error())
	}

	resumeCtx, cancel := r.operationContext(30 * time.Second)
	thread, resumed, err := r.resumeThread(resumeCtx, row.ThreadID)
	cancel()
	if err != nil {
		r.markRecoveryError(row, db.RecoveryStatusInterrupted, fmt.Errorf("恢复 thread 失败: %w", err), attempt)
		return
	}

	runTurnID := row.RunTurnID
	if runTurnID == "" {
		runTurnID = row.TurnID
	}
	lastTurn := findTurn(resumed.Turns, runTurnID)
	if lastTurn == nil {
		r.markRecoveryError(row, db.RecoveryStatusInterrupted, fmt.Errorf("app-server 中未找到 turn %s", runTurnID), attempt)
		return
	}
	logicalTurnID := row.TurnID
	if logicalTurnID == "" {
		logicalTurnID = runTurnID
	}

	switch lastTurn.Status {
	case "completed":
		r.finalizeWithoutContinuation(row, logicalTurnID, runTurnID, lastTurn)
		return
	case "failed":
		if _, err := r.reconcileRun(context.Background(), row.ThreadID, logicalTurnID, runTurnID); err != nil {
			slog.Warn("补全失败 run 事件失败", "threadID", row.ThreadID, "runTurnID", runTurnID, "error", err.Error())
		}
		r.markRecoveryError(row, db.RecoveryStatusFailed, turnError(lastTurn), attempt)
		return
	case "inProgress":
		if resumed.Status.Type == "active" {
			// 另一个客户端仍在执行该 thread，不能补失败，也不能再启动 turn。
			slog.Warn("thread 仍在运行，跳过自动恢复", "threadID", row.ThreadID, "runTurnID", runTurnID)
			_ = r.store.TurnRecoveryUpdate(context.Background(), row.ReqID, runTurnID, db.RecoveryStatusRecovering,
				"thread is still active", attempt)
			return
		}
	}

	// interrupted 或重启后残留的 inProgress：先按一次实际 run 收敛旧事件，
	// 再启动 continuation。补录和恢复是先后关系，不会同时作用在新 run 上。
	if _, err := r.reconcileRun(context.Background(), row.ThreadID, logicalTurnID, runTurnID); err != nil {
		slog.Warn("恢复前补全旧 run 事件失败", "threadID", row.ThreadID, "runTurnID", runTurnID, "error", err.Error())
	}
	r.continueTurn(row, thread, attempt)
}

func (r *turnRecovery) finalizeWithoutContinuation(row db.SessionTurn, logicalTurnID, runTurnID string, turn *jsonRpc.Turn) {
	if _, err := r.reconcileRun(context.Background(), row.ThreadID, logicalTurnID, runTurnID); err != nil {
		slog.Warn("补全已完成 turn 事件失败", "threadID", row.ThreadID, "runTurnID", runTurnID, "error", err.Error())
	}
	completedAt := time.Now()
	if err := r.store.TurnSave(context.Background(), &db.SessionTurn{
		ReqID:       row.ReqID,
		RunTurnID:   runTurnID,
		Answer:      answerFromTurn(turn),
		IsCompleted: true,
		CompletedAt: &completedAt,
	}); err != nil {
		slog.Warn("补写恢复完成状态失败", "reqID", row.ReqID, "error", err.Error())
	}
	_ = r.store.TurnRecoveryUpdate(context.Background(), row.ReqID, runTurnID, db.RecoveryStatusCompleted, "", row.RecoveryAttempt)
}

func (r *turnRecovery) startContinuation(row db.SessionTurn, thread *codex.Thread, attempt int) {
	logicalTurnID := row.TurnID
	if logicalTurnID == "" {
		logicalTurnID = row.RunTurnID
	}
	state := r.buildState(row)
	turn, err := thread.Turn()
	if err != nil {
		r.markRecoveryError(row, db.RecoveryStatusInterrupted, fmt.Errorf("构建恢复 turn 失败: %w", err), attempt)
		return
	}
	completed := make(chan error, 1)
	turn.RegisterEventHandler(r.agent.turnEvent(state, completed))
	opts := []codex.TurnStartOptions{
		codex.TurnStartWithSandboxPolicy(jsonRpc.SandboxPolicy{
			Type:          jsonRpc.SandboxPolicyTypeDangerFullAccess,
			NetworkAccess: true,
		}),
		codex.TurnStartWithApprovalPolicy(jsonRpc.Never),
		// TurnID 是逻辑对话 ID，RunTurnID 是本次实际运行 ID。
		// 这样恢复后的新 turn 仍然归到原提问下，前端回放无需知道 RunTurnID。
		codex.TurnStartWithRecoveryLink(logicalTurnID, logicalTurnID),
	}
	inputs := []jsonRpc.InputItem{{Type: "text", Text: recoveryPrompt}}
	startCtx, cancel := r.operationContext(30 * time.Second)
	err = turn.Start(startCtx, inputs, opts...)
	cancel()
	if err != nil {
		r.markRecoveryError(row, db.RecoveryStatusInterrupted, fmt.Errorf("启动恢复 turn 失败: %w", err), attempt)
		return
	}

	runTurnID := turn.TurnID()
	if logicalTurnID == "" {
		logicalTurnID = turn.RootTurnID()
	}
	r.agent.epMu.Lock()
	ct := r.agent.eps[row.SessionID]
	if ct == nil {
		ct = &codexTurn{
			threadID:       row.ThreadID,
			startTime:      time.Now().Unix(),
			lastUpdateTime: time.Now().Unix(),
			thread:         thread,
		}
		r.agent.eps[row.SessionID] = ct
	}
	ct.turn = turn
	ct.state = state
	r.agent.turnMap[runTurnID] = row.SessionID
	r.agent.turnRootMap[runTurnID] = logicalTurnID
	r.agent.epMu.Unlock()

	if err := r.store.TurnRecoveryUpdate(context.Background(), row.ReqID, runTurnID, db.RecoveryStatusRecovering, "", attempt); err != nil {
		slog.Warn("更新恢复运行 ID 失败", "reqID", row.ReqID, "error", err.Error())
	}

	var runErr error
	select {
	case runErr = <-completed:
	case <-r.agent.stopSignal.Done():
		runErr = errAgentStopping
	}
	close(completed)

	r.agent.epMu.Lock()
	if ct := r.agent.eps[row.SessionID]; ct != nil && ct.turn == turn {
		ct.turn = nil
		ct.lastUpdateTime = time.Now().Unix()
	}
	r.agent.epMu.Unlock()

	if runErr == nil {
		completedAt := time.Now()
		if err := r.store.TurnSave(context.Background(), &db.SessionTurn{
			ReqID:       row.ReqID,
			RunTurnID:   runTurnID,
			Answer:      state.LastMessage(),
			IsCompleted: true,
			CompletedAt: &completedAt,
		}); err != nil {
			slog.Warn("保存恢复结果失败", "reqID", row.ReqID, "error", err.Error())
		}
		_ = r.store.TurnRecoveryUpdate(context.Background(), row.ReqID, runTurnID, db.RecoveryStatusCompleted, "", attempt)
		return
	}

	status := db.RecoveryStatusFailed
	if errors.Is(runErr, errAgentStopping) {
		status = db.RecoveryStatusInterrupted
	}
	message := ""
	if runErr != nil {
		message = runErr.Error()
	}
	if err := r.store.TurnRecoveryUpdate(context.Background(), row.ReqID, runTurnID, status, message, attempt); err != nil {
		slog.Warn("更新恢复运行失败状态失败", "reqID", row.ReqID, "error", err.Error())
	}
	if !errors.Is(runErr, errAgentStopping) && state.EventHandler != nil {
		out := state.BuildEvent()
		out.Type = types.EventError
		out.Message = &types.Message{Role: types.RoleSystem, Content: "恢复执行失败：" + message}
		state.EventHandler(out)
	}
}

func (r *turnRecovery) buildState(row db.SessionTurn) *types.State {
	input := &types.InputMessage{
		ID:        row.ReqID,
		Session:   row.SessionID,
		Content:   row.Question,
		AgentName: r.agent.AgentID(),
		Timestamp: time.Now(),
	}
	if session := r.store.SessionQueryByID(context.Background(), row.SessionID); session != nil {
		input.Channel = session.Channel
		input.AccountID = session.AccountID
		input.ChatID = session.ChatID
		input.SenderID = session.UserID
	}
	state := types.NewState(input)
	r.mu.Lock()
	notify := r.notify
	r.mu.Unlock()
	if notify != nil {
		state.EventHandler = notify
	}
	return state
}

func (r *turnRecovery) markRecoveryError(row db.SessionTurn, status string, err error, attempt int) {
	message := ""
	if err != nil {
		message = err.Error()
	}
	if updateErr := r.store.TurnRecoveryUpdate(context.Background(), row.ReqID, row.RunTurnID, status, message, attempt); updateErr != nil {
		slog.Warn("更新恢复失败状态失败", "reqID", row.ReqID, "error", updateErr.Error())
	}
}

// operationContext 在 Agent 进入强制停止时取消恢复侧的 app-server 调用，
// 避免 Stop 已经放弃等待后 recovery goroutine 还卡在 Resume/Start 上。
func (r *turnRecovery) operationContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	stop := context.AfterFunc(r.agent.stopSignal, cancel)
	return ctx, func() {
		stop()
		cancel()
	}
}

func findTurn(turns []jsonRpc.Turn, turnID string) *jsonRpc.Turn {
	for i := len(turns) - 1; i >= 0; i-- {
		if turns[i].TurnID == turnID {
			return &turns[i]
		}
	}
	return nil
}

func turnError(turn *jsonRpc.Turn) error {
	if turn != nil && turn.Error != nil && turn.Error.Message != "" {
		return errors.New(turn.Error.Message)
	}
	return errors.New("turn failed")
}

func answerFromTurn(turn *jsonRpc.Turn) string {
	if turn == nil {
		return ""
	}
	for i := len(turn.Items) - 1; i >= 0; i-- {
		var item struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(turn.Items[i], &item) == nil && item.Type == string(jsonRpc.AgentMessage) && item.Text != "" {
			return item.Text
		}
	}
	return ""
}
