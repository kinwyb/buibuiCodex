package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Session struct {
	SessionID  string    `json:"session_id" gorm:"uniqueIndex;type:varchar(128)"`
	Seq        uint      `json:"seq" gorm:"primaryKey;autoIncrement"`
	Channel    string    `json:"channel" gorm:"index;type:varchar(64)"`    // 来源渠道
	AccountID  string    `json:"account_id" gorm:"index;type:varchar(64)"` // 账号ID（用于多账号场景）
	ChatID     string    `json:"chat_id" gorm:"index;type:varchar(64)"`    // 聊天ID
	UserID     string    `json:"user_id" gorm:"index;type:varchar(64)"`    // 用户ID
	CreateTime time.Time `json:"create_time" gorm:"index;not null"`
}

type SessionThread struct {
	ThreadID   string     `json:"thread_id" gorm:"primaryKey;type:varchar(128)"`
	SessionID  string     `json:"session_id" gorm:"index;type:varchar(128)"`
	Agent      string     `json:"agent" gorm:"index;type:varchar(64)"`
	CreateTime time.Time  `json:"create_time" gorm:"index;not null"` //创建时间
	LastUpdate time.Time  `json:"last_update" gorm:"index;not null"` //最后更新时间
	TokenUsage TokenUsage `json:"total" gorm:"embedded"`             // 当前 Thread 累计用量
	IsDel      bool       `json:"isdel" gorm:"column:isdel;index;not null;default:false"`
}

type SessionTurn struct {
	ReqID string `json:"req_id" gorm:"uniqueIndex;type:varchar(128)"`
	Seq   uint   `json:"seq" gorm:"primaryKey;autoIncrement"`
	// TurnID 是一次用户对话的逻辑 ID。中断恢复只会新增 RunTurnID，
	// 不会改变 TurnID，因此前端历史、回放和消息归属都保持稳定。
	TurnID string `json:"turn_id" gorm:"index;type:varchar(128)"`
	// RunTurnID 是当前实际运行的 app-server turn ID。一次对话首次运行时
	// 与 TurnID 相同；如果进程中断后继续执行，这里会更新为新的 turn ID。
	RunTurnID   string `json:"run_turn_id" gorm:"index;type:varchar(128)"`
	ThreadID    string `json:"thread_id" gorm:"index;type:varchar(128)"`
	SessionID   string `json:"session_id" gorm:"index;type:varchar(128)"`
	Agent       string `json:"agent" gorm:"index;type:varchar(64)"`
	Question    string `json:"question" gorm:"type:text"`         // 用户问题
	Answer      string `json:"answer" gorm:"type:text"`           // AI回答
	IsCompleted bool   `json:"is_completed" gorm:"default:false"` // 是否已完整回答
	// RecoveryStatus 表示执行层的恢复状态，与 IsCompleted 不同：
	// 事件补录完成不代表对话已经恢复完成，只有 TurnID 对应的整个逻辑对话结束才算完成。
	RecoveryStatus     string     `json:"recovery_status" gorm:"index;type:varchar(32);default:''"`
	RecoveryAttempt    int        `json:"recovery_attempt" gorm:"default:0"`
	RecoveryError      string     `json:"recovery_error" gorm:"type:text"`
	CreatedAt          time.Time  `json:"created_at" gorm:"index;not null"` // 创建时间
	CompletedAt        *time.Time `json:"completed_at"`                     // 完成时间
	ModelContextWindow int64      `json:"modelContextWindow"`               // 模型最大上下文窗口
	TokenUsage         TokenUsage `json:"last" gorm:"embedded"`             // 本轮交互用量
}

const (
	// RecoveryStatusRunning 表示首次运行正在进行。进程异常退出后会停留在该状态，
	// 下次启动据此判断该逻辑对话需要恢复。
	RecoveryStatusRunning = "running"
	// RecoveryStatusInterrupted 表示服务主动停止或重启时终止了当前 run。
	RecoveryStatusInterrupted = "interrupted"
	// RecoveryStatusRecovering 表示已经启动 recovery continuation turn。
	RecoveryStatusRecovering = "recovering"
	// RecoveryStatusCompleted 表示执行层已经正常结束。
	RecoveryStatusCompleted = "completed"
	// RecoveryStatusFailed 表示恢复或执行失败；failed 不会被自动重跑。
	RecoveryStatusFailed = "failed"
)

// TokenUsage 单次或累计的详细 Token 拆分
type TokenUsage struct {
	TotalTokens           int64 `json:"totalTokens"`           // 总 Token 数
	InputTokens           int64 `json:"inputTokens"`           // 输入 Token 数
	CachedInputTokens     int64 `json:"cachedInputTokens"`     // 命中的缓存 Token
	CacheWriteInputTokens int64 `json:"cacheWriteInputTokens"` // 写入缓存的 Token
	OutputTokens          int64 `json:"outputTokens"`          // 输出 Token 数
	ReasoningOutputTokens int64 `json:"reasoningOutputTokens"` // 模型深度思考/推理消耗的 Token (o1/o3/R1等)
}

// TurnEvent 会话事件记录
type TurnEvent struct {
	ID       string `json:"id" gorm:"uniqueIndex;type:varchar(128)"`
	Seq      uint   `json:"seq" gorm:"primaryKey;autoIncrement"`
	ThreadID string `json:"thread_id" gorm:"index;type:varchar(128)"` //threadID
	// TurnID 是逻辑对话 ID，同一轮恢复多次时所有事件都归在同一个 TurnID 下，
	// 这样前端回放仍然只需按 TurnID 查询即可得到完整执行过程。
	TurnID string `json:"turn_id" gorm:"index;type:varchar(128)"`
	// RunTurnID 是实际产生该事件的 app-server turn ID。reconciler 通过它
	// 读取对应 run 的终态，避免把旧 run 的状态套到恢复中的新 run 上。
	RunTurnID  string    `json:"run_turn_id" gorm:"index;type:varchar(128)"`
	Type       string    `json:"type" gorm:"index;type:varchar(64)"`
	Method     string    `json:"method" gorm:"index;type:varchar(64)"` //方法
	RawData    string    `json:"raw_data" gorm:"type:JSON"`
	CreateTime time.Time `json:"create_time" gorm:"index;not null"`
}

// 轮次查询的分页参数
const (
	// DefaultTurnLimit 未指定 limit 时返回的轮次数量
	DefaultTurnLimit = 200
	// MaxTurnLimit 单次查询的最大轮次数量，limit 传负数表示尽量取全量
	MaxTurnLimit = 1000
)

// 事件查询的分页参数
const (
	// DefaultTurnEventLimit 未指定 limit 时返回的事件条数
	DefaultTurnEventLimit = 500
	// MaxTurnEventLimit 单次查询的最大事件条数。
	// turn_events 里逐 token 的 delta 也只是一行，一次拉全量容易把响应撑爆，
	// 因此上限比轮次查询高但仍有硬顶，超出后由调用方用 AfterSeq 续拉。
	MaxTurnEventLimit = 20000
)

// TurnQuery 轮次查询条件
type TurnQuery struct {
	// SessionID 会话标识，必填
	SessionID string
	// ThreadID 线程标识：指定后只查该线程下的轮次
	ThreadID string
	// BeforeSeq 只取 seq 小于该值的记录，用于向前翻页（加载更早的轮次）
	BeforeSeq uint
	// Since 只取创建时间不早于该时刻的记录，用于断线后的增量补发
	Since time.Time
	// Limit 返回条数：0 使用默认值，负数表示取全量（受 MaxTurnLimit 限制）
	Limit int
}

// NormalizeLimit 归一化 limit，返回实际生效的条数
func (q TurnQuery) NormalizeLimit() int {
	switch {
	case q.Limit < 0:
		return MaxTurnLimit
	case q.Limit == 0:
		return DefaultTurnLimit
	case q.Limit > MaxTurnLimit:
		return MaxTurnLimit
	default:
		return q.Limit
	}
}

// TurnEventQuery 会话事件查询条件。
// 事件表只按 thread/turn 归属，没有 session 维度，因此调用方需先拿到轮次再反查事件。
type TurnEventQuery struct {
	// TurnIDs 轮次标识列表，按 turn_id 批量取事件（一次 IN 查询，避免逐轮 N+1）
	TurnIDs []string
	// RunTurnIDs 按实际运行 ID 过滤，供 reconciler 精确处理某一次 run。
	RunTurnIDs []string
	// ThreadID 线程标识，与 TurnIDs 二选一
	ThreadID string
	// Methods 只取这些 method 的事件。
	// 生成过程中逐 token 的 delta 也会落库（item 完成即被清理），
	// 回放时传关键帧白名单可把行数压到千分之一。
	Methods []string
	// AfterSeq 只取 seq 大于该值的事件，用于向后翻页
	AfterSeq uint
	// Limit 返回条数：0 使用默认值，负数表示取全量（受 MaxTurnEventLimit 限制）
	Limit int
}

// TurnEventOpenTurn 表示仍有 item/started 缺少对应 item/completed 的轮次。
type TurnEventOpenTurn struct {
	ThreadID  string
	TurnID    string // 逻辑对话 ID
	RunTurnID string // 实际运行 ID
}

// NormalizeLimit 归一化 limit，返回实际生效的条数
func (q TurnEventQuery) NormalizeLimit() int {
	switch {
	case q.Limit < 0:
		return MaxTurnEventLimit
	case q.Limit == 0:
		return DefaultTurnEventLimit
	case q.Limit > MaxTurnEventLimit:
		return MaxTurnEventLimit
	default:
		return q.Limit
	}
}

// 对话列表（线程分页）参数
const (
	// DefaultThreadLimit 未指定 limit 时返回的线程数
	DefaultThreadLimit = 50
	// MaxThreadLimit 单次查询的最大线程数
	MaxThreadLimit = 200
)

// ThreadQuery 对话列表查询条件。
// 列表以 thread 为粒度：一个 thread 是一次完整对话，其下挂多轮 turn。
type ThreadQuery struct {
	// Channel 渠道，与 AccountID 一起定位用户，经 sessions 表关联过滤
	Channel string
	// AccountID 账号标识（ws 渠道下即 userID）
	AccountID string
	// SessionID 只查指定会话下的线程，用于单会话维度
	SessionID string
	// BeforeLastUpdate 复合游标之一：只取最后更新时间早于该时刻的线程
	BeforeLastUpdate time.Time
	// BeforeThreadID 复合游标之一：时间相同时用它续接，避免翻页漏项
	BeforeThreadID string
	// Limit 返回条数：0 使用默认值，负数表示取全量（受 MaxThreadLimit 限制）
	Limit int
}

// NormalizeLimit 归一化 limit，返回实际生效的条数
func (q ThreadQuery) NormalizeLimit() int {
	switch {
	case q.Limit < 0:
		return MaxThreadLimit
	case q.Limit == 0:
		return DefaultThreadLimit
	case q.Limit > MaxThreadLimit:
		return MaxThreadLimit
	default:
		return q.Limit
	}
}

// ThreadSummary 线程的轮次摘要，供对话列表直接展示。
// 列表页不查全量轮次，只取首轮提问（标题）与末轮内容（预览）。
type ThreadSummary struct {
	ThreadID string
	// Turns 该线程下的轮次数量
	Turns int64
	// FirstQuestion 首轮提问，用作对话标题
	FirstQuestion string
	// LastQuestion 末轮提问
	LastQuestion string
	// LastAnswer 末轮回答
	LastAnswer string
	// LastCompleted 末轮是否正常结束
	LastCompleted bool
}

type ISession interface {
	Init()
	// SessionSave session 保存
	SessionSave(ctx context.Context, session *Session) error
	// SessionQueryByID session 查询
	SessionQueryByID(ctx context.Context, sessionID string) *Session
	// ThreadQueryByID 线程信息查询
	ThreadQueryByID(ctx context.Context, sessionID string) *SessionThread
	// ThreadSave session codex thread 保存
	ThreadSave(ctx context.Context, thread *SessionThread) error
	// ThreadDelete 软删除指定会话下的线程，只标记 isdel，不物理删除历史数据
	ThreadDelete(ctx context.Context, sessionID string, threadID string) error
	// LastThread 查询session对应的最后thread
	LastThread(ctx context.Context, sessionID string, agent string) *SessionThread
	// RequestSave 请求保存
	RequestSave(ctx context.Context, sessionID string, reqID string, quest string, agent string) error
	// TurnSave 会话信息保存
	TurnSave(ctx context.Context, turn *SessionTurn) error
	// TurnQueryByID 会话信息查询
	TurnQueryByID(ctx context.Context, turnID string) *SessionTurn
	// TurnListRecoverableByAgent 查询最近需要恢复或补齐完成状态的逻辑对话。
	// 只扫描 running/interrupted/recovering/completed，避免把历史普通失败轮次全部重新执行。
	TurnListRecoverableByAgent(ctx context.Context, agent string, since time.Time, limit int) ([]SessionTurn, error)
	// TurnRecoveryUpdate 更新执行层恢复状态。runTurnID 为空时保留原值，
	// lastError 每次都会写入，便于清理旧错误；attempt 小于 0 时保留原重试次数。
	TurnRecoveryUpdate(ctx context.Context, reqID, runTurnID, status, lastError string, attempt int) error
	// ThreadList 查询会话下的所有线程，按创建时间升序返回，用于 session -> thread -> turn 的层级遍历
	ThreadList(ctx context.Context, sessionID string) ([]SessionThread, error)
	// ThreadPage 按用户分页查询线程，按最后更新时间倒序返回，用于对话列表
	ThreadPage(ctx context.Context, query ThreadQuery) ([]SessionThread, error)
	// ThreadSummaries 批量查询线程的轮次摘要（首轮提问 / 末轮内容 / 轮次数量）
	ThreadSummaries(ctx context.Context, threadIDs []string) (map[string]ThreadSummary, error)
	// TurnList 按会话/线程分页查询轮次，结果按 seq 升序返回
	TurnList(ctx context.Context, query TurnQuery) ([]SessionTurn, error)
	// TurnEventSave 会话事件保存
	TurnEventSave(ctx context.Context, turn *TurnEvent) error
	// TurnEventUpsert 会话事件幂等保存。ID 已存在时更新内容而不是报唯一键冲突，
	// 用于 item 生命周期事件和 reconciler 生成的补全事件共用稳定 ID。
	TurnEventUpsert(ctx context.Context, turn *TurnEvent) error
	// TurnEventList 按轮次/线程查询会话事件，结果按 seq 升序返回
	TurnEventList(ctx context.Context, query TurnEventQuery) ([]TurnEvent, error)
	// TurnEventOpenTurns 找出时间窗口内存在 started 但缺少 completed 的轮次。
	TurnEventOpenTurns(ctx context.Context, since time.Time, limit int) ([]TurnEventOpenTurn, error)
	// TurnEventDeleteByMethods 删除某轮下指定 method 的事件。
	// 用于在 item 结束后丢弃逐 token 的增量帧：完整内容已随 item/completed 落库，
	// 这些碎片不再参与回放，留着只会让 turn_events 无限膨胀。
	// turnID 或 methods 为空时不做任何事，避免误删整表。
	TurnEventDeleteByMethods(ctx context.Context, turnID string, methods []string) error
	// TurnEventDeleteByRun 删除某一次实际运行下指定 method 的事件。
	// 逻辑 TurnID 在恢复后会复用，因此 delta 清理必须按 RunTurnID 限定范围。
	TurnEventDeleteByRun(ctx context.Context, runTurnID string, methods []string) error
	// TurnTokenUsage 根据 turnID 更新本轮的 token 用量，
	// contextWindow 为模型上下文窗口（0 表示上游未提供，保留原值）
	TurnTokenUsage(ctx context.Context, turnID string, tokenUsage TokenUsage, contextWindow int64) error
	// ThreadTokenUsage 根据ThreadID更新token用量
	ThreadTokenUsage(ctx context.Context, threadID string, tokenUsage TokenUsage) error
}

type sessionDefautStorage struct{}

func (s *sessionDefautStorage) ThreadQueryByID(ctx context.Context, sessionID string) *SessionThread {
	return nil
}

func (s *sessionDefautStorage) SessionQueryByID(ctx context.Context, sessionID string) *Session {
	return nil
}

func (s *sessionDefautStorage) Init() {
}

func (s *sessionDefautStorage) SessionSave(ctx context.Context, session *Session) error {
	return nil
}

func (s *sessionDefautStorage) ThreadSave(ctx context.Context, thread *SessionThread) error {
	return nil
}

func (s *sessionDefautStorage) ThreadDelete(ctx context.Context, sessionID string, threadID string) error {
	return nil
}

func (s *sessionDefautStorage) LastThread(ctx context.Context, sessionID string, agent string) *SessionThread {
	return nil
}

func (s *sessionDefautStorage) RequestSave(ctx context.Context, sessionID string, reqID string, quest string, agent string) error {
	return nil
}

func (s *sessionDefautStorage) TurnSave(ctx context.Context, turn *SessionTurn) error {
	return nil
}

func (s *sessionDefautStorage) TurnQueryByID(ctx context.Context, turnID string) *SessionTurn {
	return nil
}

func (s *sessionDefautStorage) TurnListRecoverableByAgent(ctx context.Context, agent string, since time.Time, limit int) ([]SessionTurn, error) {
	return nil, nil
}

func (s *sessionDefautStorage) TurnRecoveryUpdate(ctx context.Context, reqID, runTurnID, status, lastError string, attempt int) error {
	return nil
}

func (s *sessionDefautStorage) TurnList(ctx context.Context, query TurnQuery) ([]SessionTurn, error) {
	return nil, nil
}

func (s *sessionDefautStorage) ThreadList(ctx context.Context, sessionID string) ([]SessionThread, error) {
	return nil, nil
}

func (s *sessionDefautStorage) ThreadPage(ctx context.Context, query ThreadQuery) ([]SessionThread, error) {
	return nil, nil
}

func (s *sessionDefautStorage) ThreadSummaries(ctx context.Context, threadIDs []string) (map[string]ThreadSummary, error) {
	return nil, nil
}

func (s *sessionDefautStorage) TurnEventSave(ctx context.Context, turn *TurnEvent) error {
	return nil
}

func (s *sessionDefautStorage) TurnEventUpsert(ctx context.Context, turn *TurnEvent) error {
	return nil
}

func (s *sessionDefautStorage) TurnEventList(ctx context.Context, query TurnEventQuery) ([]TurnEvent, error) {
	return nil, nil
}

func (s *sessionDefautStorage) TurnEventOpenTurns(ctx context.Context, since time.Time, limit int) ([]TurnEventOpenTurn, error) {
	return nil, nil
}

func (s *sessionDefautStorage) TurnEventDeleteByMethods(ctx context.Context, turnID string, methods []string) error {
	return nil
}

func (s *sessionDefautStorage) TurnEventDeleteByRun(ctx context.Context, runTurnID string, methods []string) error {
	return nil
}

func (s *sessionDefautStorage) TurnTokenUsage(ctx context.Context, turnID string, tokenUsage TokenUsage, contextWindow int64) error {
	return nil
}

func (s *sessionDefautStorage) ThreadTokenUsage(ctx context.Context, threadID string, tokenUsage TokenUsage) error {
	return nil
}

type sessionStorage struct {
	db *gorm.DB
}

func newSessionStorage(db *gorm.DB) *sessionStorage {
	return &sessionStorage{db: db}
}

func (s *sessionStorage) Init() {
	s.db.AutoMigrate(&Session{}, &SessionThread{}, &SessionTurn{}, &TurnEvent{})
	s.backfillRunTurnID()
}

// backfillRunTurnID 兼容升级前的数据。旧版本没有恢复执行，TurnID 同时就是
// 实际运行的 turn ID，因此只需把空的 RunTurnID 补成 TurnID，不能改写 TurnID。
// 只更新 NULL/空值，重复启动不会覆盖恢复流程写入的新 RunTurnID。
func (s *sessionStorage) backfillRunTurnID() {
	if err := s.db.Model(&SessionTurn{}).
		Where("run_turn_id IS NULL OR run_turn_id = ''").
		Update("run_turn_id", gorm.Expr("turn_id")).Error; err != nil {
		slog.Warn("回填 session_turns.run_turn_id 失败", "error", err.Error())
	}
	if err := s.db.Model(&TurnEvent{}).
		Where("run_turn_id IS NULL OR run_turn_id = ''").
		Update("run_turn_id", gorm.Expr("turn_id")).Error; err != nil {
		slog.Warn("回填 turn_events.run_turn_id 失败", "error", err.Error())
	}
}

func (s *sessionStorage) SessionSave(ctx context.Context, session *Session) error {
	if session.CreateTime.IsZero() {
		session.CreateTime = time.Now()
	}
	return s.db.WithContext(ctx).Where(" session_id = ? ", session.SessionID).FirstOrCreate(session).Error
}

func (s *sessionStorage) SessionQueryByID(ctx context.Context, sessionID string) *Session {
	var ret Session
	s.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&ret)
	if ret.SessionID == "" {
		return nil
	}
	return &ret
}

func (s *sessionStorage) ThreadQueryByID(ctx context.Context, threadID string) *SessionThread {
	var ret SessionThread
	s.db.WithContext(ctx).Where("thread_id = ? AND isdel = ?", threadID, false).First(&ret)
	if ret.ThreadID == "" {
		return nil
	}
	return &ret
}

func (s *sessionStorage) ThreadSave(ctx context.Context, thread *SessionThread) error {
	if thread.CreateTime.IsZero() {
		thread.CreateTime = time.Now()
	}
	if thread.LastUpdate.IsZero() {
		thread.LastUpdate = time.Now()
	}
	// 如果 thread_id 已存在，则仅更新 last_update 字段；不存在则执行 INSERT
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "thread_id"}},              // 冲突判重依据的列
		DoUpdates: clause.AssignmentColumns([]string{"last_update"}), // 发生冲突时需要更新的字段
	}).Create(&thread).Error
}

func (s *sessionStorage) ThreadDelete(ctx context.Context, sessionID string, threadID string) error {
	if sessionID == "" || threadID == "" {
		return errors.New("session_id and thread_id are required")
	}
	return s.db.WithContext(ctx).Model(&SessionThread{}).
		Where("session_id = ? AND thread_id = ?", sessionID, threadID).
		Update("isdel", true).Error
}

func (s *sessionStorage) LastThread(ctx context.Context, sessionID string, agent string) *SessionThread {
	var thread = &SessionThread{}
	// 查询最近1小时内的thread
	err := s.db.WithContext(ctx).Model(thread).
		Where(" session_id = ? AND agent = ? AND isdel = ? ", sessionID, agent, false).
		Order("last_update desc").First(thread).Error
	if err != nil {
		return nil
	}
	return thread
}

func (s *sessionStorage) RequestSave(ctx context.Context, sessionID string, reqID string, quest string, agent string) error {
	req := &SessionTurn{
		SessionID: sessionID,
		ReqID:     reqID,
		Question:  quest,
		CreatedAt: time.Now(),
		Agent:     agent,
	}
	return s.db.WithContext(ctx).Save(req).Error
}

func (s *sessionStorage) TurnSave(ctx context.Context, turn *SessionTurn) error {
	return s.db.WithContext(ctx).Where("req_id = ?", turn.ReqID).Updates(turn).Error
}

func (s *sessionStorage) TurnQueryByID(ctx context.Context, turnID string) *SessionTurn {
	var ret SessionTurn
	s.db.WithContext(ctx).Where("turn_id = ?", turnID).First(&ret)
	if ret.ReqID == "" {
		return nil
	}
	return &ret
}

// TurnListRecoverableByAgent 只返回运行中、主动中断、恢复中或已完成但可能
// 还没写回 IsCompleted 的对话。历史普通失败轮次的 RecoveryStatus 为空，
// 不能被自动重跑，否则升级后会把旧失败全部执行一遍。
func (s *sessionStorage) TurnListRecoverableByAgent(ctx context.Context, agent string, since time.Time, limit int) ([]SessionTurn, error) {
	if limit <= 0 {
		limit = 200
	} else if limit > MaxTurnLimit {
		limit = MaxTurnLimit
	}
	tx := s.db.WithContext(ctx).Model(&SessionTurn{}).
		Where("agent = ? AND is_completed = ?", agent, false).
		Where("recovery_status IN ?", []string{
			RecoveryStatusRunning,
			RecoveryStatusInterrupted,
			RecoveryStatusRecovering,
			RecoveryStatusCompleted,
		})
	if !since.IsZero() {
		tx = tx.Where("created_at >= ?", since)
	}
	threads := s.tableName("SessionThread")
	tx = tx.Where(
		fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s st WHERE st.thread_id = session_turns.thread_id AND st.isdel = ?)", threads),
		true,
	)

	var list []SessionTurn
	if err := tx.Order("seq asc").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *sessionStorage) TurnRecoveryUpdate(ctx context.Context, reqID, runTurnID, status, lastError string, attempt int) error {
	if reqID == "" {
		return errors.New("req_id is required")
	}
	updates := map[string]any{
		"recovery_status": status,
		"recovery_error":  lastError,
	}
	if attempt >= 0 {
		updates["recovery_attempt"] = attempt
	}
	if runTurnID != "" {
		updates["run_turn_id"] = runTurnID
	}
	return s.db.WithContext(ctx).Model(&SessionTurn{}).
		Where("req_id = ?", reqID).
		Updates(updates).Error
}

// ThreadList 查询会话下的所有线程，按创建时间升序返回
func (s *sessionStorage) ThreadList(ctx context.Context, sessionID string) ([]SessionThread, error) {
	var list []SessionThread
	err := s.db.WithContext(ctx).
		Where("session_id = ? AND isdel = ?", sessionID, false).
		Order("create_time asc").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ThreadPage 按用户分页查询线程，按最后更新时间倒序返回。
//
// 归属过滤走 sessions 表关联，而不是对 session_id 做 LIKE 前缀匹配：
// 会话键形如 ws_<userID>_<chatID>，其中的下划线在 LIKE 里是单字符通配符，
// 前缀匹配会误命中别的用户（ws_alice 与 wsXalice）。
func (s *sessionStorage) ThreadPage(ctx context.Context, query ThreadQuery) ([]SessionThread, error) {
	tx := s.db.WithContext(ctx).Model(&SessionThread{}).
		Select("session_threads.*").
		Where("session_threads.isdel = ?", false)

	if query.SessionID != "" {
		tx = tx.Where("session_threads.session_id = ?", query.SessionID)
	}
	if query.Channel != "" || query.AccountID != "" {
		sessions := s.tableName("Session")
		tx = tx.Joins(fmt.Sprintf("JOIN %s ON %s.session_id = session_threads.session_id", sessions, sessions))
		if query.Channel != "" {
			tx = tx.Where(sessions+".channel = ?", query.Channel)
		}
		if query.AccountID != "" {
			tx = tx.Where(sessions+".account_id = ?", query.AccountID)
		}
	}
	if !query.BeforeLastUpdate.IsZero() {
		// 复合游标：时间相同时用 thread_id 续接，避免同一时刻的线程被翻页跳过
		tx = tx.Where(
			"(session_threads.last_update < ? OR (session_threads.last_update = ? AND session_threads.thread_id < ?))",
			query.BeforeLastUpdate, query.BeforeLastUpdate, query.BeforeThreadID,
		)
	}

	var list []SessionThread
	err := tx.Order("session_threads.last_update desc").
		Order("session_threads.thread_id desc").
		Limit(query.NormalizeLimit()).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ThreadSummaries 批量查询线程的轮次摘要。
//
// 列表页只需要「首轮提问当标题 + 末轮内容当预览」，所以不拉全量轮次：
// 一条聚合拿轮次数，另两条按 MIN/MAX(seq) 取首末轮，整体与线程数无关。
func (s *sessionStorage) ThreadSummaries(ctx context.Context, threadIDs []string) (map[string]ThreadSummary, error) {
	out := make(map[string]ThreadSummary, len(threadIDs))
	if len(threadIDs) == 0 {
		return out, nil
	}

	type countRow struct {
		ThreadID string
		Turns    int64
	}
	var counts []countRow
	if err := s.db.WithContext(ctx).Model(&SessionTurn{}).
		Select("thread_id, COUNT(*) AS turns").
		Where("thread_id IN ?", threadIDs).
		Group("thread_id").
		Scan(&counts).Error; err != nil {
		return nil, err
	}
	for _, c := range counts {
		out[c.ThreadID] = ThreadSummary{ThreadID: c.ThreadID, Turns: c.Turns}
	}

	table := s.tableName("SessionTurn")
	type edgeRow struct {
		ThreadID    string
		Question    string
		Answer      string
		IsCompleted bool
	}
	// 首轮：取每个线程的最小 seq
	firstSQL := fmt.Sprintf(`SELECT t.thread_id, t.question, t.answer, t.is_completed
FROM %s t JOIN (SELECT thread_id, MIN(seq) AS edge FROM %s WHERE thread_id IN ? GROUP BY thread_id) m
ON t.thread_id = m.thread_id AND t.seq = m.edge`, table, table)
	var firsts []edgeRow
	if err := s.db.WithContext(ctx).Raw(firstSQL, threadIDs).Scan(&firsts).Error; err != nil {
		return nil, err
	}
	for _, e := range firsts {
		item := out[e.ThreadID]
		item.ThreadID = e.ThreadID
		item.FirstQuestion = e.Question
		out[e.ThreadID] = item
	}

	// 末轮：取每个线程的最大 seq
	lastSQL := fmt.Sprintf(`SELECT t.thread_id, t.question, t.answer, t.is_completed
FROM %s t JOIN (SELECT thread_id, MAX(seq) AS edge FROM %s WHERE thread_id IN ? GROUP BY thread_id) m
ON t.thread_id = m.thread_id AND t.seq = m.edge`, table, table)
	var lasts []edgeRow
	if err := s.db.WithContext(ctx).Raw(lastSQL, threadIDs).Scan(&lasts).Error; err != nil {
		return nil, err
	}
	for _, e := range lasts {
		item := out[e.ThreadID]
		item.ThreadID = e.ThreadID
		item.LastQuestion = e.Question
		item.LastAnswer = e.Answer
		item.LastCompleted = e.IsCompleted
		out[e.ThreadID] = item
	}

	return out, nil
}

// tableName 返回模型对应的实际表名，避免在关联查询里写死
func (s *sessionStorage) tableName(model string) string {
	if s.db.NamingStrategy == nil {
		return ""
	}
	return s.db.NamingStrategy.TableName(model)
}

// TurnList 按会话/线程分页查询轮次。
// 倒序取最近 N 条（便于翻页与 limit 语义），返回前反转为正序。
func (s *sessionStorage) TurnList(ctx context.Context, query TurnQuery) ([]SessionTurn, error) {
	if query.SessionID == "" && query.ThreadID == "" {
		return nil, errors.New("session_id or thread_id is required")
	}

	tx := s.db.WithContext(ctx).Model(&SessionTurn{})
	if query.SessionID != "" {
		tx = tx.Where("session_id = ?", query.SessionID)
	}
	if query.ThreadID != "" {
		tx = tx.Where("thread_id = ?", query.ThreadID)
	}
	if query.BeforeSeq > 0 {
		tx = tx.Where("seq < ?", query.BeforeSeq)
	}
	if !query.Since.IsZero() {
		tx = tx.Where("created_at >= ?", query.Since)
	}
	threads := s.tableName("SessionThread")
	tx = tx.Where(
		fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s st WHERE st.thread_id = session_turns.thread_id AND st.isdel = ?)", threads),
		true,
	)

	var list []SessionTurn
	if err := tx.Order("seq desc").Limit(query.NormalizeLimit()).Find(&list).Error; err != nil {
		return nil, err
	}
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list, nil
}

func (s *sessionStorage) TurnEventSave(ctx context.Context, turn *TurnEvent) error {
	if turn.CreateTime.IsZero() {
		turn.CreateTime = time.Now()
	}
	return s.db.WithContext(ctx).Save(turn).Error
}

func (s *sessionStorage) TurnEventUpsert(ctx context.Context, turn *TurnEvent) error {
	if turn.CreateTime.IsZero() {
		turn.CreateTime = time.Now()
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"thread_id", "turn_id", "run_turn_id", "type", "method", "raw_data", "create_time",
		}),
	}).Create(turn).Error
}

// TurnEventList 按轮次/线程查询会话事件，按 seq 升序返回。
// 升序是回放的硬要求：seq 是自增主键，同一轮内严格等于事件发生顺序。
func (s *sessionStorage) TurnEventList(ctx context.Context, query TurnEventQuery) ([]TurnEvent, error) {
	if len(query.TurnIDs) == 0 && len(query.RunTurnIDs) == 0 && query.ThreadID == "" {
		return nil, errors.New("turn_ids, run_turn_ids or thread_id is required")
	}

	tx := s.db.WithContext(ctx).Model(&TurnEvent{})
	if len(query.TurnIDs) > 0 {
		tx = tx.Where("turn_id IN ?", query.TurnIDs)
	}
	if len(query.RunTurnIDs) > 0 {
		// 升级前的数据没有 run_turn_id；当时逻辑 ID 和实际 ID 相同，
		// 因此回退到 turn_id 匹配，避免历史 open turn 从此无法补全。
		tx = tx.Where("(run_turn_id IN ? OR ((run_turn_id IS NULL OR run_turn_id = '') AND turn_id IN ?))",
			query.RunTurnIDs, query.RunTurnIDs)
	}
	if query.ThreadID != "" {
		tx = tx.Where("thread_id = ?", query.ThreadID)
	}
	if len(query.Methods) > 0 {
		tx = tx.Where("method IN ?", query.Methods)
	}
	if query.AfterSeq > 0 {
		tx = tx.Where("seq > ?", query.AfterSeq)
	}
	threads := s.tableName("SessionThread")
	tx = tx.Where(
		fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s st WHERE st.thread_id = turn_events.thread_id AND st.isdel = ?)", threads),
		true,
	)

	var list []TurnEvent
	if err := tx.Order("seq asc").Limit(query.NormalizeLimit()).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *sessionStorage) TurnEventDeleteByMethods(ctx context.Context, turnID string, methods []string) error {
	// 缺任一条都退化为无条件删除，因此这里必须显式挡住
	if turnID == "" || len(methods) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Where("turn_id = ? AND method IN ?", turnID, methods).
		Delete(&TurnEvent{}).Error
}

func (s *sessionStorage) TurnEventDeleteByRun(ctx context.Context, runTurnID string, methods []string) error {
	if runTurnID == "" || len(methods) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Where("run_turn_id = ? AND method IN ?", runTurnID, methods).
		Delete(&TurnEvent{}).Error
}

// TurnEventOpenTurns 扫描最近的事件窗口，按 thread/turn 聚合 item 生命周期。
// item id 在 raw_data 里，跨 MySQL/SQLite 做 JSON 判断不划算，因此拉少量候选行后在 Go 里聚合。
func (s *sessionStorage) TurnEventOpenTurns(ctx context.Context, since time.Time, limit int) ([]TurnEventOpenTurn, error) {
	if limit <= 0 {
		limit = 200
	} else if limit > MaxTurnLimit {
		limit = MaxTurnLimit
	}

	tx := s.db.WithContext(ctx).Model(&TurnEvent{}).
		Select("thread_id", "turn_id", "run_turn_id", "method", "raw_data", "seq").
		Where("method IN ?", []string{"item/started", "item/completed"})
	if !since.IsZero() {
		tx = tx.Where("create_time >= ?", since)
	}
	threads := s.tableName("SessionThread")
	tx = tx.Where(
		fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s st WHERE st.thread_id = turn_events.thread_id AND st.isdel = ?)", threads),
		true,
	)

	var events []TurnEvent
	if err := tx.Order("seq asc").Limit(MaxTurnEventLimit).Find(&events).Error; err != nil {
		return nil, err
	}

	type lifecycle struct {
		threadID  string
		turnID    string
		runTurnID string
		started   map[string]struct{}
		completed map[string]struct{}
	}
	items := make(map[string]*lifecycle)
	order := make([]string, 0)
	for _, event := range events {
		itemID := turnEventItemID(event.RawData)
		if itemID == "" {
			continue
		}
		runTurnID := event.RunTurnID
		if runTurnID == "" {
			// 兼容升级前的数据：旧事件没有 run_turn_id，当时逻辑 ID 和实际 ID 相同。
			runTurnID = event.TurnID
		}
		key := event.ThreadID + "\x00" + event.TurnID + "\x00" + runTurnID
		entry := items[key]
		if entry == nil {
			entry = &lifecycle{
				threadID:  event.ThreadID,
				turnID:    event.TurnID,
				runTurnID: runTurnID,
				started:   make(map[string]struct{}),
				completed: make(map[string]struct{}),
			}
			items[key] = entry
			order = append(order, key)
		}
		if event.Method == "item/started" {
			entry.started[itemID] = struct{}{}
		} else {
			entry.completed[itemID] = struct{}{}
		}
	}

	result := make([]TurnEventOpenTurn, 0)
	for _, key := range order {
		entry := items[key]
		open := false
		for itemID := range entry.started {
			if _, ok := entry.completed[itemID]; !ok {
				open = true
				break
			}
		}
		if !open {
			continue
		}
		result = append(result, TurnEventOpenTurn{
			ThreadID:  entry.threadID,
			TurnID:    entry.turnID,
			RunTurnID: entry.runTurnID,
		})
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func turnEventItemID(raw string) string {
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

func (s *sessionStorage) TurnTokenUsage(ctx context.Context, turnID string, tokenUsage TokenUsage, contextWindow int64) error {
	// contextWindow 为 0 时 gorm 的 Updates(struct) 会忽略该字段，保留库里已有的值
	// 恢复后的 Token 事件携带实际 RunTurnID，因此两个 ID 都要匹配。
	return s.db.WithContext(ctx).Where("turn_id = ? OR run_turn_id = ?", turnID, turnID).
		Updates(&SessionTurn{TokenUsage: tokenUsage, ModelContextWindow: contextWindow}).Error
}

func (s *sessionStorage) ThreadTokenUsage(ctx context.Context, threadID string, tokenUsage TokenUsage) error {
	return s.db.WithContext(ctx).Where("thread_id = ?", threadID).Updates(&SessionThread{TokenUsage: tokenUsage}).Error
}
