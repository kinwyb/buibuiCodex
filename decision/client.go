package decision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// 接口地址与环境变量。
const (
	// APIPath 决策模型接口路径。
	APIPath = "/compatible-mode/v1/systemone"

	APIKeyEnv      = "DASHSCOPE_API_KEY"
	BaseURLEnv     = "DASHSCOPE_BASE_URL"
	WorkspaceIDEnv = "DASHSCOPE_WORKSPACE_ID"
	RegionEnv      = "DASHSCOPE_REGION"

	RegionBeijing   = "cn-beijing"
	RegionSingapore = "ap-southeast-1"

	// singleID 单题快捷方法使用的问题 id。
	singleID = "decision"

	defaultTimeout   = 30 * time.Second
	maxResponseBytes = 8 << 20
)

// Client 决策模型客户端，零值不可用，请用 New 创建。
type Client struct {
	// APIKey 访问令牌，必填。
	APIKey string
	// Model 模型名，默认 decision-model-preview。
	// 单次请求可用 Request.Model 覆盖。
	Model string
	// Endpoint 归一化后的完整接口地址（网关 + /compatible-mode/v1/systemone）。
	// 由 WithBaseURL/WithWorkspace 设置，只给域名时自动补齐路径。
	Endpoint string
	// HTTPClient 底层 HTTP 客户端，为空时使用 defaultTimeout 创建。
	HTTPClient *http.Client
}

// Option New 的配置项。
type Option func(*Client)

// WithAPIKey 显式指定 API Key。
func WithAPIKey(key string) Option {
	return func(c *Client) { c.APIKey = key }
}

// WithModel 指定模型名。
func WithModel(model string) Option {
	return func(c *Client) { c.Model = model }
}

// WithBaseURL 指定网关地址，例如 https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com。
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.Endpoint = baseURL }
}

// WithWorkspace 按业务空间 ID 与地域拼出网关地址，地域为空取华北2（北京）。
// 业务空间 ID 见「地域与域名」文档。
func WithWorkspace(workspaceID, region string) Option {
	return func(c *Client) {
		if region == "" {
			region = RegionBeijing
		}
		c.Endpoint = fmt.Sprintf("https://%s.%s.maas.aliyuncs.com", workspaceID, region)
	}
}

// WithHTTPClient 复用外部 HTTP 客户端（连接池、代理、测试桩）。
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.HTTPClient = h }
}

// New 创建客户端。未显式配置时读取环境变量
// DASHSCOPE_API_KEY / DASHSCOPE_BASE_URL / DASHSCOPE_WORKSPACE_ID / DASHSCOPE_REGION。
func New(opts ...Option) (*Client, error) {
	c := &Client{
		APIKey:   os.Getenv(APIKeyEnv),
		Model:    DefaultModel,
		Endpoint: os.Getenv(BaseURLEnv),
	}
	if c.Endpoint == "" {
		if ws := os.Getenv(WorkspaceIDEnv); ws != "" {
			WithWorkspace(ws, os.Getenv(RegionEnv))(c)
		}
	}
	for _, opt := range opts {
		opt(c)
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("decision: 缺少 API Key（传 WithAPIKey 或设置 %s）", APIKeyEnv)
	}
	if strings.TrimSpace(c.Endpoint) == "" {
		return nil, fmt.Errorf("decision: 缺少网关地址（传 WithBaseURL/WithWorkspace 或设置 %s、%s）", BaseURLEnv, WorkspaceIDEnv)
	}
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: defaultTimeout}
	}
	c.Endpoint = normalizeBaseURL(c.Endpoint)
	// 提前校验地址，避免把 {WorkspaceId} 这类占位符留到请求阶段才报错。
	if u, err := url.Parse(c.Endpoint); err != nil || u.Host == "" {
		return nil, fmt.Errorf("decision: 网关地址不合法 %q（需替换 {WorkspaceId}，且带 https://）", c.Endpoint)
	}
	return c, nil
}

// normalizeBaseURL 统一成带 /compatible-mode/v1/systemone 的完整地址。
func normalizeBaseURL(base string) string {
	base = strings.TrimSuffix(strings.TrimSpace(base), "/")
	switch {
	case strings.HasSuffix(base, APIPath):
		return base
	case strings.HasSuffix(base, "/compatible-mode/v1"):
		return base + "/systemone"
	default:
		return base + APIPath
	}
}

// Ask 发起一次决策：state 可以是文本、对象或数组，questions 的 key 为自定义问题 id。
func (c *Client) Ask(ctx context.Context, state any, questions map[string]Question) (*Response, error) {
	return c.AskRequest(ctx, &Request{State: state, Questions: questions})
}

// AskRequest 发起自定义请求，Request.Model 为空时用客户端模型。
func (c *Client) AskRequest(ctx context.Context, req *Request) (*Response, error) {
	if c == nil {
		return nil, fmt.Errorf("decision: client 为 nil")
	}
	if req == nil {
		return nil, fmt.Errorf("decision: request 为 nil")
	}
	if req.Model == "" {
		req.Model = c.Model
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("decision: 序列化请求失败: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("decision: 创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("decision: 调用失败: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("decision: 读取响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newAPIError(resp, payload)
	}

	var out Response
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("decision: 解析响应失败: %w, body=%s", err, trim(payload))
	}
	if out.RequestID == "" {
		out.RequestID = headerRequestID(resp)
	}
	return &out, nil
}

// Classify 多选一快捷调用，返回选项名与置信度。
func (c *Client) Classify(ctx context.Context, state any, instructions string, options ChoiceOptions) (string, float64, error) {
	a, err := c.askSingle(ctx, state, NewChoice(instructions, options))
	if err != nil {
		return "", 0, err
	}
	return a.Label(), a.Confidence, nil
}

// Judge 是非判断快捷调用，返回是否为「是」及 P(yes)（0=否 1=是）。
func (c *Client) Judge(ctx context.Context, state any, instructions string) (bool, float64, error) {
	a, err := c.askSingle(ctx, state, NewYesNo(instructions))
	if err != nil {
		return false, 0, err
	}
	return a.Yes(), a.YesProbability(), nil
}

// Rate 有序量表快捷调用，用 Answer.Value/Level/LevelText 取结果。
func (c *Client) Rate(ctx context.Context, state any, instructions string, levels ...string) (Answer, error) {
	return c.askSingle(ctx, state, NewScore(instructions, levels...))
}

func (c *Client) askSingle(ctx context.Context, state any, q Question) (Answer, error) {
	resp, err := c.Ask(ctx, state, map[string]Question{singleID: q})
	if err != nil {
		return Answer{}, err
	}
	a, err := resp.MustAnswer(singleID)
	if err != nil {
		return Answer{}, err
	}
	return a, nil
}

// headerRequestID 服务端未在 body 里回传 request_id 时的兜底。
func headerRequestID(resp *http.Response) string {
	for _, k := range []string{"X-Request-Id", "X-Dashscope-Request-Id"} {
		if v := resp.Header.Get(k); v != "" {
			return v
		}
	}
	return ""
}

func trim(p []byte) string {
	s := strings.TrimSpace(string(p))
	if len(s) > 512 {
		return s[:512] + "..."
	}
	return s
}
