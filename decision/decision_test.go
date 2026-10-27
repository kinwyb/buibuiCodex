package decision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 文档「工单分流示例」的返回结果。
const sampleResponse = `{
  "model": "decision-model-preview",
  "request_id": "7b986c65-b223-9341-b5f0-b988e27ecaac",
  "answers": {
    "department": {"type": "choice", "choice": "billing", "confidence": 0.88,
      "probabilities": {"billing": 0.94, "technical": 0.06}},
    "escalate": {"type": "noul", "noul": 0.99},
    "severity": {"type": "score", "score": 2.25, "confidence": 0.91,
      "legend": {"0": "轻微问题，不影响功能", "1": "部分功能受影响，但存在替代方案",
        "2": "核心功能不可用，没有替代方案", "3": "造成严重业务或安全影响"},
      "probabilities": {"0": 0.0, "1": 0.01, "2": 0.73, "3": 0.26}}
  },
  "usage": {"input_tokens": 125},
  "latency_ms": 52.9
}`

// stub 用 RoundTripper 代替真实监听端口，方便在受限环境里跑测试。
type stub struct {
	handler http.HandlerFunc
	calls   int
}

func (s *stub) RoundTrip(r *http.Request) (*http.Response, error) {
	s.calls++
	rec := httptest.NewRecorder()
	s.handler(rec, r)
	return rec.Result(), nil
}

const stubBaseURL = "https://decision.example.com/compatible-mode/v1"

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *stub) {
	t.Helper()
	st := &stub{handler: handler}
	c, err := New(WithAPIKey("sk-test"), WithBaseURL(stubBaseURL), WithHTTPClient(&http.Client{Transport: st}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, st
}

func TestNormalizeBaseURL(t *testing.T) {
	cases := map[string]string{
		"https://ws.cn-beijing.maas.aliyuncs.com":                              "https://ws.cn-beijing.maas.aliyuncs.com" + APIPath,
		"https://ws.cn-beijing.maas.aliyuncs.com/":                             "https://ws.cn-beijing.maas.aliyuncs.com" + APIPath,
		"https://ws.cn-beijing.maas.aliyuncs.com/compatible-mode/v1":           "https://ws.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/systemone",
		"https://ws.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/systemone": "https://ws.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/systemone",
	}
	for in, want := range cases {
		if got := normalizeBaseURL(in); got != want {
			t.Errorf("normalizeBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewRequiresKeyAndBase(t *testing.T) {
	t.Setenv(APIKeyEnv, "")
	t.Setenv(BaseURLEnv, "")
	if _, err := New(WithBaseURL("https://x.example.com")); err == nil {
		t.Fatal("缺少 API Key 时应报错")
	}
	if _, err := New(WithAPIKey("k")); err == nil {
		t.Fatal("缺少接口地址时应报错")
	}
	c, err := New(WithAPIKey("k"), WithBaseURL("https://decision.example.com"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	want := "https://decision.example.com" + APIPath
	if c.Endpoint != want {
		t.Errorf("Endpoint = %q, want %q", c.Endpoint, want)
	}
	if c.Model != DefaultModel {
		t.Errorf("Model = %q, want %q", c.Model, DefaultModel)
	}
}

func TestNewRejectsBadAddress(t *testing.T) {
	// 占位符没替换、漏写 scheme，都应在创建时就报错，而不是等到发请求。
	for _, base := range []string{"https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com", "ws-1.cn-beijing.maas.aliyuncs.com"} {
		if _, err := New(WithAPIKey("k"), WithBaseURL(base)); err == nil {
			t.Errorf("地址 %q 应被拒绝", base)
		}
	}
}

func TestAskRequestAndAnswers(t *testing.T) {
	var gotPath, gotAuth, gotContentType string
	var gotBody Request
	c, st := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth, gotContentType = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Errorf("请求体不是合法 JSON: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, sampleResponse)
	})

	state := map[string]any{"ticket_id": "T-1001", "content": "订单支付后超过 24 小时仍未到账。"}
	resp, err := c.Ask(context.Background(), state, map[string]Question{
		"department": NewChoice("应该由哪个团队处理？", ChoiceOptions{
			"billing": "支付、退款和账单问题", "technical": "产品故障和集成问题"}),
		"escalate": NewYesNo("是否需要立即通知值班人员？"),
		"severity": NewScore("这个问题有多严重？",
			"轻微问题，不影响功能", "部分功能受影响，但存在替代方案",
			"核心功能不可用，没有替代方案", "造成严重业务或安全影响"),
	})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if st.calls != 1 {
		t.Fatalf("calls = %d, want 1", st.calls)
	}
	if gotPath != APIPath {
		t.Errorf("path = %q, want %q", gotPath, APIPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if gotBody.Model != DefaultModel {
		t.Errorf("请求 model = %q", gotBody.Model)
	}
	if len(gotBody.Questions) != 3 || gotBody.Questions["escalate"].Type != TypeNoul {
		t.Errorf("请求 questions 不符: %+v", gotBody.Questions)
	}
	if _, ok := gotBody.State.(map[string]any); !ok {
		t.Errorf("请求 state 应为对象，实际 %T", gotBody.State)
	}

	if resp.RequestID != "7b986c65-b223-9341-b5f0-b988e27ecaac" {
		t.Errorf("RequestID = %q", resp.RequestID)
	}
	if resp.Usage.InputTokens != 125 || resp.LatencyMS != 52.9 {
		t.Errorf("usage/latency = %+v / %v", resp.Usage, resp.LatencyMS)
	}

	dept := resp.Answer("department")
	if got := dept.Label(); got != "billing" {
		t.Errorf("department label = %q", got)
	}
	if dept.Confidence != 0.88 || dept.Probability("billing") != 0.94 {
		t.Errorf("department confidence/prob = %v/%v", dept.Confidence, dept.Probability("billing"))
	}
	if top, p := dept.TopProbability(); top != "billing" || p != 0.94 {
		t.Errorf("department top = %q/%v", top, p)
	}

	esc := resp.Answer("escalate")
	if !esc.Yes() || esc.YesProbability() != 0.99 {
		t.Errorf("escalate = %v/%v", esc.Yes(), esc.YesProbability())
	}

	sev := resp.Answer("severity")
	if sev.Value() != 2.25 || sev.Level() != 2 {
		t.Errorf("severity value/level = %v/%d", sev.Value(), sev.Level())
	}
	if sev.LevelText() != "核心功能不可用，没有替代方案" {
		t.Errorf("severity text = %q", sev.LevelText())
	}
	if sev.LevelTextAt(3) != "造成严重业务或安全影响" {
		t.Errorf("severity text(3) = %q", sev.LevelTextAt(3))
	}
	if got, want := sev.String(), "score=2.25 level=2(核心功能不可用，没有替代方案) confidence=0.91"; got != want {
		t.Errorf("Answer.String() = %q, want %q", got, want)
	}

	if _, err := resp.MustAnswer("nope"); err == nil {
		t.Error("MustAnswer 未知问题应报错")
	}
}

func TestTaskBuilder(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, sampleResponse)
	})
	task := NewTask("这段评论是否在索要退款？").
		Choice("department", "应该由哪个团队处理？", ChoiceOptions{"billing": "账单"}).
		YesNo("escalate", "是否需要通知值班？").
		Score("severity", "有多严重？", "轻微", "中等", "严重").
		Add("extra", NewYesNoWith("是否包含手机号？", "包含", "不包含"))

	if ids := task.IDs(); len(ids) != 4 || ids[0] != "department" || ids[3] != "severity" {
		t.Fatalf("IDs = %v", ids)
	}
	resp, err := task.Ask(context.Background(), c)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if len(resp.Answers) != 3 {
		t.Fatalf("answers = %d 项", len(resp.Answers))
	}
	if resp.Answer("extra").Type != "" {
		t.Error("未返回的问题应为零值")
	}
}

func TestSingleShotHelpers(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(raw, &req)
		q, ok := req.Questions[singleID]
		if !ok {
			t.Errorf("单题 id 应为 %q，实际 %v", singleID, req.Questions)
		}
		var body string
		switch q.Type {
		case TypeChoice:
			body = `{"answers":{"decision":{"type":"choice","choice":"spam","confidence":0.77,"probabilities":{"spam":0.77,"ham":0.23}}}}`
		case TypeNoul:
			body = `{"answers":{"decision":{"type":"noul","noul":0.31}}}`
		case TypeScore:
			body = `{"answers":{"decision":{"type":"score","score":1.43,"confidence":0.9,"legend":{"0":"低","1":"中","2":"高"}}}}`
		}
		io.WriteString(w, body)
	})

	if label, conf, err := c.Classify(context.Background(), "买一发三，限时秒杀", "是否垃圾内容？",
		ChoiceOptions{"spam": "营销或垃圾", "ham": "正常"}); err != nil || label != "spam" || conf != 0.77 {
		t.Errorf("Classify = %q/%v/%v", label, conf, err)
	}
	if yes, p, err := c.Judge(context.Background(), "我要退货退款", "是否在索要退款？"); err != nil || yes || p != 0.31 {
		t.Errorf("Judge = %v/%v/%v", yes, p, err)
	}
	a, err := c.Rate(context.Background(), "工单内容", "严重度？", "低", "中", "高")
	if err != nil {
		t.Fatalf("Rate: %v", err)
	}
	if a.Value() != 1.43 || a.Level() != 1 || a.LevelText() != "中" {
		t.Errorf("Rate answer = %s", a)
	}
}

func TestRequestValidation(t *testing.T) {
	state := "内容"
	cases := map[string]*Request{
		"空 state":     {Questions: map[string]Question{"a": NewYesNo("q")}},
		"空 questions": {State: state},
		"未知类型":        {State: state, Questions: map[string]Question{"a": {Type: "other"}}},
		"choice 类型错":  {State: state, Questions: map[string]Question{"a": {Type: TypeChoice, Criteria: ScoreLevels{"x"}}}},
		"choice 无选项":  {State: state, Questions: map[string]Question{"a": NewChoice("q", ChoiceOptions{})}},
		"score 等级过少":  {State: state, Questions: map[string]Question{"a": NewScore("q", "只有这级")}},
		"问题 id 为空":    {State: state, Questions: map[string]Question{" ": NewYesNo("q")}},
	}
	for name, req := range cases {
		if _, err := (&Client{APIKey: "k", Endpoint: "http://x", Model: DefaultModel, HTTPClient: http.DefaultClient}).AskRequest(context.Background(), req); err == nil {
			t.Errorf("%s: 应校验失败", name)
		}
	}

	// 问题数上限。
	tooMany := make(map[string]Question, MaxQuestions+1)
	for i := 0; i <= MaxQuestions; i++ {
		tooMany[fmt.Sprintf("q%d", i)] = NewYesNo("q")
	}
	err := (&Request{State: state, Questions: tooMany}).Validate()
	if err == nil {
		t.Fatal("问题数超限应报错")
	}

	// criteria 为 nil 的 noul 合法。
	okReq := &Request{State: state, Questions: map[string]Question{"a": NewYesNo("q")}}
	if err := okReq.Validate(); err != nil {
		t.Fatalf("合法请求报错: %v", err)
	}
	if okReq.Model != DefaultModel {
		t.Errorf("Validate 应补齐模型名，实际 %q", okReq.Model)
	}
}

func TestAPIError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":{"type":"authentication_error","message":"Invalid API-key provided."}}`)
	})
	_, err := c.Ask(context.Background(), "state", map[string]Question{"a": NewYesNo("q")})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("应返回 *APIError，实际 %T: %v", err, err)
	}
	if apiErr.Status != http.StatusUnauthorized || apiErr.Code != "authentication_error" {
		t.Errorf("APIError = %+v", apiErr)
	}
	if apiErr.Message == "" {
		t.Error("应解析出 message")
	}
	if apiErr.IsRateLimit() {
		t.Error("401 不是限流")
	}
}

func TestAPIErrorDashScopeStyle(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-429")
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"code":"Throttling.RateQuota","message":"Requests rate limit exceeded","request_id":"req-1"}`)
	})
	_, err := c.Ask(context.Background(), "state", map[string]Question{"a": NewYesNo("q")})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("应返回 *APIError，实际 %T", err)
	}
	if !apiErr.IsRateLimit() {
		t.Errorf("应识别为限流: %+v", apiErr)
	}
	if apiErr.RequestID != "req-1" {
		t.Errorf("RequestID = %q", apiErr.RequestID)
	}
}

func TestBadJSONResponse(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "not json")
	})
	if _, err := c.Ask(context.Background(), "state", map[string]Question{"a": NewYesNo("q")}); err == nil {
		t.Fatal("非法响应体应报错")
	}
}
