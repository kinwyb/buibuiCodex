package cron

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"uuid"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/robfig/cron/v3"
)

type Manager struct {
	store      db.ITask
	cronEngine *cron.Cron
	jobMap     map[string]cron.EntryID
	mu         sync.Mutex
	bus        *bus.MessageBus
	ctx        context.Context
}

func NewManager(ctx context.Context, store db.ITask, bus *bus.MessageBus) *Manager {
	c := cron.New(cron.WithSeconds()) // 开启秒级支持
	return &Manager{
		store:      store,
		jobMap:     make(map[string]cron.EntryID),
		cronEngine: c,
		bus:        bus,
		ctx:        ctx,
	}
}

func (m *Manager) Start() error {
	m.cronEngine.Start()
	taskMap, err := m.store.Load(m.ctx)
	if err != nil {
		return err
	}
	for _, task := range taskMap {
		err = m.AddTask(task)
		if err != nil {
			return err
		}
	}
	return nil
}

// AddTask 添加任务,负责生成标准 cronExpr（如 "0 0 8 * * *"）和 taskName（如 "clean_log"）
func (m *Manager) AddTask(task *db.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	isNew := false
	if task.ID == "" {
		task.ID = uuid.New().String()
		isNew = true
	}

	// 检查是否已存在同名任务，若存在先删除（相当于修改）
	if oldID, exists := m.jobMap[task.ID]; exists {
		m.cronEngine.Remove(oldID)
	}

	// 动态绑定具体的业务逻辑（这里以闭包形式模拟）
	id, err := m.cronEngine.AddFunc(task.Cron, func() {
		m.executeRealLogic(task)
	})

	if err != nil {
		return fmt.Errorf("添加失败：Cron表达式错误或不支持 -> %v", err)
	}

	// 记录映射关系
	m.jobMap[task.ID] = id
	if isNew {
		m.store.Save(m.ctx, task)
	}
	return nil
}

// DeleteTask 删除任务
func (m *Manager) DeleteTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, exists := m.jobMap[taskID]
	if exists {
		// 从 cron 引擎中彻底移除
		m.cronEngine.Remove(id)
		// 从映射表中删除
		delete(m.jobMap, taskID)
	}
	m.store.Delete(m.ctx, taskID)

	return nil
}

// Tasks 获取所有定时任务
func (m *Manager) Tasks() (map[string]*db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tasks, err := m.store.Load(m.ctx)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	taskMap := make(map[string]*db.Task)
	for taskID, _ := range m.jobMap {
		if task, ok := tasks[taskID]; ok {
			taskMap[taskID] = task
		}
	}
	return taskMap, nil
}

func (m *Manager) executeRealLogic(task *db.Task) {
	slog.Info("【定时触发】正在执行任务:" + task.Name)
	fullTaskInfo, err := m.store.QueryByID(m.ctx, task.ID)
	if err != nil {
		slog.Error("task info query err", "error", err)
		return
	} else if fullTaskInfo.Session == nil {
		slog.Error("task session query fail")
		return
	}
	inbound := &types.InputMessage{
		ID:        fmt.Sprintf("task_%s", uuid.NewV4().String()),
		Channel:   fullTaskInfo.Session.Channel,
		AccountID: fullTaskInfo.Session.AccountID,
		SenderID:  fullTaskInfo.Session.UserID,
		ChatID:    fullTaskInfo.Session.ChatID,
		Content:   task.Content,
		Timestamp: time.Now(),
	}

	// 发布到 MessageBus
	if err := m.bus.PublishInput(context.Background(), inbound); err != nil {
		slog.Error("Failed to publish scheduled task",
			"task_id", task.ID,
			"channel", fullTaskInfo.Session.Channel,
			"chat_id", fullTaskInfo.Session.ChatID,
			"sender_id", fullTaskInfo.Session.UserID,
			"error", err)
		return
	}
	if task.OneTime { //单次任务，执行之后就直接删除
		m.DeleteTask(task.ID)
	}
}

// RegisterTools 获取工具列表
func (m *Manager) RegisterTools() {
	types.RegisterToolGroup("cron", "定时任务相关工具,允许添加,查询,删除定时任务",
		&addTaskTool{tm: m}, &listTaskTool{tm: m}, &deleteTaskTool{tm: m})
}

func (m *Manager) Stop() {
	m.cronEngine.Stop()
}
