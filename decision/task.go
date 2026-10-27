package decision

import (
	"context"
	"sort"
)

// Task 一次请求里同时问多个问题的构造器，问题之间互不影响，共享同一份 state。
// 问题数上限为 MaxQuestions，超出部分 Ask 会直接报错，避免白跑一次网络请求。
//
//	resp, err := decision.NewTask(ticket).
//		Choice("department", "应该由哪个团队处理？", decision.ChoiceOptions{
//			"billing": "支付、退款和账单问题", "technical": "产品故障和集成问题"}).
//		YesNo("escalate", "是否需要立即通知值班人员？").
//		Score("severity", "这个问题有多严重？", "轻微", "影响可用", "核心不可用", "严重业务影响").
//		Ask(ctx, client)
type Task struct {
	// State 待决策的业务状态：文本、对象或数组。
	State any
	// Model 非空时覆盖客户端默认模型。
	Model string
	// Questions 问题表，key 为自定义问题 id。
	Questions map[string]Question
}

// NewTask 以给定 state 创建任务。
func NewTask(state any) *Task {
	return &Task{State: state, Questions: map[string]Question{}}
}

// Add 追加任意问题。
func (t *Task) Add(id string, q Question) *Task {
	if t.Questions == nil {
		t.Questions = map[string]Question{}
	}
	t.Questions[id] = q
	return t
}

// Choice 追加多选一问题。
func (t *Task) Choice(id, instructions string, options ChoiceOptions) *Task {
	return t.Add(id, NewChoice(instructions, options))
}

// YesNo 追加是非判断问题。
func (t *Task) YesNo(id, instructions string) *Task {
	return t.Add(id, NewYesNo(instructions))
}

// Score 追加有序量表问题，levels 从低到高。
func (t *Task) Score(id, instructions string, levels ...string) *Task {
	return t.Add(id, NewScore(instructions, levels...))
}

// IDs 返回问题 id 列表（按字典序，便于日志和断言）。
func (t *Task) IDs() []string {
	ids := make([]string, 0, len(t.Questions))
	for id := range t.Questions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Request 生成待发送的请求体。
func (t *Task) Request() *Request {
	return &Request{Model: t.Model, State: t.State, Questions: t.Questions}
}

// Ask 用 client 执行任务。
func (t *Task) Ask(ctx context.Context, c *Client) (*Response, error) {
	return c.AskRequest(ctx, t.Request())
}
