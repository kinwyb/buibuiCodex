// 本文件实现「agent 操作 Web 界面」的两个基础工具。
//
// 分工：
//
//	ui_observe  取用户当前所见页面的快照（页面结构 + 可执行命令 + 数据摘要）
//	ui_act      执行一条界面命令（命令 id 来自上一次快照）
//
// 两个工具都是纯传输层：把请求按会话下发给浏览器、等回执、把结果原样交给模型。
// 页面结构的语义由前端负责生产，工具描述负责告诉模型怎么读——后端不为每个业务页面改代码。

package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// 界面请求的操作码，与 protocol.UIRequestData.Op 一一对应
const (
	uiOpObserve = "observe"
	uiOpAct     = "act"
	// uiOpNavigate 打开/切换页面：它是「去哪」，不是「操作当前页」，
	// 所以不需要 rev —— 跨页面的版本号没有意义。
	uiOpNavigate = "navigate"
)

// patchHint 界面回执里增量补丁的读法。
//
// 回执采用 ui-snap-c 紧凑文本格式（非 JSON），第一行是元数据，后续行是编码内容。
// 格式规范见 skills/ui-operation/SKILL.md 的「回执格式」段。
//
// v4 起档位由前端的桥比对新旧快照自己定（见 bodaWay 的 ui-bridge/handler.ts decideReceipt），
// 并用结构指纹 fp / basedOn 让模型自查手里的快照还作不作数。
const patchHint = "回执分三档，由前端桥比对新旧页面状态后自动定档：" +
	"结构变了（切页、打开或关闭表单弹窗、命令表换了一套、表的列目录变了）回完整 snapshot —— 整份替换手里那份；" +
	"取值变了回 patch —— 对上一份快照的增量，合进手里那份，没出现的段表示没变；" +
	"段内也按条给（form.fields 按字段 id、tables 按 name），但表里的 filters 与 selection 是**整段覆盖**" +
	"（flt 为空就是现在一个条件都没筛，sel:0 就是没有选中），没列出的项才保持原值；" +
	"列目录（f/t/y/fi/h 的 TSV）、guide、param-hints 不会重复下发，别当成缺失数据；" +
	"执行完没有可见变化就只回 result。" +
	"完整快照头行带 fp、补丁头行带 basedOn：两者对不上（或手里没有快照）就说明那份已过期，先 ui_observe 再动手。" +
	"失败回 err 行 + patch；rev 在每次回执头行里给，下一条命令直接用它。" +
	"回执第一行是元数据（ok/err、rev、result），后续行是 ui-snap-c 编码的快照或补丁文本，不是 JSON。"

// UITransport 把「服务端 → 浏览器」的反向请求能力注入给界面工具。
//
// 由 *ws.Server 实现（见 core/ws/server/ui.go 的 Request）。
// 这里只声明接口而不直接依赖 ws 包，是为了保持包依赖单向：core 不反向依赖 ws。
type UITransport interface {
	RequestChannel() string
	Request(ctx context.Context, sessionID string, req *protocol.UIRequestData) (*protocol.UIResponseData, error)
}

// UIBridge 界面工具与传输层之间的后置绑定中转。
//
// 时序问题：工具必须在 Manager.InitFromConfig 期间注册（agent 创建时就要带上工具），
// 而 ws 服务在那之后才启动。所以先用本对象占位，ws 起来后调用 SetTransport 注入。
// 注入前调用工具会得到一个明确的错误，而不是 panic。
type UIBridge struct {
	mu        sync.RWMutex
	transport UITransport
}

// NewUIBridge 创建界面工具桥
func NewUIBridge() *UIBridge {
	return &UIBridge{}
}

// SetTransport 注入传输层实现，通常传入 *ws.Server
func (b *UIBridge) SetTransport(t UITransport) {
	b.mu.Lock()
	b.transport = t
	b.mu.Unlock()
}

// request 发起一次界面请求，统一处理「通道未就绪」与「无会话上下文」
func (b *UIBridge) request(ctx context.Context, channel string, sessionID string, req *protocol.UIRequestData) (*protocol.UIResponseData, error) {
	b.mu.RLock()
	transport := b.transport
	b.mu.RUnlock()
	if transport == nil {
		return nil, errors.New("界面通道未启用：WebSocket 服务尚未启动")
	}
	if sessionID == "" {
		return nil, errors.New("界面工具缺少会话上下文：仅能在具体的 Web 会话里调用")
	}
	if transport.RequestChannel() != "" && transport.RequestChannel() != channel {
		return nil, errors.New("界面工具不支持：仅能在 Web 会话来源消息中调用")
	}
	return transport.Request(ctx, sessionID, req)
}

// RegisterUITools 注册界面工具组。
//
// 必须在 Manager.InitFromConfig 之前调用：工具组是按 agent 创建时统一挂载的，
// 时机错了 agent 就看不到这两个工具。
func RegisterUITools(bridge *UIBridge) {
	if bridge == nil {
		return
	}
	types.RegisterToolGroup("ui",
		"操作 Web 界面：获取用户当前所见页面的快照，并执行界面上的命令",
		&UIObserveTool{bridge: bridge},
		&UIActTool{bridge: bridge},
		&UINavigateTool{bridge: bridge},
	)
}

// UIObserveTool 获取用户当前所见页面的快照
type UIObserveTool struct {
	bridge *UIBridge
}

func (t *UIObserveTool) Name() string { return "ui_observe" }

func (t *UIObserveTool) Description() string {
	return "获取用户当前所见页面的快照：页面标识、可执行命令（actions）、数据查询条件（query）、" +
		"页面上的表格（tables：列目录、行数与排序、表内筛选 filters、选中 selection）与当前打开的表单。" +
		"快照是用户此刻的视野，不是完整数据集：表段只有目录，要行数据用 <表名>.page 现取。" +
		"任何界面操作前都应先调用本工具，以取得最新的 rev 与确实可用的命令 id。" +
		"注意：query / tables[].filters / tables[].selection 只是「当前是什么」，不能直接改；" +
		"要改变它们必须执行 actions 里对应的命令（如 query.set、<表名>.filter、<表名>.select）。" +
		"能不能按某列筛看 columns[].filter；枚举列的候选值不随快照下发，需要时用 <表名>.filter.options 现取。" +
		"commands 之外不存在任何操作入口。" +
		"guide 是这一页的操作说明（顺序、前置条件、业务术语）；" +
		"用户停在主页、没打开任何业务页面时，快照会带 navigation 列出可打开的页面，用 ui_navigate 打开。" +
		patchHint
}

func (t *UIObserveTool) Parameters() types.ToolParams {
	return types.ToolParams{
		Type:       "object",
		Properties: map[string]jsonRpc.ToolProperties{},
	}
}

func (t *UIObserveTool) Execute(ctx context.Context, state *types.State, _ map[string]any) (*types.InputMessage, error) {
	resp, err := t.bridge.request(ctx, channelOf(state), sessionIDOf(state), &protocol.UIRequestData{Op: uiOpObserve})
	if err != nil {
		return nil, err
	}
	return uiResponseMessage(resp), nil
}

// UIActTool 执行一条界面命令
type UIActTool struct {
	bridge *UIBridge
}

func (t *UIActTool) Name() string { return "ui_act" }

func (t *UIActTool) Description() string {
	return "执行一条界面命令。cmd 必须取自最近一次 ui_observe 返回的 actions[].id，" +
		"并带上同一次快照的 rev；params 按该 action 的 params 声明构造（键名即参数名），" +
		"没有 params 声明就表示这条命令不需要参数。返回执行结果，页面状态按 result / patch / snapshot 三档给。" +
		"回执里的状态是尽力而为的最新视图（命令可能还在异步落定）；需要权威状态时再 ui_observe 一次。 " +
		"失败（rev 已过期、命令不再可用、参数不合法）会返回具体原因与此刻的增量变化：按原因修正后再来，" +
		"只有 rev 过期才需要重新 ui_observe。" +
		"不要凭猜测构造命令 id 或参数名：快照里没有的就一定不存在。" +
		patchHint
}

func (t *UIActTool) Parameters() types.ToolParams {
	return types.ToolParams{
		Type: "object",
		Properties: map[string]jsonRpc.ToolProperties{
			"cmd": {
				Type:        "string",
				Description: "命令 ID，必须来自最近一次 ui_observe 返回的 actions[].id",
			},
			"params": {
				Type:        "object",
				Description: "命令参数：键名与取值按快照里该 action 的 params 声明构造；该 action 没有 params 时省略",
			},
			"rev": {
				Type:        "integer",
				Description: "最近一次 ui_observe 返回的快照版本号，用于避免按过期页面状态操作",
			},
		},
		Required: []string{"cmd"},
	}
}

func (t *UIActTool) Execute(ctx context.Context, state *types.State, params map[string]any) (*types.InputMessage, error) {
	cmd, _ := params["cmd"].(string)
	if cmd == "" {
		return nil, errors.New("cmd 参数缺失：命令 ID 必须来自最近一次 ui_observe 的 actions[].id")
	}
	req := &protocol.UIRequestData{Op: uiOpAct, Cmd: cmd}
	if raw, ok := params["params"]; ok && raw != nil {
		encoded, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("params 序列化失败: %w", err)
		}
		req.Params = encoded
	}
	if rev, ok := int64Param(params["rev"]); ok {
		req.Rev = rev
	}

	resp, err := t.bridge.request(ctx, channelOf(state), sessionIDOf(state), req)
	if err != nil {
		return nil, err
	}
	return uiResponseMessage(resp), nil
}

// UINavigateTool 打开或切换页面
type UINavigateTool struct {
	bridge *UIBridge
}

func (t *UINavigateTool) Name() string { return "ui_navigate" }

func (t *UINavigateTool) Description() string {
	return "打开或切换一个页面（等价于点左侧菜单），也可以只给 query 按名称搜索可用页面。" +
		"可用页面的清单见 ui_observe 返回的 navigation；菜单较大时用 query 检索。" +
		"目标页面打开并就绪后才会返回，回执里直接是新页面的快照，可以基于它继续 ui_act。" +
		"本工具不需要 rev：打开另一个页面跟当前页面的版本无关。" +
		"注意 params 只用于「查看某条单据」这类页面入参，不要用它传筛选条件 —— " +
		"参数不同会被当成不同页签，同一个页面会因此开出多个标签页。"
}

func (t *UINavigateTool) Parameters() types.ToolParams {
	return types.ToolParams{
		Type: "object",
		Properties: map[string]jsonRpc.ToolProperties{
			"view": {
				Type:        "string",
				Description: "目标页面标识（菜单的 View 字段），取值来自 ui_observe 的 navigation[].view 或本工具搜索结果的 view",
			},
			"query": {
				Type:        "string",
				Description: "按名称搜索可用页面，只搜不打开；与 view 二选一，不填 view 时才生效",
			},
			"params": {
				Type:        "object",
				Description: "传给目标页面的入参（例如查看某张单需要 orderId）。注意参数不同会开成不同页签，不要用来传筛选条件",
			},
		},
	}
}

func (t *UINavigateTool) Execute(ctx context.Context, state *types.State, params map[string]any) (*types.InputMessage, error) {
	view, _ := params["view"].(string)
	query, _ := params["query"].(string)
	view = strings.TrimSpace(view)
	query = strings.TrimSpace(query)
	if view == "" && query == "" {
		return nil, errors.New("需要提供 view（要打开的页面）或 query（按名称搜索可用页面）")
	}

	req := &protocol.UIRequestData{Op: uiOpNavigate, View: view, Query: query}
	if raw, ok := params["params"]; ok && raw != nil {
		encoded, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("params 序列化失败: %w", err)
		}
		req.Params = encoded
	}

	resp, err := t.bridge.request(ctx, channelOf(state), sessionIDOf(state), req)
	if err != nil {
		return nil, err
	}
	return uiResponseMessage(resp), nil
}

// sessionIDOf 取当前会话键（ws_<userID>_<chatID> 形态），即浏览器连接的订阅键
func sessionIDOf(state *types.State) string {
	if state == nil {
		return ""
	}
	return state.SessionID
}

func channelOf(state *types.State) string {
	if state == nil || state.Input == nil {
		return ""
	}
	return state.Input.Channel
}

// uiResponseMessage 把浏览器回执转成模型可读的文本。
//
// 前端的 snapshot / patch 都是 ui-snap-c 编码文本（JSON string，由 bodaWay 的 encoder.ts 生产）。
// 这里原样输出文本，不再套 JSON —— 二次转义会吃掉压缩收益，也增加模型解析成本。
//
// 输出格式：
//
//	第一行：元数据（ok/err、rev、result）
//	后续行：ui-snap-c 编码的快照或补丁
func uiResponseMessage(resp *protocol.UIResponseData) *types.InputMessage {
	if resp == nil {
		return &types.InputMessage{Content: "err\n界面没有返回任何结果"}
	}

	snapStr := rawText(resp.Snapshot)
	patchStr := rawText(resp.Patch)

	// 失败回执：err 头 + 可选的补丁
	if !resp.OK {
		reason := resp.Error
		if reason == "" {
			reason = "界面操作失败，且未返回原因"
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "err rev=%d reason=%s", resp.Rev, quoteVal(reason))
		if patchStr != "" {
			sb.WriteString("\n")
			sb.WriteString(patchStr)
		}
		return &types.InputMessage{Content: sb.String()}
	}

	// 成功回执：ok 头 + 可选的快照或补丁
	var sb strings.Builder
	fmt.Fprintf(&sb, "ok rev=%d", resp.Rev)
	if len(resp.Result) > 0 {
		fmt.Fprintf(&sb, " result=%s", string(resp.Result))
	}
	if snapStr != "" {
		sb.WriteString("\n")
		sb.WriteString(snapStr)
	} else if patchStr != "" {
		sb.WriteString("\n")
		sb.WriteString(patchStr)
	}
	return &types.InputMessage{Content: sb.String()}
}

// rawText 取出回执里的编码文本。
// 正常情况下 raw 是 JSON 字符串（ui-snap-c 文本）；万一不是，原样返回原始字节，绝不丢数据。
func rawText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

// quoteVal 给含空格或特殊字符的值加双引号
func quoteVal(s string) string {
	if strings.ContainsAny(s, " \t\n\"|=") {
		r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`)
		return `"` + r.Replace(s) + `"`
	}
	return s
}

// int64Param 兼容模型传入的数字形态（JSON 解析后通常是 float64）
func int64Param(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}
