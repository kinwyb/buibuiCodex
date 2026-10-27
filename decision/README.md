# decision — 百炼决策模型 API

封装 `POST /compatible-mode/v1/systemone`（模型 `decision-model-preview`）。
一次请求携带业务 `state` 与若干 typed 问题，一次前向返回判定结果与概率分布，**不生成文本**，
因此延迟与成本和输出长度无关，适合工单分流、内容审核、Agent 路由与结果校验。

文档：<https://docs.bailian.console.aliyun.com/zh/model-studio/decision-model-api>

## 三类问题

| 类型 | 含义 | criteria | 返回 |
| --- | --- | --- | --- |
| `choice` | 多选一 | 选项名 → 描述（1-255，建议补 `other` 兜底） | `choice` + `probabilities` + `confidence` |
| `noul` | 是否判断 | 可选 `{"true":…,"false":…}` | `noul` = P(yes)，0=否 1=是 |
| `score` | 有序量表 | 从低到高的等级描述（2-255，建议 3-7 级） | `score` 期望值 + `legend` + `probabilities` + `confidence` |

限制：问题数 ≤ 16（延迟随问题数近线性增长），上下文 ≤ 65536 token，超长 `state` 会被服务端拒绝。

## 创建客户端

```go
client, err := decision.New(
    decision.WithAPIKey(os.Getenv("DASHSCOPE_API_KEY")),
    decision.WithBaseURL("https://<your-host>/compatible-mode/v1/systemone"),
)
```

地址直接填控制台给出的完整接口地址，本包不参与业务空间与域名的拼装（地域对照见文档「地域与域名」）；
只填到域名时会自动补上 `/compatible-mode/v1/systemone`。
配置留空时回落到环境变量 `DASHSCOPE_API_KEY` / `DASHSCOPE_BASE_URL`。
`WithHTTPClient` 可复用连接池或注入测试桩。

## 一次问多个问题

```go
state := map[string]any{
    "ticket_id": "T-1001",
    "content":   "订单支付后超过 24 小时仍未到账，要求立即处理。",
}

resp, err := decision.NewTask(state).
    Choice("department", "应该由哪个团队处理？", decision.ChoiceOptions{
        "billing":   "支付、退款和账单问题",
        "technical": "产品故障和集成问题",
    }).
    YesNo("escalate", "是否需要立即通知值班人员？").
    Score("severity", "这个问题有多严重？",
        "轻微问题，不影响功能",
        "部分功能受影响，但存在替代方案",
        "核心功能不可用，没有替代方案",
        "造成严重业务或安全影响").
    Ask(ctx, client)
if err != nil {
    return err
}

dept := resp.Answer("department")
dept.Label()          // "billing"
dept.Confidence       // 0.88
resp.Answer("escalate").Yes()          // P(yes) >= 0.5
sev := resp.Answer("severity")
sev.Value()           // 2.25 概率加权期望，可落在两级之间
sev.Level()           // 2 四舍五入后的等级索引
sev.LevelText()       // "核心功能不可用，没有替代方案"
resp.Usage.InputTokens
resp.LatencyMS
```

问题 id 由调用方自定义，`Answers` 的 key 与之一一对应。需要精确阈值时用
`YesProbability()`、`Probability(key)`、`TopProbability()`，`MustAnswer(id)` 在缺失答案时报错。

## 单题快捷方法

```go
team, conf, err := client.Classify(ctx, ticketText, "应该由哪个团队处理？",
    decision.ChoiceOptions{"billing": "账单", "technical": "故障"})

needEscalate, p, err := client.Judge(ctx, ticketText, "是否需要立即通知值班人员？")

a, err := client.Rate(ctx, ticketText, "这个问题有多严重？", "轻微", "有影响", "核心不可用", "严重")
```

## 错误处理

非 2xx 返回 `*APIError`（含 `Status`/`Code`/`Message`/`RequestID`/`Body`），
兼容 OpenAI 与 DashScope 两种错误体；`IsRateLimit()` 用于退避重试。
本地校验（问题数、选项数、criteria 类型、空 state）在发出请求前完成，不消耗额度。

```go
var apiErr *decision.APIError
if errors.As(err, &apiErr) && apiErr.IsRateLimit() {
    // 指数退避后重试
}
```

## 验证

```bash
go test ./decision/          # 测试用 RoundTripper 桩，不需要网络和端口
go vet ./decision/
```
