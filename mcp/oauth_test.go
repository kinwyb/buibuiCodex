package mcp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// newHS256Token 用对称密钥构造令牌，用于验证算法混淆防护。
func newHS256Token(t *testing.T, secret []byte) string {
	t.Helper()
	now := time.Now()
	claims := &Claims{
		UserID: "1@1",
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   "1@1",
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatalf("构造 HS256 令牌失败: %v", err)
	}
	return s
}

const (
	testJWKSURL = "https://localhost/.well-known/jwks.json"
	testIssuer  = "https://localhost"
)

// readASToken 读取由授权服务器签发的真实 RS256 令牌（离线时跳过用例）。
func readASToken(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("/tmp/as_token.txt")
	if err != nil {
		t.Skip("未找到 /tmp/as_token.txt，跳过")
	}
	return strings.TrimSpace(string(b))
}

// 授权服务器签发的 RS256 令牌：应通过 JWKS 验签。
func TestRS256TokenAccepted(t *testing.T) {
	m := NewAuthMiddleware(testIssuer, "", testJWKSURL, "")
	claims, err := m.validateJWT(readASToken(t))
	if err != nil {
		t.Fatalf("期望验签通过，实际失败: %v", err)
	}
	t.Logf("验签通过: sub=%s user_id=%s iss=%s aud=%v",
		claims.Subject, claims.UserID, claims.Issuer, claims.Audience)
}

// 对称密钥签发的 HS256 令牌：必须被拒绝（防算法混淆攻击）。
func TestHS256TokenRejected(t *testing.T) {
	secret := []byte("3cf4c0ab56ed79be8350997d5b4ada81bddb2cd3d0ae09f5d9208de04882b93b")
	hs := newHS256Token(t, secret)

	m := NewAuthMiddleware(testIssuer, "", testJWKSURL, "")
	if _, err := m.validateJWT(hs); err == nil {
		t.Fatal("HS256 令牌不应通过校验（算法混淆漏洞）")
	} else {
		t.Logf("已正确拒绝 HS256 令牌: %v", err)
	}
}

// audience 不匹配时必须拒绝。
func TestWrongAudienceRejected(t *testing.T) {
	m := NewAuthMiddleware(testIssuer, "http://example.com/not-me", testJWKSURL, "")
	if _, err := m.validateJWT(readASToken(t)); err == nil {
		t.Fatal("audience 不匹配时不应通过校验")
	} else {
		t.Logf("已正确拒绝 audience 不匹配的令牌: %v", err)
	}
}

// issuer 不匹配时必须拒绝。
func TestWrongIssuerRejected(t *testing.T) {
	m := NewAuthMiddleware("https://evil.example.com", "", testJWKSURL, "")
	if _, err := m.validateJWT(readASToken(t)); err == nil {
		t.Fatal("issuer 不匹配时不应通过校验")
	} else {
		t.Logf("已正确拒绝 issuer 不匹配的令牌: %v", err)
	}
}

// 说明：AS 目前签发的令牌 aud 是 client_id，而非资源标识（RFC 8707）。
// 因此当 audience 按规范配成资源地址时，即使验签通过也会被拒。
// 本用例记录这一现状，AS 支持 resource 参数后此用例的期望值应改为「通过」。
func TestResourceAudienceMismatch(t *testing.T) {
	m := NewAuthMiddleware(testIssuer, "http://localhost:9090", testJWKSURL, "")
	_, err := m.validateJWT(readASToken(t))
	if err != nil {
		t.Logf("audience=资源标识 时被拒（符合当前 AS 行为）: %v", err)
	} else {
		t.Logf("audience=资源标识 时通过 —— AS 已支持 RFC 8707，可启用 audience 严格校验")
	}
}

// 中间件对缺失 Authorization 头的响应应为 401 且带 WWW-Authenticate。
func TestMiddlewareUnauthorized(t *testing.T) {
	m := NewAuthMiddleware(testIssuer, "", testJWKSURL,
		"http://localhost:9090/.well-known/oauth-protected-resource")

	rec := httptest.NewRecorder()
	m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("不应进入下游 handler")
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，实际 %d", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); !strings.Contains(got, "resource_metadata") {
		t.Fatalf("WWW-Authenticate 头缺失或格式不对: %q", got)
	}
	t.Logf("401 响应正确，WWW-Authenticate=%s", rec.Header().Get("WWW-Authenticate"))
}
