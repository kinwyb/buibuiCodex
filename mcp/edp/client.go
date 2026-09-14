package edp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var defaultAPIClient *APIClient

// APIClient API客户端
type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewAPIClient 创建API客户端
func NewAPIClient(baseURL string) *APIClient {
	defaultAPIClient = &APIClient{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
	return defaultAPIClient
}

// GetAPIClient 获取API客户端
func GetAPIClient() *APIClient {
	return defaultAPIClient
}

// NewRequest 创建一个请求
func (c *APIClient) NewRequest(req *Request) (*http.Request, error) {
	// 构建完整URL
	u, err := url.Parse(c.BaseURL + req.Path)
	if err != nil {
		return nil, fmt.Errorf("解析URL失败: %w", err)
	}

	// 添加查询参数
	if len(req.QueryParams) > 0 {
		q := u.Query()
		for k, v := range req.QueryParams {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	// 序列化请求体
	var reqBody []byte
	if req.Body != nil {
		reqBody, err = json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
	}

	// 创建请求
	httpReq, err := http.NewRequest(req.Method, u.String(), bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// 设置认证token
	if req.Token != "" {
		httpReq.Header.Set("token", req.Token)
	}
	return httpReq, nil
}

type RequestOption func(*Request)

type Request struct {
	Method      string
	Path        string
	Token       string
	QueryParams map[string]string
	Body        any
}

func WithToken(token string) RequestOption {
	return func(r *Request) {
		r.Token = token
	}
}

func WithQueryParams(params map[string]string) RequestOption {
	return func(r *Request) {
		r.QueryParams = params
	}
}

func WithBody(body any) RequestOption {
	return func(r *Request) {
		r.Body = body
	}
}

func NewRequest(method string, path string, options ...RequestOption) *Request {
	req := &Request{
		Method: method,
		Path:   path,
	}
	for _, option := range options {
		option(req)
	}
	return req
}

// Response 统一响应结构，Data 根据接口不同使用泛型指定类型。
type Response[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     string `json:"errmsg"`
	Data    T      `json:"data"`
}

// DoClient 执行HTTP请求，返回泛型响应。
func DoClient[T any](req *Request) (*Response[T], error) {

	respBody, err := DoClientRow(req)
	if err != nil {
		return nil, err
	}

	// 解析响应
	var result Response[T]
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != 0 {
		return nil, errors.New(result.Err)
	}

	return &result, nil
}

// DoClientRow 执行HTTP请求，返回原始内容
func DoClientRow(req *Request) ([]byte, error) {
	// 执行请求
	c := GetAPIClient()
	if c == nil {
		return nil, errors.New("APIClient is nil")
	}
	if req == nil {
		return nil, errors.New("request is nil")
	}
	request, err := c.NewRequest(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
