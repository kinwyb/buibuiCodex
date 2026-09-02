package db

import (
	"context"
	"uuid"

	"gorm.io/gorm"
)

type Task struct {
	ID          string   `json:"task_id" gorm:"primaryKey;type:varchar(128)"` // 任务唯一ID
	Name        string   `json:"name" gorm:"type:varchar(128)"`               // 任务名称
	Description string   `json:"description" gorm:"type:varchar(128)"`        // 任务描述
	Enabled     bool     `json:"enabled" gorm:"default:true"`                 // 是否启用
	OneTime     bool     `json:"one_time" gorm:"default:false"`               // 是否为一次性任务
	Cron        string   `json:"cron" gorm:"type:varchar(30)"`                // 定时配置
	SessionID   string   `json:"session_id" gorm:"index;type:varchar(128)"`   //session_id
	Content     string   `json:"content" gorm:"type:text"`                    // 任务内容,发送给ai的内容
	Session     *Session `json:"-" gorm:"-"`                                  //session信息
}

// ITask 任务状态存储接口
type ITask interface {
	Init()
	// Load 加载所有任务状态
	Load(ctx context.Context) (map[string]*Task, error)
	// TaskQueryBySessionID 根据session_id查询任务
	TaskQueryBySessionID(ctx context.Context, sessionID string) ([]*Task, error)
	// Save 保存单个任务状态
	Save(ctx context.Context, task *Task) error
	// Delete 删除单个任务状态
	Delete(ctx context.Context, taskID string) error
	// QueryByID 根据ID查询task任务
	QueryByID(ctx context.Context, taskID string) (*Task, error)
}

type taskDefaultStorage struct{}

func (t *taskDefaultStorage) Init() {
}

func (t *taskDefaultStorage) Load(ctx context.Context) (map[string]*Task, error) {
	return nil, nil
}

func (t *taskDefaultStorage) TaskQueryBySessionID(ctx context.Context, sessionID string) ([]*Task, error) {
	return nil, nil
}

func (t *taskDefaultStorage) Save(ctx context.Context, task *Task) error {
	return nil
}

func (t *taskDefaultStorage) Delete(ctx context.Context, taskID string) error {
	return nil
}

func (t *taskDefaultStorage) QueryByID(ctx context.Context, taskID string) (*Task, error) {
	return nil, nil
}

type taskStorage struct {
	db *gorm.DB
}

func newTaskStorage(db *gorm.DB) *taskStorage {
	return &taskStorage{
		db: db,
	}
}

func (t *taskStorage) Init() {
	t.db.AutoMigrate(&Task{})
}

func (t *taskStorage) Load(ctx context.Context) (map[string]*Task, error) {
	var tasks []*Task
	err := t.db.WithContext(ctx).Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	ret := make(map[string]*Task)
	for _, task := range tasks {
		ret[task.ID] = task
	}
	return ret, nil
}

func (t *taskStorage) TaskQueryBySessionID(ctx context.Context, sessionID string) ([]*Task, error) {
	var tasks []*Task
	err := t.db.WithContext(ctx).Where(" session_id = ? ", sessionID).Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (t *taskStorage) Save(ctx context.Context, task *Task) error {
	if task.ID == "" {
		task.ID = uuid.NewV4().String()
	}
	return t.db.WithContext(ctx).Create(task).Error
}

func (t *taskStorage) Delete(ctx context.Context, taskID string) error {
	if taskID == "" {
		return nil
	}
	return t.db.WithContext(ctx).Delete(&Task{ID: taskID}).Error
}

func (t *taskStorage) QueryByID(ctx context.Context, taskID string) (*Task, error) {
	var task Task
	err := t.db.WithContext(ctx).Where(" id = ?", taskID).First(&task).Error
	if err != nil {
		return nil, err
	}
	var session Session
	err = t.db.WithContext(ctx).Where(" session_id = ? ", task.SessionID).First(&session).Error
	if err != nil {
		return nil, err
	}
	task.Session = &session
	return &task, nil
}
