package decision_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/kinwyb/buibuiCodex/decision"
)

// stubTransport 按问题类型返回固定答案，让示例离线可跑且输出确定。
// 真实调用不需要它，去掉 WithHTTPClient 即可走网络。
type stubTransport struct{}

func (stubTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	raw, _ := io.ReadAll(r.Body)
	var req decision.Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}

	answers := map[string]map[string]any{}
	for id, q := range req.Questions {
		switch q.Type {
		case decision.TypeChoice:
			answers[id] = map[string]any{
				"type":          "choice",
				"choice":        "billing",
				"confidence":    0.88,
				"probabilities": map[string]float64{"billing": 0.94, "technical": 0.06},
			}
		case decision.TypeNoul:
			answers[id] = map[string]any{"type": "noul", "noul": 0.99}
		case decision.TypeScore:
			answers[id] = map[string]any{
				"type":          "score",
				"score":         2.25,
				"confidence":    0.91,
				"legend":        map[string]string{"0": "轻微", "1": "有影响", "2": "核心不可用", "3": "严重业务影响"},
				"probabilities": map[string]float64{"0": 0.0, "1": 0.01, "2": 0.73, "3": 0.26},
			}
		}
	}

	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rec).Encode(map[string]any{
		"model":      decision.DefaultModel,
		"request_id": "7b986c65-b223-9341-b5f0-b988e27ecaac",
		"answers":    answers,
		"usage":      map[string]int{"input_tokens": 125},
		"latency_ms": 52.9,
	}); err != nil {
		return nil, err
	}
	return rec.Result(), nil
}

func newExampleClient() *decision.Client {
	c, err := decision.New(
		decision.WithAPIKey("sk-example"),
		decision.WithBaseURL("https://ws-example.cn-beijing.maas.aliyuncs.com"),
		decision.WithHTTPClient(&http.Client{Transport: stubTransport{}}),
	)
	if err != nil {
		panic(err)
	}
	return c
}

// 一次请求同时完成「派单团队 choice」「是否升级 noul」「严重度 score」三项决策。
func Example_multiQuestion() {
	state := map[string]any{
		"ticket_id": "T-1001",
		"content":   "订单支付后超过 24 小时仍未到账，用户无法继续使用核心服务，要求立即处理。",
	}

	resp, err := decision.NewTask(state).
		Choice("department", "应该由哪个团队处理？", decision.ChoiceOptions{
			"billing":   "支付、退款和账单问题",
			"technical": "产品故障和集成问题",
		}).
		YesNo("escalate", "是否需要立即通知值班人员？").
		Score("severity", "这个问题有多严重？", "轻微", "有影响", "核心不可用", "严重业务影响").
		Ask(context.Background(), newExampleClient())
	if err != nil {
		fmt.Println(err)
		return
	}

	dept := resp.Answer("department")
	escalate := resp.Answer("escalate")
	severity := resp.Answer("severity")

	fmt.Printf("%s %.0f%%\n", dept.Label(), dept.Confidence*100)
	fmt.Printf("escalate=%v p_yes=%.2f\n", escalate.Yes(), escalate.YesProbability())
	fmt.Printf("severity=%.2f level=%d(%s)\n", severity.Value(), severity.Level(), severity.LevelText())
	fmt.Printf("tokens=%d latency=%.1fms\n", resp.Usage.InputTokens, resp.LatencyMS)
	// Output:
	// billing 88%
	// escalate=true p_yes=0.99
	// severity=2.25 level=2(核心不可用)
	// tokens=125 latency=52.9ms
}

// 单题快捷方法适合审核、校验这类只关心一个判断的场景。
func ExampleClient_Judge() {
	yes, p, err := newExampleClient().Judge(context.Background(),
		"上周买的到现在还没发货，赶紧给我退款，不然投诉了。",
		"这段评论是否在索要退款？")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("需要退款诉求=%v P(yes)=%.2f\n", yes, p)
	// Output:
	// 需要退款诉求=true P(yes)=0.99
}
