package decision

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError 调用失败时的错误，服务端返回码详见「错误码」文档。
type APIError struct {
	Status    int
	Code      string
	Message   string
	RequestID string
	// Body 原始响应体，便于排查未识别的错误格式。
	Body string
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "decision: 接口返回 %d", e.Status)
	if e.Code != "" {
		fmt.Fprintf(&b, " code=%s", e.Code)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, " message=%s", e.Message)
	}
	if e.RequestID != "" {
		fmt.Fprintf(&b, " request_id=%s", e.RequestID)
	}
	return b.String()
}

// IsRateLimit 是否为限流错误，调用方可以退避重试。
func (e *APIError) IsRateLimit() bool {
	if e == nil {
		return false
	}
	if e.Status == http.StatusTooManyRequests {
		return true
	}
	code := strings.ToLower(e.Code)
	return strings.Contains(code, "throttl") || strings.Contains(code, "rate_limit") || strings.Contains(code, "429")
}

// errBody 兼容 OpenAI 风格与 DashScope 风格两种错误结构。
type errBody struct {
	Error struct {
		Code    any    `json:"code"`
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
	Code      string `json:"code"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func newAPIError(resp *http.Response, payload []byte) *APIError {
	e := &APIError{Status: resp.StatusCode, Body: trim(payload), RequestID: headerRequestID(resp)}
	var b errBody
	if err := json.Unmarshal(payload, &b); err == nil {
		e.Message = firstNonEmpty(b.Error.Message, b.Message)
		e.Code = firstNonEmpty(anyString(b.Error.Code), b.Code, b.Error.Type)
		e.RequestID = firstNonEmpty(b.RequestID, e.RequestID)
	}
	if e.Message == "" {
		e.Message = resp.Status
	}
	return e
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// anyString 错误码可能是数字，也可能是字符串。
func anyString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	default:
		return fmt.Sprint(x)
	}
}
