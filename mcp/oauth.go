package mcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type authMiddleware struct {
	jwtSecret []byte
	//userStore UserStore
}

func NewAuthMiddleware(secret []byte) *authMiddleware {
	return &authMiddleware{
		jwtSecret: secret,
		//userStore: store,
	}
}

func (m *authMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			//http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
			next.ServeHTTP(w, r)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate JWT token
		claims, err := m.validateJWT(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Load user information
		//user, err := m.userStore.GetUser(claims.UserID)
		//if err != nil {
		//	http.Error(w, "User not found", http.StatusUnauthorized)
		//	return
		//}

		// Add user to request context
		ctx := context.WithValue(r.Context(), "user", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *authMiddleware) validateJWT(tokenString string) (*Claims, error) {
	// Note: This example uses a hypothetical JWT library
	// In practice, you would use a real JWT library like github.com/golang-jwt/jwt
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		return m.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}
