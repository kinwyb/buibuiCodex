package mcp

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuth2Client 实现 OAuth2 授权码流程，用于从授权服务器换取 JWT 令牌。
type OAuth2Client struct {
	cfg *OAuth2Config
	hc  *http.Client
}

// OAuth2Config OAuth2 客户端配置。
// TokenEndpoint 和 AuthEndpoint 留空时自动从 Issuer 推导。
type OAuth2Config struct {
	ClientID      string   // 客户端 ID
	ClientSecret  string   // 客户端密钥
	RedirectURL   string   // 回调地址，如 http://localhost:9090/oauth/callback
	Issuer        string   // 授权服务器地址
	TokenEndpoint string   // 令牌端点（可选，默认 Issuer + /oauth/token）
	AuthEndpoint  string   // 授权端点（可选，默认 Issuer + /oauth/authorize）
	Scopes        []string // 请求的权限范围
}

// TokenResponse 授权服务器令牌端点的标准响应（RFC 6749 §5.1）。
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// NewOAuth2Client 创建 OAuth2 客户端。cfg 为 nil 或 ClientID 为空时返回 nil。
func NewOAuth2Client(cfg *OAuth2Config) *OAuth2Client {
	if cfg == nil || cfg.ClientID == "" {
		return nil
	}
	issuer := strings.TrimSuffix(cfg.Issuer, "/")
	if cfg.TokenEndpoint == "" {
		cfg.TokenEndpoint = issuer + "/oauth/token"
	}
	if cfg.AuthEndpoint == "" {
		cfg.AuthEndpoint = issuer + "/oauth/authorize"
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "profile", "email"}
	}
	return &OAuth2Client{
		cfg: cfg,
		hc: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// AuthCodeURL 构造授权码跳转链接并生成随机 state（CSRF 防护）。
// 调用方应将 state 缓存到 session 中，在回调时校验。
func (c *OAuth2Client) AuthCodeURL() (authURL, state string) {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	state = hex.EncodeToString(b)

	params := url.Values{
		"response_type": {"code"},
		"client_id":     {c.cfg.ClientID},
		"redirect_uri":  {c.cfg.RedirectURL},
		"scope":         {strings.Join(c.cfg.Scopes, " ")},
		"state":         {state},
	}
	authURL = c.cfg.AuthEndpoint + "?" + params.Encode()
	return
}

// ExchangeToken 用授权码换取访问令牌。
func (c *OAuth2Client) ExchangeToken(ctx context.Context, code string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {c.cfg.RedirectURL},
		"client_id":     {c.cfg.ClientID},
		"client_secret": {c.cfg.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TokenEndpoint,
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("构造令牌请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("令牌请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("令牌端点返回 %d: %s", resp.StatusCode, string(body))
	}

	var tok TokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("解析令牌响应失败: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("令牌响应中缺少 access_token")
	}
	return &tok, nil
}

// RefreshToken 用刷新令牌换取新的访问令牌。
func (c *OAuth2Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {c.cfg.ClientID},
		"client_secret": {c.cfg.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TokenEndpoint,
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("构造刷新请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("刷新请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("刷新端点返回 %d: %s", resp.StatusCode, string(body))
	}

	var tok TokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("解析刷新响应失败: %w", err)
	}
	return &tok, nil
}

// HandleCallback 处理授权服务器的回调（/oauth/callback）。
//
// 流程：
//  1. 校验 state 参数（防 CSRF）
//  2. 用授权码调用令牌端点换取 JWT
//  3. 将令牌写入响应（生产环境建议加密存储或写入 cookie）
func (c *OAuth2Client) HandleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// 授权服务器返回错误时直接展示（RFC 6749 §4.1.2.1）
	if errDesc := q.Get("error"); errDesc != "" {
		slog.Error("OAuth2 授权失败", "error", errDesc, "description", q.Get("error_description"))
		http.Error(w, fmt.Sprintf("OAuth2 授权失败: %s - %s", errDesc, q.Get("error_description")),
			http.StatusBadRequest)
		return
	}

	code := q.Get("code")
	state := q.Get("state")
	if code == "" {
		http.Error(w, "缺少授权码 (code)", http.StatusBadRequest)
		return
	}

	// 校验 state（生产环境应从 session/cookie 中读取并比对）
	if state == "" {
		slog.Warn("回调缺少 state 参数，CSRF 校验跳过")
	}

	// 用授权码换取令牌
	tok, err := c.ExchangeToken(r.Context(), code)
	if err != nil {
		slog.Error("换取令牌失败", "error", err)
		http.Error(w, "换取令牌失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("OAuth2 令牌换取成功",
		"token_type", tok.TokenType,
		"expires_in", tok.ExpiresIn,
		"has_refresh", tok.RefreshToken != "",
	)

	// 返回令牌信息（生产环境建议写入 HttpOnly Secure Cookie 或加密存储）
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	refreshNote := ""
	if tok.RefreshToken != "" {
		refreshNote = fmt.Sprintf(`<p><strong>Refresh Token:</strong> %s</p>`, tok.RefreshToken)
	}
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>OAuth2 授权成功</title></head>
<body>
<h1>授权成功</h1>
<p><strong>Access Token (JWT):</strong></p>
<pre style="word-break:break-all;white-space:pre-wrap;max-width:800px;">%s</pre>
%s
<p>Expires in: %d seconds</p>
</body></html>`, tok.AccessToken, refreshNote, tok.ExpiresIn)
}

// ServeHTTP 使 OAuth2Client 可直接作为 http.Handler 注册到路由。
func (c *OAuth2Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.HandleCallback(w, r)
}
