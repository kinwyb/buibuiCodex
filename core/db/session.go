package db

import (
	"context"
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
}

type SessionTurn struct {
	ReqID              string     `json:"req_id" gorm:"uniqueIndex;type:varchar(128)"`
	Seq                uint       `json:"seq" gorm:"primaryKey;autoIncrement"`
	TurnID             string     `json:"turn_id" gorm:"index;type:varchar(128)"`
	ThreadID           string     `json:"thread_id" gorm:"index;type:varchar(128)"`
	SessionID          string     `json:"session_id" gorm:"index;type:varchar(128)"`
	Agent              string     `json:"agent" gorm:"index;type:varchar(64)"`
	Question           string     `json:"question" gorm:"type:text"`         // 用户问题
	Answer             string     `json:"answer" gorm:"type:text"`           // AI回答
	IsCompleted        bool       `json:"is_completed" gorm:"default:false"` // 是否已完整回答
	CreatedAt          time.Time  `json:"created_at" gorm:"index;not null"`  // 创建时间
	CompletedAt        time.Time  `json:"completed_at"`                      // 完成时间
	ModelContextWindow int64      `json:"modelContextWindow"`                // 模型最大上下文窗口
	TokenUsage         TokenUsage `json:"last" gorm:"embedded"`              // 本轮交互用量
}

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
	ID         string    `json:"id" gorm:"uniqueIndex;type:varchar(128)"`
	Seq        uint      `json:"seq" gorm:"primaryKey;autoIncrement"`
	ThreadID   string    `json:"thread_id" gorm:"index;type:varchar(128)"` //threadID
	TurnID     string    `json:"turn_id" gorm:"index;type:varchar(128)"`   //turnID
	Type       string    `json:"type" gorm:"index;type:varchar(64)"`
	Method     string    `json:"method" gorm:"index;type:varchar(64)"` //方法
	RawData    string    `json:"raw_data" gorm:"type:JSON"`
	CreateTime time.Time `json:"create_time" gorm:"index;not null"`
}

type ISession interface {
	Init()
	// SessionSave session 保存
	SessionSave(ctx context.Context, session *Session) error
	// SessionQueryByID session 查询
	SessionQueryByID(ctx context.Context, sessionID string) *Session
	// ThreadSave session codex thread 保存
	ThreadSave(ctx context.Context, thread *SessionThread) error
	// LastThread 查询session对应的最后thread
	LastThread(ctx context.Context, sessionID string, agent string) *SessionThread
	// RequestSave 请求保存
	RequestSave(ctx context.Context, reqID string, quest string, agent string) error
	// TurnSave 会话信息保存
	TurnSave(ctx context.Context, turn *SessionTurn) error
	// TurnQueryByID 会话信息查询
	TurnQueryByID(ctx context.Context, turnID string) *SessionTurn
	// TurnEventSave 会话事件保存
	TurnEventSave(ctx context.Context, turn *TurnEvent) error
	// TurnTokenUsage 根据turnID更新token用量
	TurnTokenUsage(ctx context.Context, turnID string, tokenUsage TokenUsage) error
	// ThreadTokenUsage 根据ThreadID更新token用量
	ThreadTokenUsage(ctx context.Context, threadID string, tokenUsage TokenUsage) error
}

type sessionDefautStorage struct{}

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

func (s *sessionDefautStorage) LastThread(ctx context.Context, sessionID string, agent string) *SessionThread {
	return nil
}

func (s *sessionDefautStorage) RequestSave(ctx context.Context, reqID string, quest string, agent string) error {
	return nil
}

func (s *sessionDefautStorage) TurnSave(ctx context.Context, turn *SessionTurn) error {
	return nil
}

func (s *sessionDefautStorage) TurnQueryByID(ctx context.Context, turnID string) *SessionTurn {
	return nil
}

func (s *sessionDefautStorage) TurnEventSave(ctx context.Context, turn *TurnEvent) error {
	return nil
}

func (s *sessionDefautStorage) TurnTokenUsage(ctx context.Context, turnID string, tokenUsage TokenUsage) error {
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

func (s *sessionStorage) LastThread(ctx context.Context, sessionID string, agent string) *SessionThread {
	var thread = &SessionThread{}
	// 查询最近1小时内的thread
	err := s.db.WithContext(ctx).Model(thread).
		Where(" session_id = ? AND agent = ? ", sessionID, agent).
		Order("last_update desc").First(thread).Error
	if err != nil {
		return nil
	}
	return thread
}

func (s *sessionStorage) RequestSave(ctx context.Context, reqID string, quest string, agent string) error {
	req := &SessionTurn{
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

func (s *sessionStorage) TurnEventSave(ctx context.Context, turn *TurnEvent) error {
	if turn.CreateTime.IsZero() {
		turn.CreateTime = time.Now()
	}
	return s.db.WithContext(ctx).Save(turn).Error
}

func (s *sessionStorage) TurnTokenUsage(ctx context.Context, turnID string, tokenUsage TokenUsage) error {
	return s.db.WithContext(ctx).Where("turn_id = ?", turnID).Updates(&SessionTurn{TokenUsage: tokenUsage}).Error
}

func (s *sessionStorage) ThreadTokenUsage(ctx context.Context, threadID string, tokenUsage TokenUsage) error {
	return s.db.WithContext(ctx).Where("thread_id = ?", threadID).Updates(&SessionThread{TokenUsage: tokenUsage}).Error
}
