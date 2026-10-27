// Package replay 把落库的 codex 原始事件还原成可直接下发的 ws 帧。
//
// 设计要点：产出的就是 kind=agent 的信封，与 bridge 实时转换出来的帧完全同构。
// 前端因此只需要一条渲染管线——历史、回放、实时走同一个处理器，
// 不必判断「这条是历史还是新消息」。
//
// 只还原关键帧（item 生命周期、审批、错误），不重放 token 级 delta：
// 回放是给人检查「哪一步出了问题」用的，逐帧打字机对判断毫无帮助，
// 却会让载荷膨胀几个数量级。delta 在库里仍然完整保留，需要时按 method 再取。
package replay

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kinwyb/buibuiCodex/codex"
	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// 回放上限的默认值
const (
	// DefaultMaxFrames 单轮最多还原的帧数
	DefaultMaxFrames = 200
	// DefaultBodyLimit 单个文本内容（思考、命令输出、工具结果）的截断字节数
	DefaultBodyLimit = 4096
)

// keyMethods 回放需要的关键帧 method 白名单。
// 数据库侧按它过滤，把逐 token 的 delta 挡在查询之外。
var keyMethods = []string{
	string(codex.ItemStarted),
	string(codex.ItemCompleted),
	string(codex.ItemToolCall),
	string(codex.ItemCommandExecutionRequestApproval),
	string(codex.ExecCommandApproval),
	string(codex.ItemFileChangeRequestApproval),
	string(codex.ItemPermissionsRequestApproval),
	string(codex.ApplyPatchApproval),
	string(codex.TurnCompleted),
	string(codex.NotificationMethodError),
}

// KeyMethods 返回关键帧白名单，供调用方拼 db.TurnEventQuery.Methods。
func KeyMethods() []string {
	out := make([]string, len(keyMethods))
	copy(out, keyMethods)
	return out
}

// Options 回放参数
type Options struct {
	// SessionID 会话标识，写入帧信封
	SessionID string
	// ReqID 请求标识，写入帧信封（同一轮的帧共享，前端据此归组）
	ReqID string
	// Agent agent 名称，写入载荷
	Agent string
	// MaxFrames 最多返回的帧数，<=0 取 DefaultMaxFrames
	MaxFrames int
	// BodyLimit 单个文本内容的截断字节数，<=0 取 DefaultBodyLimit；
	// 传负数表示不截断（单轮详情场景）
	BodyLimit int
}

// Build 把一轮的事件还原为 agent 帧序列。
//
// 第二个返回值表示「回放有损」：帧数超出上限被丢弃，或任一文本内容被截断。
// 调用方可据此提示用户「这不是完整过程」。
//
// events 必须按 seq 升序（db.TurnEventList 已保证），输出顺序即执行顺序。
func Build(events []db.TurnEvent, opts Options) ([]*protocol.Envelope, bool) {
	maxFrames := opts.MaxFrames
	if maxFrames == 0 {
		maxFrames = DefaultMaxFrames
	}
	if maxFrames < 0 {
		maxFrames = int(^uint(0) >> 1)
	}
	bodyLimit := opts.BodyLimit
	if bodyLimit == 0 {
		bodyLimit = DefaultBodyLimit
	}

	frames := make([]*protocol.Envelope, 0, len(events))
	lossy := false
	for _, ev := range events {
		if len(frames) >= maxFrames {
			return frames, true
		}
		data, eventID := stepOf(ev, bodyLimit)
		if data == nil {
			continue
		}
		lossy = lossy || data.Truncated
		if opts.Agent != "" {
			data.Agent = opts.Agent
		}
		frames = append(frames, envelopeOf(ev, data, eventID, opts))
	}
	return frames, lossy
}

// envelopeOf 把载荷包装成与实时流一致的信封
func envelopeOf(ev db.TurnEvent, data *protocol.AgentData, eventID string, opts Options) *protocol.Envelope {
	env := protocol.NewEnvelope(protocol.KindAgent, data.Event).
		WithSession(opts.SessionID, opts.ReqID).
		WithEventID(eventID).
		WithData(data)
	// 用事件表的自增 seq 回填，前端可直接当作翻页游标
	env.Seq = int64(ev.Seq)
	if !ev.CreateTime.IsZero() {
		env.TS = ev.CreateTime.UnixMilli()
	}
	return env
}

// stepOf 单条事件 → 回放步骤。
// 第二个返回值是上游实体标识（item ID / callId），供前端按实体归并多次更新；
// data 为 nil 表示该事件不参与回放。
func stepOf(ev db.TurnEvent, bodyLimit int) (*protocol.AgentData, string) {
	switch ev.Method {
	case string(codex.ItemStarted), string(codex.ItemCompleted):
		return itemStep(ev, bodyLimit)
	case string(codex.ItemCommandExecutionRequestApproval),
		string(codex.ExecCommandApproval),
		string(codex.ItemFileChangeRequestApproval),
		string(codex.ItemPermissionsRequestApproval),
		string(codex.ApplyPatchApproval):
		// 审批帧的标识在 data.approval.id 里，不再占用 eventID
		return approvalStep(ev, bodyLimit), ""
	case string(codex.ItemToolCall):
		return toolCallStep(ev, bodyLimit)
	case string(codex.TurnCompleted):
		return turnCompletedStep(ev, bodyLimit), ""
	case string(codex.NotificationMethodError):
		return errorStep(ev, bodyLimit), ""
	default:
		return nil, ""
	}
}

// itemStep 还原 item 生命周期事件（思考 / 消息 / 命令 / 搜索 / 工具）
func itemStep(ev db.TurnEvent, bodyLimit int) (*protocol.AgentData, string) {
	var lifecycle jsonRpc.ItemLifecycleEvent
	if err := json.Unmarshal([]byte(ev.RawData), &lifecycle); err != nil {
		return nil, ""
	}
	item := lifecycle.Item
	done := ev.Method == string(codex.ItemCompleted)
	if item.Type == "" {
		return nil, ""
	}

	data := &protocol.AgentData{
		// 回放一律是全量快照：前端按 EventID/工具 ID 覆盖同一条消息，
		// 与实时流里 completed 帧的语义一致。
		Delta: false,
		Meta:  map[string]any{"codex_item_type": string(item.Type)},
	}

	switch item.Type {
	case jsonRpc.Reasoning:
		data.Event = eventOf(done, types.EventReasoningCompleted, types.EventReasoningStart)
		data.Role = string(types.RoleAssistant)
		data.Reasoning, data.Truncated = clip(reasoningText(lifecycle), bodyLimit)

	case jsonRpc.AgentMessage:
		data.Event = eventOf(done, types.EventMessageCompleted, types.EventMessageStart)
		data.Role = string(types.RoleAssistant)
		data.Content, data.Truncated = clip(item.Text, bodyLimit)

	case jsonRpc.UserMessage:
		data.Event = eventOf(done, types.EventUserMessageCompleted, types.EventUserMessageStart)
		data.Role = string(types.RoleUser)
		data.Content, data.Truncated = clip(item.Text, bodyLimit)

	case jsonRpc.CommandExecution:
		data.Event = eventOf(done, types.EventToolCompleted, types.EventToolStart)
		data.Role = string(types.RoleTool)
		call := types.ToolCall{
			ID:     item.ID,
			Type:   types.ToolTypeCommand,
			Name:   "command",
			Params: item.Command,
			Extra:  map[string]any{"status": item.Status},
		}
		if item.Cwd != "" {
			call.Extra["cwd"] = item.Cwd
		}
		if item.ExitCode != 0 {
			call.Extra["exitCode"] = item.ExitCode
		}
		if item.DurationMs > 0 {
			call.Extra["durationMs"] = item.DurationMs
		}
		for i, action := range item.CommandActions {
			call.Extra[fmt.Sprintf("action_%d", i)] = fmt.Sprintf("%s_%s", action.Type, action.Command)
		}
		call.Result, data.Truncated = clip(item.AggregatedOutput, bodyLimit)
		data.ToolCalls = []types.ToolCall{call}

	case jsonRpc.WebSearch:
		data.Event = eventOf(done, types.EventToolCompleted, types.EventToolStart)
		data.Role = string(types.RoleTool)
		call := types.ToolCall{
			ID:     item.ID,
			Type:   types.ToolTypeWebSearch,
			Name:   "web_search",
			Params: item.Query,
			Extra:  map[string]any{},
		}
		if item.Action.Type != "" {
			call.Extra["actionType"] = item.Action.Type
		}
		for i, q := range item.Action.Queries {
			call.Extra[fmt.Sprintf("query_%d", i)] = q
		}
		call.Result, data.Truncated = clip(rawText(item.Results), bodyLimit)
		data.ToolCalls = []types.ToolCall{call}

	case jsonRpc.DynamicToolCall:
		data.Event = eventOf(done, types.EventToolCompleted, types.EventToolStart)
		data.Role = string(types.RoleTool)
		name := item.Tool
		if name == "" {
			name = "tool"
		}
		params, _ := json.Marshal(item.Arguments)
		if string(params) == "null" {
			params = nil
		}
		call := types.ToolCall{
			ID:   item.ID,
			Type: types.ToolTypeFunction,
			Name: name,
		}
		extra := map[string]any{}
		if item.Namespace != "" {
			extra["namespace"] = item.Namespace
		}
		if item.CallId != "" {
			extra["callID"] = item.CallId
		}
		// 与实时流保持同一套 extra 字段，前端才能用同一条渲染分支
		if item.Status != "" {
			extra["status"] = item.Status
		}
		if item.ToolCallItemDetail.Success != nil {
			extra["success"] = *item.ToolCallItemDetail.Success
		}
		if item.DurationMs > 0 {
			extra["durationMs"] = item.DurationMs
		}
		if len(extra) > 0 {
			call.Extra = extra
		}
		call.Params, _ = clip(string(params), bodyLimit)
		// 结果在 contentItems 里，result 只是兜底；两条链路共用同一份解析
		call.Result, data.Truncated = clip(item.ToolCallItemDetail.ResultText(), bodyLimit)
		data.ToolCalls = []types.ToolCall{call}

	default:
		// 未单独建模的 item 类型（fileChange / mcpToolCall / plan ...）
		// 统一降级为工具步骤，原始 item 保留在结果里，信息不丢。
		data.Event = eventOf(done, types.EventToolCompleted, types.EventToolStart)
		data.Role = string(types.RoleTool)
		data.Meta["unparsed"] = true
		call := types.ToolCall{
			ID:    item.ID,
			Type:  types.ToolTypeFunction,
			Name:  string(item.Type),
			Extra: map[string]any{"raw": true},
		}
		call.Result, data.Truncated = clip(rawItem(ev.RawData), bodyLimit)
		data.ToolCalls = []types.ToolCall{call}
	}
	// 同一个 item 的 started / completed 共享 item.ID，
	// 前端靠它把两次更新收敛成一条，不必依赖到达顺序。
	return data, item.ID
}

// approvalStep 还原审批请求，让用户能看到「哪一步停下来等确认、被允许还是被拒」
func approvalStep(ev db.TurnEvent, bodyLimit int) *protocol.AgentData {
	var approval jsonRpc.BaseApprovalEvent
	_ = json.Unmarshal([]byte(ev.RawData), &approval)

	content := strings.TrimSpace(approval.Reason)
	// 命令类审批把具体命令带上，只有理由的话看不出要执行什么
	if strings.Contains(strings.ToLower(ev.Method), "command") && len(approval.Command) > 0 {
		command := strings.Join(approval.Command, " ")
		if content == "" {
			content = "执行命令：" + command
		} else {
			content = fmt.Sprintf("执行命令：%s，理由：%s", command, content)
		}
	}
	if content == "" {
		return nil
	}

	data := &protocol.AgentData{
		Event: string(types.EventApproval),
		Role:  string(types.RoleSystem),
		Delta: false,
		Meta:  map[string]any{"approval_method": ev.Method},
	}
	data.Content, data.Truncated = clip(content, bodyLimit)
	data.Approval = &protocol.ApprovalData{ID: approval.ItemID, Content: data.Content}
	return data
}

// toolCallStep 还原本地工具调用请求。
// 正常流程后面还会跟上同 ID 的 item/completed（带结果），前端按工具 ID 覆盖即可。
func toolCallStep(ev db.TurnEvent, bodyLimit int) (*protocol.AgentData, string) {
	var call jsonRpc.ToolCall
	if err := json.Unmarshal([]byte(ev.RawData), &call); err != nil {
		return nil, ""
	}
	name := call.Tool
	if name == "" {
		name = "tool"
	}
	params, _ := json.Marshal(call.Arguments)
	if string(params) == "null" {
		params = nil
	}
	data := &protocol.AgentData{
		Event: string(types.EventToolStart),
		Role:  string(types.RoleTool),
		Delta: false,
		Meta:  map[string]any{"codex_item_type": string(jsonRpc.DynamicToolCall)},
	}
	tool := types.ToolCall{
		ID:   call.CallId,
		Type: types.ToolTypeFunction,
		Name: name,
	}
	if call.Namespace != "" {
		tool.Extra = map[string]any{"namespace": call.Namespace}
	}
	tool.Params, _ = clip(string(params), bodyLimit)
	data.ToolCalls = []types.ToolCall{tool}
	return data, call.CallId
}

// turnCompletedStep 只在轮次以错误收尾时产出帧。
// 正常结束不需要额外帧——message_completed 已经代表回答收敛。
func turnCompletedStep(ev db.TurnEvent, bodyLimit int) *protocol.AgentData {
	var turnEvt jsonRpc.TurnEvent
	if err := json.Unmarshal([]byte(ev.RawData), &turnEvt); err != nil {
		return nil
	}
	if turnEvt.Turn.Error == nil {
		return nil
	}
	message := turnEvt.Turn.Error.Message
	if message == "" {
		message = "本轮执行失败"
	}
	data := &protocol.AgentData{
		Event: string(types.EventError),
		Role:  string(types.RoleSystem),
		Delta: false,
		Meta: map[string]any{
			"codex_turn_status":   turnEvt.Turn.Status,
			"codex_turn_duration": turnEvt.Turn.DurationMs,
		},
	}
	if details := turnEvt.Turn.Error.AdditionalDetails; details != "" {
		data.Meta["codex_error_details"] = clipRaw(details, bodyLimit)
	}
	msg, truncated := clip(message, bodyLimit)
	data.Content = msg
	data.Truncated = truncated
	data.Error = &protocol.ErrorBody{Code: "turn_failed", Message: msg}
	return data
}

// errorStep 还原 error 通知
func errorStep(ev db.TurnEvent, bodyLimit int) *protocol.AgentData {
	var errEvt jsonRpc.CodexErrorEvent
	if err := json.Unmarshal([]byte(ev.RawData), &errEvt); err != nil {
		return nil
	}
	message := errEvt.Error.Message
	if message == "" {
		return nil
	}
	data := &protocol.AgentData{
		Event: string(types.EventError),
		Role:  string(types.RoleSystem),
		Delta: false,
	}
	if errEvt.WillRetry {
		data.Meta = map[string]any{"willRetry": true}
	}
	data.Content, data.Truncated = clip(message, bodyLimit)
	data.Error = &protocol.ErrorBody{Code: "agent_error", Message: data.Content}
	return data
}

// ---- 文本处理 ----

// reasoningText 拼出完整思考文本。
// summary 是可展示的思考摘要，content 是更原始的思考分段，两者都取以免丢内容。
func reasoningText(lifecycle jsonRpc.ItemLifecycleEvent) string {
	item := lifecycle.Item
	parts := make([]string, 0, len(item.Summary)+len(item.Content))
	for _, s := range item.Summary {
		if s = strings.TrimSpace(s); s != "" {
			parts = append(parts, s)
		}
	}
	for _, c := range item.Content {
		switch v := c.(type) {
		case string:
			if v = strings.TrimSpace(v); v != "" {
				parts = append(parts, v)
			}
		case map[string]any:
			if text, ok := v["text"].(string); ok {
				if text = strings.TrimSpace(text); text != "" {
					parts = append(parts, text)
				}
			}
		}
	}
	return strings.Join(parts, "\n")
}

// rawText 把 json.RawMessage 还原成可读文本：字符串字面量去掉引号，其余原样返回。
func rawText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

// rawItem 取事件里的 item 原文，用于未建模的 item 类型降级展示
func rawItem(rawData string) string {
	var envelope struct {
		Item json.RawMessage `json:"item"`
	}
	if err := json.Unmarshal([]byte(rawData), &envelope); err != nil {
		return ""
	}
	return string(envelope.Item)
}

// clip 按字节截断超长内容，返回结果与是否发生过截断。
// 回退到 UTF-8 字符边界，避免把多字节字符切成乱码。
func clip(s string, limit int) (string, bool) {
	if limit < 0 || len(s) <= limit {
		return s, false
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "\n…（已截断，原文 " + strconv.Itoa(len(s)) + " 字节）", true
}

// clipRaw 是 clip 的丢弃截断标记版本，用于元数据等不参与展示的字段
func clipRaw(s string, limit int) string {
	out, _ := clip(s, limit)
	return out
}

// eventOf 按生命周期阶段选取事件类型
func eventOf(done bool, completed, started types.EventType) string {
	if done {
		return string(completed)
	}
	return string(started)
}

// ensureExtra 保证 Extra 可写
func ensureExtra(extra map[string]any) map[string]any {
	if extra == nil {
		return map[string]any{}
	}
	return extra
}
