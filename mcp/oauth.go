package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// authMiddleware 校验 Authorization 头中的 OAuth2 Bearer Token。
//
// 本服务是资源服务器（RS），令牌由授权服务器（AS）以非对称方式（RS256）签发，
// 因此必须使用 AS 公布的 JWKS 公钥验签，而不能使用任何对称密钥。
type authMiddleware struct {
	keys *keyStore

	issuer   string // 期望的 iss（授权服务器标识），为空则跳过校验
	audience string // 期望的 aud（本服务的资源标识），为空则跳过校验
	// resourceMetadataURL 用于在 401 响应中通过 WWW-Authenticate 告知客户端去哪里发现元数据
	resourceMetadataURL string
}

// NewAuthMiddleware 创建认证中间件。
// jwksURL 为授权服务器的 JWKS 端点，例如 https://localhost/.well-known/jwks.json
func NewAuthMiddleware(issuer, audience, jwksURL, resourceMetadataURL string) *authMiddleware {
	return &authMiddleware{
		keys:                newKeyStore(jwksURL),
		issuer:              issuer,
		audience:            audience,
		resourceMetadataURL: resourceMetadataURL,
	}
}

func (m *authMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			// RFC 9728：受保护资源应在 401 中回传资源元数据地址，供客户端发现授权服务器
			if m.resourceMetadataURL != "" {
				w.Header().Set("WWW-Authenticate",
					fmt.Sprintf(`Bearer resource_metadata=%q`, m.resourceMetadataURL))
			}
			http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := m.validateJWT(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "user", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *authMiddleware) validateJWT(tokenString string) (*Claims, error) {
	opts := []jwt.ParserOption{
		// 只允许 RS256。若不加此限制，攻击者可把 header 的 alg 改成 HS256，
		// 诱导服务端把公钥（或任何已知字符串）当作对称密钥验签，即算法混淆攻击。
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithExpirationRequired(),
	}
	if m.issuer != "" {
		opts = append(opts, jwt.WithIssuer(m.issuer))
	}
	if m.audience != "" {
		opts = append(opts, jwt.WithAudience(m.audience))
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("token header 缺少 kid")
		}
		// 按 kid 取 AS 公布的 RSA 公钥
		return m.keys.Key(kid)
	}, opts...)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// AS 使用标准 sub 承载用户标识，这里映射到业务字段
	if claims.UserID == "" {
		claims.UserID = claims.Subject
	}
	return claims, nil
}

// Claims 是令牌中本服务关心的字段。
// jwt.RegisteredClaims 已包含 iss / sub / aud / exp / iat 等标准声明。
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}
