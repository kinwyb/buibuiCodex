package preflow

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
	"github.com/kinwyb/buibuiCodex/decision"
)

// fakeTransport 按 op 依次吐出预设回执，并记录服务端下发的请求
type fakeTransport struct {
	replies map[string][]*protocol.UIResponseData
	reqs    []*protocol.UIRequestData
	err     error
}

func (f *fakeTransport) Request(_ context.Context, _ string, req *protocol.UIRequestData) (*protocol.UIResponseData, error) {
	f.reqs = append(f.reqs, req)
	if f.err != nil {
		return nil, f.err
	}
	queue := f.replies[req.Op]
	if len(queue) == 0 {
		return nil, errors.New("没有预设回执: " + req.Op)
	}
	resp := queue[0]
	f.replies[req.Op] = queue[1:]
	return resp, nil
}

func (f *fakeTransport) ops() []string {
	ops := make([]string, 0, len(f.reqs))
	for _, req := range f.reqs {
		ops = append(ops, req.Op)
	}
	return ops
}

// fakeDecider 按调用顺序给出选项题答案，同时记下送给模型的请求体
type fakeDecider struct {
	picks []choice
	reqs  []*decision.Request
	err   error
}

type choice struct {
	answer     string
	confidence float64
}

func (f *fakeDecider) AskRequest(_ context.Context, req *decision.Request) (*decision.Response, error) {
	f.reqs = append(f.reqs, req)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.picks) == 0 {
		return nil, errors.New("决策答案不够用了")
	}
	p := f.picks[0]
	f.picks = f.picks[1:]

	var id string
	for k := range req.Questions {
		id = k
	}
	return &decision.Response{Answers: map[string]decision.Answer{
		id: {Type: decision.TypeChoice, Choice: p.answer, Confidence: p.confidence},
	}}, nil
}

func snapshotJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}

func okResp(rev int64, snap any) *protocol.UIResponseData {
	return &protocol.UIResponseData{OK: true, Rev: rev, Snapshot: snapshotJSON(snap)}
}

var homeSnapshot = map[string]any{
	"view": "Home", "title": "工作台", "rev": 1,
	"navigation": []any{
		map[string]any{"view": "OrderList", "title": "生产工单"},
		map[string]any{"view": "YieldReport", "title": "良率报表"},
	},
}

var orderSnapshot = map[string]any{
	"view": "OrderList", "title": "生产工单", "rev": 7,
	"actions": []any{
		map[string]any{"id": "create", "label": "新建工单"},
		map[string]any{"id": "detail", "label": "查看明细", "params": map[string]any{"orderId": "string"}},
	},
}

func newRouter(cfg Config, tr Transport, dec Decider) *Router {
	return New(cfg, dec, tr, nil)
}

func enabledConfig() Config {
	return Config{Enabled: true, Timeout: time.Second, DeepAction: true, Annotate: true}
}

func TestRouteNavigateThenAct(t *testing.T) {
	tr := &fakeTransport{replies: map[string][]*protocol.UIResponseData{
		opObserve:  {okResp(1, homeSnapshot)},
		opNavigate: {okResp(7, orderSnapshot)},
		opAct:      {okResp(8, map[string]any{"view": "OrderList", "title": "生产工单"})},
	}}
	dec := &fakeDecider{picks: []choice{{"生产工单", 0.36}, {"新建工单", 0.52}}}

	res := newRouter(enabledConfig(), tr, dec).Route(context.Background(), &Request{
		ReqID: "r1", SessionID: "ws_web_c1", Content: "帮我建个工单，二号线停机了",
	})

	if got, want := strings.Join(tr.ops(), ","), "observe,navigate,act"; got != want {
		t.Fatalf("界面请求序列 = %s, want %s", got, want)
	}
	if len(res.Steps) != 2 {
		t.Fatalf("应有两步动作: %+v", res.Steps)
	}
	if res.Steps[0] != (Step{Kind: opNavigate, Target: "OrderList", Label: "生产工单", Rev: 7}) {
		t.Errorf("navigate 步骤 = %+v", res.Steps[0])
	}
	if res.Steps[1].Target != "create" || res.Steps[1].Label != "新建工单" || res.Steps[1].Rev != 8 {
		t.Errorf("act 步骤 = %+v", res.Steps[1])
	}
	if !res.Applied() {
		t.Fatal("Applied 应为 true")
	}
	for _, want := range []string{"已打开页面", "生产工单", "已点开", "新建工单", "rev=8", "请勿重复 ui_navigate"} {
		if !strings.Contains(res.Note, want) {
			t.Errorf("Note 应包含 %q，实际 %q", want, res.Note)
		}
	}

	// navigate 用的是快照里的 view，不是选项名
	if got := tr.reqs[1].View; got != "OrderList" {
		t.Errorf("navigate view = %q", got)
	}
	// act 带上了打开后页面的版本号
	if got := tr.reqs[2]; got.Cmd != "create" || got.Rev != 7 {
		t.Errorf("act 请求 = %+v", got)
	}
}

func TestRouteStaysOnCurrentPage(t *testing.T) {
	tr := &fakeTransport{replies: map[string][]*protocol.UIResponseData{
		opObserve: {okResp(1, homeSnapshot)},
	}}
	dec := &fakeDecider{picks: []choice{{optionCurrent, 0.44}}}

	res := newRouter(enabledConfig(), tr, dec).Route(context.Background(), &Request{
		ReqID: "r1", SessionID: "s", Content: "今天天气怎么样",
	})

	if len(res.Steps) != 0 || res.Note != "" {
		t.Fatalf("不该动界面: steps=%+v note=%q", res.Steps, res.Note)
	}
	if got, want := strings.Join(tr.ops(), ","), "observe"; got != want {
		t.Errorf("界面请求 = %s, want %s", got, want)
	}
	if !strings.Contains(res.Reason, "模型判定不切换（选了 current）") {
		t.Errorf("Reason = %q", res.Reason)
	}
}

// 模型选空白项（other / current）就不动界面：没有置信度门槛，选谁就照做
func TestRouteFollowsBlankChoice(t *testing.T) {
	tr := &fakeTransport{replies: map[string][]*protocol.UIResponseData{
		opObserve: {okResp(1, homeSnapshot)},
	}}
	dec := &fakeDecider{picks: []choice{{optionOther, 0.51}}}

	res := newRouter(enabledConfig(), tr, dec).Route(context.Background(), &Request{SessionID: "s", Content: "看下良率"})
	if len(res.Steps) != 0 {
		t.Fatalf("选了 other 不该动界面: %+v", res.Steps)
	}
	if !strings.Contains(res.Reason, "选了 other") {
		t.Errorf("Reason = %q", res.Reason)
	}
	if got, want := strings.Join(tr.ops(), ","), "observe"; got != want {
		t.Errorf("不该有 navigate: %s", got)
	}
}

// 低置信度不再拦动作：52 个菜单里 top1 只有 0.36 是常态，选了就照做
func TestRouteActsOnLowConfidenceChoice(t *testing.T) {
	tr := &fakeTransport{replies: map[string][]*protocol.UIResponseData{
		opObserve:  {okResp(1, homeSnapshot)},
		opNavigate: {okResp(7, orderSnapshot)},
	}}
	dec := &fakeDecider{picks: []choice{{"生产工单", 0.36}, {optionNone, 0.6}}}

	res := newRouter(enabledConfig(), tr, dec).Route(context.Background(), &Request{
		SessionID: "s", Content: "我要查看越南的预订单",
	})
	if len(res.Steps) != 1 || res.Steps[0].Target != "OrderList" {
		t.Fatalf("置信度 0.36 也要按选择执行: %+v reason=%q", res.Steps, res.Reason)
	}
}

// 选项名对不上菜单时不猜：宁可不跳转
func TestRouteIgnoresUnknownOption(t *testing.T) {
	tr := &fakeTransport{replies: map[string][]*protocol.UIResponseData{
		opObserve: {okResp(1, homeSnapshot)},
	}}
	dec := &fakeDecider{picks: []choice{{"不存在的菜单", 0.9}}}

	res := newRouter(enabledConfig(), tr, dec).Route(context.Background(), &Request{SessionID: "s", Content: "开个页面"})
	if len(res.Steps) != 0 || !strings.Contains(res.Reason, "对不上菜单") {
		t.Fatalf("对不上就不动: %+v reason=%q", res.Steps, res.Reason)
	}
}

func TestRouteHaltsWhenNavigateFails(t *testing.T) {
	tr := &fakeTransport{replies: map[string][]*protocol.UIResponseData{
		opObserve:  {okResp(1, homeSnapshot)},
		opNavigate: {{OK: false, Error: "页面不存在"}},
	}}
	dec := &fakeDecider{picks: []choice{{"生产工单", 0.7}}}

	res := newRouter(enabledConfig(), tr, dec).Route(context.Background(), &Request{SessionID: "s", Content: "建工单"})
	if len(res.Steps) != 0 || res.Note != "" {
		t.Fatalf("navigate 失败不该继续: %+v %q", res.Steps, res.Note)
	}
	if !strings.Contains(res.Reason, "页面不存在") {
		t.Errorf("Reason = %q", res.Reason)
	}
	// 关键是别在原来的首页上继续点入口
	if got, want := strings.Join(tr.ops(), ","), "observe,navigate"; got != want {
		t.Errorf("界面请求 = %s, want %s", got, want)
	}
}

func TestRoutePendingActionNeedsParams(t *testing.T) {
	tr := &fakeTransport{replies: map[string][]*protocol.UIResponseData{
		opObserve: {okResp(7, orderSnapshot)},
	}}
	// 快照里没有菜单，所以只会问一次「要不要点开入口」
	dec := &fakeDecider{picks: []choice{{"查看明细", 0.77}}}

	res := newRouter(enabledConfig(), tr, dec).Route(context.Background(), &Request{SessionID: "s", Content: "看下 12345 号工单的明细"})
	if len(res.Steps) != 0 {
		t.Fatalf("需要参数的入口不该代办: %+v", res.Steps)
	}
	if len(res.Pending) != 1 || res.Pending[0].Target != "detail" {
		t.Fatalf("Pending = %+v", res.Pending)
	}
	if !strings.Contains(res.Note, "需要业务参数") || !strings.Contains(res.Note, "查看明细") {
		t.Errorf("Note 应把待办交给 agent: %q", res.Note)
	}
	if got, want := strings.Join(tr.ops(), ","), "observe"; got != want {
		t.Errorf("不该下发 act: %s", got)
	}
}

func TestRouteSkipsSilently(t *testing.T) {
	cases := map[string]struct {
		cfg     Config
		req     *Request
		noTr    bool
		noDec   bool
		nilSelf bool
	}{
		"未启用":      {Config{Timeout: time.Second}, &Request{SessionID: "s", Content: "建工单"}, false, false, false},
		"无决策器":     {enabledConfig(), &Request{SessionID: "s", Content: "建工单"}, false, true, false},
		"无界面通道":    {enabledConfig(), &Request{SessionID: "s", Content: "建工单"}, true, false, false},
		"空内容":      {enabledConfig(), &Request{SessionID: "s"}, false, false, false},
		"斜杠命令":     {enabledConfig(), &Request{SessionID: "s", Content: "/new 建工单"}, false, false, false},
		"无会话":      {enabledConfig(), &Request{Content: "建工单"}, false, false, false},
		"空请求":      {enabledConfig(), nil, false, false, false},
		"Router为空": {enabledConfig(), &Request{SessionID: "s", Content: "建工单"}, false, false, true},
	}
	for name, tc := range cases {
		tr := &fakeTransport{}
		dec := &fakeDecider{picks: []choice{{"生产工单", 0.99}}}
		var transport Transport
		if !tc.noTr {
			transport = tr
		}
		var decider Decider
		if !tc.noDec {
			decider = dec
		}
		r := New(tc.cfg, decider, transport, nil)
		if tc.nilSelf {
			r = nil
		}

		res := r.Route(context.Background(), tc.req)
		if res == nil {
			t.Fatalf("%s: Route 不该返回 nil", name)
		}
		if len(res.Steps) != 0 || len(res.Pending) != 0 || res.Note != "" {
			t.Errorf("%s: 不该有动作或提示: %+v %q", name, res.Steps, res.Note)
		}
		if len(tr.reqs) != 0 {
			t.Errorf("%s: 不该发起界面请求: %v", name, tr.ops())
		}
		if len(dec.reqs) != 0 {
			t.Errorf("%s: 不该调用决策模型", name)
		}
	}
}

func TestRouteTransportUnavailable(t *testing.T) {
	tr := &fakeTransport{err: errors.New("no ui host connected for session")}
	dec := &fakeDecider{picks: []choice{{"生产工单", 0.7}}}

	res := newRouter(enabledConfig(), tr, dec).Route(context.Background(), &Request{SessionID: "s", Content: "建工单"})
	if len(res.Steps) != 0 || res.Note != "" {
		t.Fatalf("页面不可用时不该有动作: %+v %q", res.Steps, res.Note)
	}
	if !strings.Contains(res.Reason, "取页面快照失败") {
		t.Errorf("Reason = %q", res.Reason)
	}
	if len(dec.reqs) != 0 {
		t.Error("页面都取不到，不应该再花决策模型的额度")
	}
}

func TestRouteSnapshotWithoutRoutes(t *testing.T) {
	tr := &fakeTransport{replies: map[string][]*protocol.UIResponseData{
		opObserve: {okResp(3, map[string]any{"view": "Settings", "title": "系统设置"})},
	}}
	dec := &fakeDecider{}

	res := newRouter(enabledConfig(), tr, dec).Route(context.Background(), &Request{SessionID: "s", Content: "建工单"})
	if len(res.Steps) != 0 {
		t.Fatalf("既没有菜单也没有入口: %+v", res.Steps)
	}
	if len(dec.reqs) != 0 {
		t.Error("无可选项时不该调用决策模型")
	}
}

func TestRouteBudgetStopsWork(t *testing.T) {
	tr := &slowTransport{delay: 40 * time.Millisecond}
	dec := &fakeDecider{picks: []choice{{"生产工单", 0.7}}}
	cfg := Config{Enabled: true, Timeout: 10 * time.Millisecond, DeepAction: true, Annotate: true}

	started := time.Now()
	res := newRouter(cfg, tr, dec).Route(context.Background(), &Request{SessionID: "s", Content: "建工单"})
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("超出总预算仍在等待: %s", elapsed)
	}
	if len(res.Steps) != 0 || !strings.Contains(res.Reason, "取页面快照失败") {
		t.Errorf("应按超时降级: %+v %q", res.Steps, res.Reason)
	}
}

type slowTransport struct{ delay time.Duration }

func (s *slowTransport) Request(ctx context.Context, _ string, _ *protocol.UIRequestData) (*protocol.UIResponseData, error) {
	select {
	case <-time.After(s.delay):
		return okResp(1, homeSnapshot), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// 选项表现在是唯一的「模型答案 -> 真实目标」通道：重名必须能分别选中，
// 模型加减空格或标点也要能对回去，含糊不清时宁可不跳。
func TestOptionsLookup(t *testing.T) {
	o := newOptions()
	o.add("预订单", "V1", "V1", "销售管理")
	o.add("预订单", "V2", "V2", "")
	o.add("", "VEmpty", "VEmpty", "")
	o.addBlank(optionCurrent, "当前页够用", "")

	if got := o.lookup("预订单"); got != "V1" {
		t.Errorf("重名时第一个应可原样选中，实际 %q", got)
	}
	// 第二个「预订单」拿到带序号的名字，两者都能单独选中；没有标题的菜单退回 view 编号
	if got := o.lookup("预订单#2"); got != "V2" {
		t.Errorf("重名的第二个应映射到 V2，实际 %q（选项=%v）", got, o.crit)
	}
	if got := o.lookup("V3"); got != "VEmpty" {
		t.Errorf("空标题应退回 view 编号，实际 %q", got)
	}
	if got := o.crit["预订单"]; got != "销售管理" {
		t.Errorf("菜单说明要进选项描述，实际 %q", got)
	}
	// 空格与全半角差异要能容错
	if got := o.lookup("  预订单  "); got != "V1" {
		t.Errorf("带空格的回传应对得上，实际 %q", got)
	}
	// 空白项没有目标
	if got := o.lookup(optionCurrent); got != "" {
		t.Errorf("current 不该有目标，实际 %q", got)
	}
	// 多个候选都能匹配时不猜
	if got := o.lookup("订"); got != "" {
		t.Errorf("歧义时不该返回目标，实际 %q", got)
	}
	if got := o.lookup("根本不存在的菜单"); got != "" {
		t.Errorf("对不上时不该返回目标，实际 %q", got)
	}
}

func TestRouteStateSentToModel(t *testing.T) {
	tr := &fakeTransport{replies: map[string][]*protocol.UIResponseData{
		opObserve:  {okResp(1, homeSnapshot)},
		opNavigate: {okResp(7, orderSnapshot)},
	}}
	dec := &fakeDecider{picks: []choice{{"生产工单", 0.9}, {optionNone, 0.9}}}

	newRouter(enabledConfig(), tr, dec).Route(context.Background(), &Request{
		SessionID: "s", Content: "帮我建个工单",
	})
	if len(dec.reqs) != 2 {
		t.Fatalf("应发起两次决策，实际 %d", len(dec.reqs))
	}
	state, ok := dec.reqs[0].State.(map[string]any)
	if !ok {
		t.Fatalf("state 结构 = %T", dec.reqs[0].State)
	}
	if state["user_message"] != "帮我建个工单" {
		t.Errorf("state 应带上用户原话: %v", state["user_message"])
	}
	q := dec.reqs[0].Questions["page"]
	if q.Type != decision.TypeChoice {
		t.Fatalf("页面判断应是选择题: %+v", q)
	}
	options, ok := q.Criteria.(decision.ChoiceOptions)
	if !ok {
		t.Fatalf("criteria 类型 = %T", q.Criteria)
	}
	// 选项名就是菜单标题本身（语义化比 p1/p2 好判），外加 current / other 两个空白项
	if _, ok := options["生产工单"]; !ok || options[optionCurrent] == "" || options[optionOther] == "" {
		t.Errorf("选项应覆盖菜单标题并带兜底: %+v", options)
	}
	if strings.Contains(options["生产工单"], "V0189") {
		t.Errorf("不该把无意义的 view 编号塞进选项描述: %q", options["生产工单"])
	}
	if len(dec.reqs[1].Questions["entry"].Criteria.(decision.ChoiceOptions)) != len(orderSnapshot["actions"].([]any))+2 {
		t.Error("入口选项数应与快照一致（外加两个兜底）")
	}
}
