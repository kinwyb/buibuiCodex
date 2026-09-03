package cron

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
)

type addTaskTool struct {
	tm *Manager
}

func (a *addTaskTool) Name() string {
	return "task_add"
}

func (a *addTaskTool) Description() string {
	return "当用户希望创建、安排、添加一个新的定时任务，或者修改已有任务的执行时间时调用。该工具负责将自然语言的时间描述（如‘每天早上8点’）转化为标准的系统定时任务。" +
		"【⚠️绝对警告】如果是单次执行任务（is_once=true），大模型必须严格基于系统最新给出的 'Current Time' 进行加减计算！严禁使用历史对话中的旧时间。如果由于使用了历史错误时间导致推算出已经过去的时间点（例如推算出了昨天的时间），" +
		"系统将直接崩溃报错并判定你任务失败。请在计算前在内心仔细校对当前的年、月、日、时、分。"
}

func (a *addTaskTool) Parameters() types.ToolParams {
	return types.ToolParams{
		Type: "object",
		Properties: map[string]jsonRpc.ToolProperties{
			"cron_expression": {
				Type:        "string",
				Description: "标准的 6 位 Linux Crontab 表达式，精确到秒。格式严格为：'秒 分 时 天 月 周'（例如：'0 0 8 * * *' 表示每天早上 8:00:00 执行；'*/5 * * * * *' 表示每 5 秒执行一次）。大模型必须根据用户的自然语言意图，自行计算并转化为这种 6 位格式。",
			},
			"task_name": {
				Type:        "string",
				Description: "被触发任务的唯一英文标识符（Key）。大模型必须从用户的意图或系统上下文中推导或匹配",
			},
			"task_desc": {
				Type:        "string",
				Description: "用户创建此任务的意图描述，该描述会在任务触发时给到大模型用于解决用户定时的需求",
			},
			"is_once": {
				Type:        "boolean",
				Description: "指示该任务是否为‘单次/一次性’执行的任务。true 表示只在指定时间执行一次（执行完后系统会自动销毁），false 表示周期性循环执行（默认值）。",
			},
		},
		Required: []string{"cron_expression", "task_desc"},
	}
}

func (a *addTaskTool) Execute(ctx context.Context, state *types.State, params map[string]any) (*types.InputMessage, error) {
	cronV := params["cron_expression"]
	if cronV == nil {
		return nil, errors.New("cron_expression is required")
	}
	cron := cronV.(string)
	if cron == "" {
		return nil, errors.New("cron_expression is required")
	}
	taskName := fmt.Sprintf("task_%d", rand.Int31())
	taskNameParam := params["task_name"]
	if taskNameParam != nil {
		if taskNameVal, ok := taskNameParam.(string); ok {
			taskName = taskNameVal
		}
	}
	var taskDesc string
	taskDescParam := params["task_desc"]
	if taskDescParam != nil {
		if taskDescVal, ok := taskDescParam.(string); ok {
			taskDesc = taskDescVal
		}
	}
	if taskDesc == "" {
		return nil, errors.New("task_desc is required")
	}
	isOne := false
	isOneParam := params["is_once"]
	if isOneParam != nil {
		if isOneVal, ok := isOneParam.(bool); ok {
			isOne = isOneVal
		}
	}
	task := db.Task{
		Name:        taskName,
		Description: "",
		Enabled:     true,
		OneTime:     isOne,
		Cron:        cron,
		SessionID:   state.SessionID,
		Content:     taskDesc,
	}
	err := a.tm.AddTask(&task)
	if err != nil {
		return nil, err
	}
	return &types.InputMessage{
		Content:   "定时任务创建成功,任务ID:[" + task.ID + "]",
		Timestamp: time.Now(),
	}, nil
}

type listTaskTool struct {
	tm *Manager
}

func (l *listTaskTool) Name() string {
	return "task_list"
}

func (l *listTaskTool) Description() string {
	return "当用户想要查看、列出、查询当前系统中已经存在的定时任务时调用。或者用户想要删除任务时调用获取任务ID用于删除。"
}

func (l *listTaskTool) Parameters() types.ToolParams {
	return types.ToolParams{
		Type: "object",
	}
}

func (l *listTaskTool) Execute(ctx context.Context, state *types.State, params map[string]any) (*types.InputMessage, error) {
	tasks, err := l.tm.store.TaskQueryBySessionID(ctx, state.SessionID)
	if err != nil {
		return nil, err
	}
	var taskList []map[string]string
	for _, task := range tasks {
		m := map[string]string{
			"任务ID": task.ID,
			"任务名称": task.Name,
			"任务内容": task.Content,
		}
		if v, ok := l.tm.jobMap[task.ID]; ok {
			// 对应 c.Entry(id).Next.Format("2006-01-02 15:04:05")
			m["下次执行时间"] = l.tm.cronEngine.Entry(v).Next.Format(time.DateTime)
		}
		taskList = append(taskList, m)
	}
	if len(taskList) == 0 {
		return &types.InputMessage{Content: "没有任务"}, nil
	}
	data, err := json.Marshal(taskList)
	if err != nil {
		return nil, fmt.Errorf("任务信息序列化失败：%w", err)
	}
	return &types.InputMessage{Content: string(data)}, nil
}

type deleteTaskTool struct {
	tm *Manager
}

func (d *deleteTaskTool) Name() string {
	return "task_delete"
}

func (d *deleteTaskTool) Description() string {
	return "当用户想要删除当前系统中已经存在的定时任务时调用。根据用户描述先用cron_task_list获取任务列表拿到要删除任务的ID再调用该工具删除"
}

func (d *deleteTaskTool) Parameters() types.ToolParams {
	return types.ToolParams{
		Type: "object",
		Properties: map[string]jsonRpc.ToolProperties{
			"task_id": {
				Type:        "string",
				Description: "任务的ID，可以通过cron_task_list工具获取",
			},
		},
		Required: []string{"task_id"},
	}
}

func (d *deleteTaskTool) Execute(ctx context.Context, state *types.State, params map[string]any) (*types.InputMessage, error) {
	var taskID string
	taskIDParam := params["task_id"]
	if taskIDParam != nil {
		if taskIDVal, ok := taskIDParam.(string); ok {
			taskID = taskIDVal
		}
	}
	if taskID == "" {
		return nil, errors.New("task_id is required")
	}
	task, err := d.tm.store.QueryByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task != nil {
		if task.SessionID != state.SessionID {
			return nil, errors.New("无权删除该任务")
		}
		err = d.tm.DeleteTask(taskID)
		if err != nil {
			return nil, err
		}
	}
	return &types.InputMessage{Content: "删除成功"}, nil
}
