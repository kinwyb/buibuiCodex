package codex

import (
	"context"
	"fmt"
	"strings"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
)

type TurnStartOptions = func(params *jsonRpc.StartTurnParams)

// TurnStartWithApprovalPolicy 设置审批权限
func TurnStartWithApprovalPolicy(policy jsonRpc.ApprovalPolicy) TurnStartOptions {
	return func(params *jsonRpc.StartTurnParams) {
		params.ApprovalPolicy = policy
	}
}

func TurnStartWithSandboxPolicy(policy jsonRpc.SandboxPolicy) TurnStartOptions {
	return func(params *jsonRpc.StartTurnParams) {
		params.SandboxPolicy = &policy
	}
}

type Turn struct {
	client *Client
	thread *Thread
	turnID string
	event  EventHandler
}

// NewTurn 创建一个turn
func NewTurn(client *Client, thread *Thread) *Turn {
	return &Turn{client: client, thread: thread}
}

func NewTurnWithID(thread *Thread, turnID string) (*Turn, error) {
	if thread == nil {
		return nil, fmt.Errorf("thread cannot be nil")
	}
	return &Turn{client: thread.client, thread: thread, turnID: turnID}, nil
}

func (t *Turn) ThreadID() string {
	return t.thread.threadID
}

func (t *Turn) TurnID() string {
	return t.turnID
}

// BuildInputWithText 文本输入
func (t *Turn) BuildInputWithText(text string, inputs ...jsonRpc.InputItem) []jsonRpc.InputItem {
	input := jsonRpc.InputItem{
		Type: "text",
		Text: text,
	}
	return append(inputs, input)
}

// BuildInputWithImage 远程图片输入
func (t *Turn) BuildInputWithImage(url string, inputs ...jsonRpc.InputItem) []jsonRpc.InputItem {
	input := jsonRpc.InputItem{
		Type: "image",
		URL:  url,
	}
	return append(inputs, input)
}

// BuildInputWithLocalImage 本地图片输入
func (t *Turn) BuildInputWithLocalImage(path string, inputs ...jsonRpc.InputItem) []jsonRpc.InputItem {
	input := jsonRpc.InputItem{
		Type: "localImage",
		Path: path,
	}
	return append(inputs, input)
}

// Start 启动一个turn
func (t *Turn) Start(ctx context.Context, input []jsonRpc.InputItem, options ...TurnStartOptions) error {
	params := jsonRpc.StartTurnParams{
		ThreadID: t.thread.threadID,
		Input:    input,
	}
	for _, option := range options {
		option(&params)
	}

	var res jsonRpc.StartTurnResult
	err := t.client.Call(ctx, TurnStart, params, &res)
	if err != nil {
		return err
	}
	t.turnID = res.Turn.TurnID
	t.thread.SubScribeEvent(t.turnID, t.eventHandler)
	return nil
}

// Interrupt 停止中断一个turn
func (t *Turn) Interrupt(ctx context.Context) error {
	params := jsonRpc.TurnInterruptParams{
		ThreadID: t.thread.threadID,
		TurnID:   t.turnID,
	}
	err := t.client.Call(ctx, TurnInterrupt, params, nil)
	if err != nil {
		return err
	}
	return nil
}

// Steer 添加额外的用户信息给当前的运行的trun
func (t *Turn) Steer(ctx context.Context, input []jsonRpc.InputItem) error {
	params := jsonRpc.TurnSteerParams{
		ThreadID: t.thread.threadID,
		TurnID:   t.turnID,
		Input:    input,
	}
	err := t.client.Call(ctx, TurnSteer, params, nil)
	if err != nil {
		return err
	}
	return nil
}

// RegisterEventHandler 注册事件处理
func (t *Turn) RegisterEventHandler(handler EventHandler) {
	t.event = handler
}

func (t *Turn) eventHandler(event *Event) {
	if t.event != nil {
		t.event(event)
		return
	}
	switch event.Method {
	// 1. 节点状态更新（捕获思考开始、工具调用开始）
	case string(ItemCompleted):
		itemEvt, err := event.To[jsonRpc.ItemLifecycleEvent]()
		if err != nil {
			fmt.Printf("item lifecycle event error: %v\n", err)
			return
		}
		item := itemEvt.Item
		switch item.Type {
		case jsonRpc.Reasoning:
			fmt.Printf("🤔 [Agent 思考结果] %s \n", strings.Join(item.ReasoningItemDetail.Summary, "\n"))
		case jsonRpc.AgentMessage:
			fmt.Printf("😊 [Agent 回答] %s\n", item.AgentMessageItemDetail.Text)
		case jsonRpc.CommandExecution:
			fmt.Printf("✅ [工具执行完毕] %s \n", item.CommandExecutionItemDetail.AggregatedOutput)
		default:
			fmt.Printf("⚠️ [%s completed] => %s \n", item.Type, event.RawData.String())
		}
		// 1. 节点状态更新（捕获思考开始、工具调用开始）
	case string(ItemStarted):
		itemEvt, err := event.To[jsonRpc.ItemLifecycleEvent]()
		if err != nil {
			fmt.Printf("item lifecycle event error: %v\n", err)
			return
		}
		item := itemEvt.Item
		switch item.Type {
		case jsonRpc.Reasoning:
			fmt.Printf("\n🤔 [Agent 思考中...]\n")
		case jsonRpc.CommandExecution, jsonRpc.WebSearch:
			fmt.Printf("\n🛠️ [执行工具/Shell]: %s\n", item.Command)
		case jsonRpc.DynamicToolCall:
			fmt.Printf("\n[动态工具调用]: %s\n", item.ID)
		default:
			fmt.Printf("⚠️ [%s started] => %s \n", item.Type, event.RawData.String())
		}
	case string(ItemReasoningSummaryTextDelta), string(ItemReasoningTextDelta), string(ItemAgentMessageDelta):
		itemEvt, err := event.To[jsonRpc.TextDeltaEvent]()
		if err != nil {
			fmt.Printf("item lifecycle event error: %v\n", err)
			return
		}
		fmt.Printf("%s", itemEvt.Delta)
	case string(ThreadTokenUsageUpdated):
		itemEvt, err := event.To[jsonRpc.TokenUsageNotification]()
		if err != nil {
			fmt.Printf("item lifecycle event error: %v\n", err)
			return
		}
		fmt.Printf("\n\n🎉 [Turn token消耗统计] Total Tokens: %d,Last Tokens: %d\n",
			itemEvt.TokenUsage.Total.TotalTokens, itemEvt.TokenUsage.Last.TotalTokens)
	}
}
