// Package preflow 在用户消息进入 agent 之前，用决策模型把「要不要打开某个页面、
// 要不要点开某个入口」这类高频界面路由判断先做掉。
//
// 为什么放在这里：这些判断只关心「用户这句话对应菜单里的哪一项」，是分类问题，
// 交给决策模型一次前向就能出结果（选项名 + 概率 + 置信度，不生成文本）；
// 原来要靠 agent 花 token 去 ui_observe 找 navigation、再 ui_navigate 打开，
// 打开后还要再 observe 一次才知道有没有子入口。前置之后 agent 直接开工做业务。
//
// 界面动作复用的是已有的反向 RPC（ui.request 的 observe / navigate / act），
// 因此协议和前端都不需要改；任何一步失败都只是「不做前置」，消息照旧进 agent。
package preflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
	"github.com/kinwyb/buibuiCodex/decision"
)

// 界面操作码，与 protocol.UIRequestData.Op 对应
const (
	opObserve  = "observe"
	opNavigate = "navigate"
	opAct      = "act"
)

// 决策选项里的兜底项：选项名用 ASCII，避免模型回传时因空格/标点变形
const (
	optionCurrent = "current" // 无需切换页面
	optionNone    = "none"    // 无需点开任何入口
	optionOther   = "other"   // 以上都不合适
)

// Transport 界面反向请求能力，由 *ws.Server 实现（与 core.UITransport 同签名）
type Transport interface {
	Request(ctx context.Context, sessionID string, req *protocol.UIRequestData) (*protocol.UIResponseData, error)
}

// Decider 决策模型调用能力，由 *decision.Client 实现
type Decider interface {
	AskRequest(ctx context.Context, req *decision.Request) (*decision.Response, error)
}

// Config 前置路由的开关与预算
type Config struct {
	Enabled bool
	// Timeout 整个前置流程的总预算：一次 observe + 两次决策 + navigate + act。
	// 这段时间用户消息还没进 agent，所以宁可短一点也不许拖。
	Timeout time.Duration
	// DeepAction 打开页面后是否继续判断子入口（弹窗、表单、抽屉）
	DeepAction bool
	// Annotate 是否把「已做完的界面动作」作为一行说明附到用户消息里，
	// 让 agent 不必再 observe/navigate 一遍。
	Annotate bool
	// MaxPages / MaxActions 送给模型的候选数量上限：选项太多既拖慢决策也稀释判断
	MaxPages   int
	MaxActions int
}

// SetDefaults 填充默认值
func (c *Config) SetDefaults() {
	if c.Timeout <= 0 {
		c.Timeout = 8 * time.Second
	}
	if c.MaxPages <= 0 {
		c.MaxPages = 60
	}
	if c.MaxActions <= 0 {
		c.MaxActions = 40
	}
}

// Request 一次前置路由的输入
type Request struct {
	// ReqID 本次用户消息的 ID，只用于日志串联
	ReqID string
	// SessionID 会话键，界面请求按它投递给浏览器
	SessionID string
	// Content 用户原话
	Content string
}

// Step 一步已执行或待执行的界面动作
type Step struct {
	// Kind navigate 打开页面 | act 点开入口
	Kind string `json:"kind"`
	// Target navigate 的 view 或 act 的命令 id
	Target string `json:"target"`
	// Label 人可读的名字
	Label string `json:"label,omitempty"`
	// Rev 动作后页面的版本号
	Rev int64 `json:"rev,omitempty"`
}

// Decision 一次决策模型判断的轨迹
type Decision struct {
	// Kind page 选页面 | entry 选入口
	Kind string `json:"kind"`
	// Answer 模型给出的选项名（菜单标题本身；current / none / other 表示不动界面）
	Answer string `json:"answer"`
	// Target 选项映射回的真实目标：view 或命令 id，映射不上为空
	Target string `json:"target,omitempty"`
	// Confidence 模型对该答案的置信度
	Confidence float64 `json:"confidence"`
	// Margin 概率第一与第二的差距，用于判断模型是不是在几个相近菜单之间犹豫
	Margin float64 `json:"margin,omitempty"`
	// Options 送出去判断的候选数量（含兜底项）
	Options int `json:"options"`
	// Top 概率最高的几项，调阈值时全靠它
	Top []Candidate `json:"top,omitempty"`
	// Error 调用或解析失败的原因
	Error string `json:"error,omitempty"`
}

// Candidate 一个候选项及其概率
type Candidate struct {
	Name   string  `json:"name"`
	Target string  `json:"target,omitempty"`
	Prob   float64 `json:"prob"`
}

// Result 前置路由的结果。Route 永不返回 nil：调用方只看 Steps/Note 是否为空。
type Result struct {
	// Steps 已经替用户做完的界面动作
	Steps []Step
	// Pending 决策模型判断该点开、但本层不能代做的入口（需要业务参数）
	Pending []Step
	// Reason 没动作或提前结束的原因（日志与排查用）
	Reason string
	// Decisions 每一次判断的过程记录，排查「为什么没反应」就看这里
	Decisions []Decision
	// Note 附加到用户消息里的一行说明，为空表示消息保持原样
	Note string
	// Elapsed 整个前置流程耗时
	Elapsed time.Duration
}

// Applied 是否真的动过界面
func (r *Result) Applied() bool {
	return r != nil && len(r.Steps) > 0
}

// Router 决策模型 + 界面传输组成的前置路由
type Router struct {
	cfg Config
	dec Decider
	tr  Transport
	log *slog.Logger
}

// New 创建 Router。dec/tr 允许为空，此时 Route 直接跳过（便于配置未就绪时照常运行）。
func New(cfg Config, dec Decider, tr Transport, logger *slog.Logger) *Router {
	cfg.SetDefaults()
	if logger == nil {
		logger = slog.Default()
	}
	return &Router{cfg: cfg, dec: dec, tr: tr, log: logger}
}

// Enabled 是否启用前置路由
func (r *Router) Enabled() bool {
	return r != nil && r.cfg.Enabled && r.dec != nil && r.tr != nil
}

// Route 依次执行：取当前页面快照 -> 决策要不要换页面 -> 决策要不要点开子入口。
//
// 任何一步失败都只记录原因并返回，调用方继续把消息投递给 agent。
func (r *Router) Route(ctx context.Context, req *Request) *Result {
	res := &Result{}
	content := ""
	if req != nil {
		content = trimSpace(req.Content)
	}
	if req == nil || !r.Enabled() || content == "" || req.SessionID == "" || isCommand(content) {
		res.Reason = "未启用或无需前置路由"
		return res
	}

	started := time.Now()
	defer func() {
		res.Elapsed = time.Since(started)
		r.logResult(req, res)
	}()

	// 总预算：超时后剩下的步骤直接放弃，界面保持当前状态，不影响后续投递
	ctx, cancel := context.WithTimeout(ctx, r.cfg.Timeout)
	defer cancel()

	page, err := r.observe(ctx, req.SessionID)
	if err != nil {
		res.Reason = "取页面快照失败：" + err.Error()
		return res
	}
	current := page

	// 1. 要不要打开另一个页面
	halted := false
	if len(page.Nav) > 0 {
		view, reason, err := r.pickPage(ctx, res, content, page)
		switch {
		case err != nil:
			res.Reason = "页面决策失败：" + err.Error()
			// 连该去哪都没判出来，就别在当前页面上点入口了：那多半是在错的页面上瞎点
			halted = true
		case view == "":
			res.Reason = reason
		default:
			opened, err := r.navigate(ctx, req.SessionID, view)
			if err != nil {
				res.Reason = fmt.Sprintf("打开页面 %s 失败：%s", view, err.Error())
				halted = true
			} else {
				label := view
				if item, ok := page.NavOf(view); ok && item.Title != "" {
					label = item.Title
				}
				res.Steps = append(res.Steps, Step{Kind: opNavigate, Target: view, Label: label, Rev: opened.Rev})
				current = opened
			}
		}
	}

	// 2. 页面就位后，要不要再点一个子入口
	if !halted && r.cfg.DeepAction && len(current.Actions) > 0 {
		id, reason, err := r.pickAction(ctx, res, content, current)
		switch {
		case err != nil:
			appendReason(res, "入口决策失败："+err.Error())
		case id == "":
			appendReason(res, reason)
		default:
			action, ok := current.Action(id)
			if !ok {
				appendReason(res, "决策返回了快照里没有的入口 "+id)
				break
			}
			if action.RequiresParams {
				// 参数得从业务里取，那是 agent 的活；这里只报告「该点开哪个」
				res.Pending = append(res.Pending, Step{Kind: opAct, Target: id, Label: action.Label, Rev: current.Rev})
				break
			}
			acted, err := r.act(ctx, req.SessionID, id, current.Rev)
			if err != nil {
				appendReason(res, fmt.Sprintf("点开入口 %s 失败：%s", id, err.Error()))
				break
			}
			res.Steps = append(res.Steps, Step{Kind: opAct, Target: id, Label: action.Label, Rev: acted.Rev})
			current = acted
		}
	}

	if r.cfg.Annotate {
		res.Note = r.note(res, current)
	}
	return res
}

// note 把做完的动作告诉 agent：它就不必再 observe 找菜单、navigate 开页面了。
func (r *Router) note(res *Result, current *Page) string {
	if res == nil || (len(res.Steps) == 0 && len(res.Pending) == 0) {
		return ""
	}
	var done, waiting []string
	for _, step := range res.Steps {
		switch step.Kind {
		case opNavigate:
			done = append(done, fmt.Sprintf("已打开页面「%s」(view=%s)", step.Label, step.Target))
		case opAct:
			done = append(done, fmt.Sprintf("已点开「%s」(cmd=%s)", step.Label, step.Target))
		}
	}
	for _, step := range res.Pending {
		waiting = append(waiting, fmt.Sprintf("「%s」(cmd=%s) 需要业务参数，未替你点开", step.Label, step.Target))
	}

	// 只判出该点哪个入口、一个动作都没做时，标题不能写成「已就绪」骗 agent
	head := "[界面已就绪]"
	if len(done) == 0 {
		head = "[界面待处理]"
	}
	out := head + " " + strings.Join(done, " ")
	if len(waiting) > 0 {
		if len(done) > 0 {
			out += "；"
		}
		out += strings.Join(waiting, "；")
	}
	if current != nil && current.Rev > 0 {
		out += fmt.Sprintf("。当前页面 rev=%d", current.Rev)
	}
	out += "。请勿重复 ui_navigate；需要最新界面状态再 ui_observe。"
	return out
}

// pickPage 判断要不要打开另一个页面。
//
// 返回目标 view；为空时 reason 说明为什么不动界面（模型选了空白项 current / other，
// 或选中的名字对不上菜单）。不设置信度门槛：模型选哪个就开哪个，不需要跳转时它会选 current。
func (r *Router) pickPage(ctx context.Context, res *Result, content string, page *Page) (string, string, error) {
	opts := newOptions()
	for i, item := range page.Nav {
		if i >= r.cfg.MaxPages {
			break
		}
		opts.add(item.Title, item.View, item.View, item.Desc)
	}
	opts.addBlank(optionCurrent, "不用打开新页面，当前页面「"+clip(page.Name(), 40)+"」已经够用", "")
	opts.addBlank(optionOther, "以上页面都不合适，或这句话与打开页面无关", "")

	answer, err := r.ask(ctx, map[string]any{
		"user_message": clip(content, 600),
		"current_page": map[string]any{"view": page.View, "title": clip(page.Name(), 40)},
	}, "page", "用户这句话需要先打开哪个页面（等价于点左侧菜单）？已经在合适的页面、或这句话与打开页面无关，就选 current。", opts)
	if err != nil {
		r.trace(res, Decision{Kind: "page", Answer: answer.Label(), Confidence: answer.Confidence,
			Options: len(opts.byName), Top: r.top(answer, opts), Error: err.Error()})
		return "", "", err
	}
	view := opts.lookup(answer.Label())
	r.trace(res, Decision{Kind: "page", Answer: answer.Label(), Target: view,
		Confidence: answer.Confidence, Margin: r.margin(answer, opts), Options: len(opts.byName), Top: r.top(answer, opts)})
	if view == "" {
		return "", r.blankReason("切换", answer.Label(), opts), nil
	}
	return view, "", nil
}

// pickAction 判断要不要点开当前页面上的一个入口，规则与 pickPage 相同。
func (r *Router) pickAction(ctx context.Context, res *Result, content string, page *Page) (string, string, error) {
	opts := newOptions()
	for i, item := range page.Actions {
		if i >= r.cfg.MaxActions {
			break
		}
		opts.add(item.Label, item.ID, item.ID, item.Desc)
	}
	opts.addBlank(optionNone, "不需要点开任何入口", "")
	opts.addBlank(optionOther, "以上入口都不合适，或这句话与点开入口无关", "")

	answer, err := r.ask(ctx, map[string]any{
		"user_message": clip(content, 600),
		"page":         map[string]any{"view": page.View, "title": clip(page.Name(), 40)},
	}, "entry", "要完成用户的诉求，是否需要先点开当前页面上的某个入口（按钮、弹窗、表单、抽屉）？不需要就选 none。", opts)
	if err != nil {
		r.trace(res, Decision{Kind: "entry", Answer: answer.Label(), Confidence: answer.Confidence,
			Options: len(opts.byName), Top: r.top(answer, opts), Error: err.Error()})
		return "", "", err
	}
	id := opts.lookup(answer.Label())
	r.trace(res, Decision{Kind: "entry", Answer: answer.Label(), Target: id,
		Confidence: answer.Confidence, Margin: r.margin(answer, opts), Options: len(opts.byName), Top: r.top(answer, opts)})
	if id == "" {
		return "", r.blankReason("点开", answer.Label(), opts), nil
	}
	return id, "", nil
}

// ask 发一道选择题。问题数只有 1 个：决策模型的延迟随问题数近线性增长，
// 两题分开问还能在中间用上刚打开的页面，比一次问完更准。
func (r *Router) ask(ctx context.Context, state any, id, instructions string, options *options) (decision.Answer, error) {
	resp, err := r.dec.AskRequest(ctx, &decision.Request{
		State:     state,
		Questions: map[string]decision.Question{id: decision.NewChoice(instructions, options.criteria())},
	})
	if err != nil {
		return decision.Answer{}, err
	}
	answer, err := resp.MustAnswer(id)
	if err != nil {
		return decision.Answer{}, err
	}
	return answer, nil
}

// observe 取当前页面快照
func (r *Router) observe(ctx context.Context, sessionID string) (*Page, error) {
	resp, err := r.ui(ctx, sessionID, &protocol.UIRequestData{Op: opObserve})
	if err != nil {
		return nil, err
	}
	if resp == nil || !resp.OK {
		return nil, uiError(resp)
	}
	page := parseSnapshot(resp.Snapshot, resp.Rev)
	if page == nil {
		return nil, errors.New("快照里没有可用的页面信息")
	}
	return page, nil
}

// navigate 打开目标页面，返回新页面快照
func (r *Router) navigate(ctx context.Context, sessionID, view string) (*Page, error) {
	resp, err := r.ui(ctx, sessionID, &protocol.UIRequestData{Op: opNavigate, View: view})
	if err != nil {
		return nil, err
	}
	if resp == nil || !resp.OK {
		return nil, uiError(resp)
	}
	page := parseSnapshot(resp.Snapshot, resp.Rev)
	if page == nil {
		// 页面已打开但没回快照：后续步骤不再依赖界面状态，只带个版本号
		return &Page{View: view, Rev: resp.Rev}, nil
	}
	return page, nil
}

// act 点开一个入口，返回执行后的页面快照
func (r *Router) act(ctx context.Context, sessionID, cmd string, rev int64) (*Page, error) {
	resp, err := r.ui(ctx, sessionID, &protocol.UIRequestData{Op: opAct, Cmd: cmd, Rev: rev})
	if err != nil {
		return nil, err
	}
	if resp == nil || !resp.OK {
		return nil, uiError(resp)
	}
	page := parseSnapshot(resp.Snapshot, resp.Rev)
	if page == nil {
		return &Page{Rev: resp.Rev}, nil
	}
	return page, nil
}

// ui 下发一次界面请求并记录结果：排查「页面没反应」时，这里是唯一的现场证据
func (r *Router) ui(ctx context.Context, sessionID string, req *protocol.UIRequestData) (*protocol.UIResponseData, error) {
	started := time.Now()
	resp, err := r.tr.Request(ctx, sessionID, req)
	outcome := "ok"
	switch {
	case err != nil:
		outcome = "失败: " + err.Error()
	case resp == nil:
		outcome = "失败: 界面没有返回任何结果"
	case !resp.OK:
		outcome = "页面拒绝: " + resp.Error
	}
	r.log.Info("preflow 界面请求",
		"session", sessionID, "op", req.Op, "view", req.View, "cmd", req.Cmd, "rev", req.Rev,
		"outcome", outcome, "snapshotBytes", snapshotLen(resp),
		"elapsed", time.Since(started).Round(time.Millisecond).String())
	return resp, err
}

// snapshotLen 安全取回执里的快照字节数：0 通常意味着前端没把页面结构回传，
// 那正是「决策说打开、后续却没动作」的常见原因。
func snapshotLen(resp *protocol.UIResponseData) int {
	if resp == nil {
		return 0
	}
	return len(resp.Snapshot)
}

// trace 记录一次判断并立刻打印：后面即使超时或界面拒绝，也能看出模型当时选了什么
func (r *Router) trace(res *Result, d Decision) {
	if res != nil {
		res.Decisions = append(res.Decisions, d)
	}
	r.log.Info("preflow 决策",
		"kind", d.Kind, "answer", d.Answer, "target", d.Target,
		"confidence", d.Confidence, "margin", d.Margin, "options", d.Options,
		"top", jsonValue(d.Top), "error", d.Error)
}

// options 选择题的选项表：选项名用菜单标题本身（语义化的名字比 p1/p2 好判），
// 并保留「选项名 -> 真实目标」的映射，模型回传什么名字都能对回去。
type options struct {
	byName    map[string]string // 选项名 -> view / 命令 id（空白项为空）
	crit      decision.ChoiceOptions
	normIndex map[string]string // 归一化选项名 -> 选项名
	blank     map[string]bool   // 哪些选项名表示「不用动界面」
}

func newOptions() *options {
	return &options{
		byName:    map[string]string{},
		crit:      decision.ChoiceOptions{},
		normIndex: map[string]string{},
		blank:     map[string]bool{},
	}
}

// add 加一个真实候选：name 缺失时退回 target 的编号，重名时补序号保证唯一
func (o *options) add(name, target, fallback, desc string) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fmt.Sprintf("%s%d", prefixOr(fallback, "x"), len(o.byName)+1)
	}
	// 撞名就带上序号：两个都叫「预订单」的菜单必须能分别选中
	if _, dup := o.byName[name]; dup {
		name = fmt.Sprintf("%s#%d", name, len(o.byName)+1)
	}
	o.byName[name] = target
	o.normIndex[normalizeName(name)] = name
	text := clip(desc, 80)
	if text == "" {
		text = "「" + name + "」"
	}
	o.crit[name] = text
}

// addBlank 加一个「不用动界面」的兜底选项
func (o *options) addBlank(name, desc, target string) {
	o.byName[name] = target
	o.normIndex[normalizeName(name)] = name
	o.crit[name] = desc
	o.blank[name] = true
}

// criteria 送给决策模型的「选项名 -> 描述」表
func (o *options) criteria() decision.ChoiceOptions { return o.crit }

// prefixOr 取首字母做编号前缀，view 为空时也有个能用的名字
func prefixOr(s, def string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	return string([]rune(s)[0])
}

// lookup 把模型回传的选项名对回真实目标；对不上就当作没选（宁可不跳转也不乱跳）
func (o *options) lookup(label string) string {
	label = strings.TrimSpace(label)
	if target, ok := o.byName[label]; ok {
		return target
	}
	if name, ok := o.normIndex[normalizeName(label)]; ok {
		return o.byName[name]
	}
	// 模型偶尔会加减几个字：只在唯一命中时兜底，多个候选相近就不猜
	name, ok := uniqueContain(o.normIndex, normalizeName(label))
	if !ok {
		return ""
	}
	return o.byName[name]
}

// top 概率最高的几项，调参和排查「为什么跳错页」全靠它
func (r *Router) top(answer decision.Answer, o *options) []Candidate {
	const limit = 3
	names := make([]string, 0, len(answer.Probabilities))
	for name := range answer.Probabilities {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		pi, pj := answer.Probabilities[names[i]], answer.Probabilities[names[j]]
		if pi != pj {
			return pi > pj
		}
		return names[i] < names[j]
	})
	if len(names) == 0 {
		return nil
	}
	if len(names) > limit {
		names = names[:limit]
	}
	out := make([]Candidate, 0, len(names))
	for _, name := range names {
		out = append(out, Candidate{Name: name, Target: o.byName[name], Prob: answer.Probabilities[name]})
	}
	return out
}

// margin 第一与第二的概率差；服务端没回概率表时为 0，不参与任何判断
func (r *Router) margin(answer decision.Answer, o *options) float64 {
	top := r.top(answer, o)
	if len(top) < 2 {
		return 0
	}
	return top[0].Prob - top[1].Prob
}

// blankReason 说清为什么没动界面：选了空白项，还是名字对不上
func (r *Router) blankReason(action, label string, o *options) string {
	name, ok := o.normIndex[normalizeName(label)]
	if ok && o.blank[name] {
		return fmt.Sprintf("模型判定不%s（选了 %s）", action, name)
	}
	return "选中的选项对不上菜单或入口：" + label
}

func normalizeName(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// uniqueContain 只返回唯一命中的一方，避免「预订单」同时命中好几个时瞎猜
func uniqueContain(index map[string]string, needle string) (string, bool) {
	if needle == "" {
		return "", false
	}
	var hit string
	for norm, name := range index {
		if strings.Contains(norm, needle) || strings.Contains(needle, norm) {
			if hit != "" {
				return "", false
			}
			hit = name
		}
	}
	return hit, hit != ""
}

// uiError 浏览器回执里的失败原因
func uiError(resp *protocol.UIResponseData) error {
	if resp == nil {
		return errors.New("界面没有返回任何结果")
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return errors.New("界面操作失败，且未返回原因")
}

// logResult 每次都打印：用户看到「页面没反应」时，这条日志就是答案本身
func (r *Router) logResult(req *Request, res *Result) {
	r.log.Info("preflow 界面前置路由",
		"session", req.SessionID, "reqID", req.ReqID, "message", clip(req.Content, 40),
		"applied", jsonValue(res.Steps), "pending", jsonValue(res.Pending),
		"decisions", jsonValue(res.Decisions), "note", res.Note,
		"reason", res.Reason, "elapsed", res.Elapsed.Round(time.Millisecond).String())
}

func appendReason(res *Result, reason string) {
	if res.Reason == "" {
		res.Reason = reason
		return
	}
	res.Reason += "; " + reason
}

// isCommand 斜杠命令（/new、/steer）不做界面路由：那是给运行时的指令，不是业务话
func isCommand(content string) bool {
	return strings.HasPrefix(content, "/")
}

func trimSpace(s string) string { return strings.TrimSpace(s) }

// jsonValue 把结构压成一行日志文本，失败时退回类型名
func jsonValue(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%T", v)
	}
	return string(raw)
}
