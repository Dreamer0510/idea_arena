package middleware

import (
	"context"
	"strings"

	"idea_arena/internal/pkg/auth"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

type authKey struct{}

// AuthClaims 从 context 中获取认证信息
func AuthClaims(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(authKey{}).(*auth.Claims)
	return claims
}

// JWTAuth JWT 认证中间件
func JWTAuth(jwtHelper *auth.JWTHelper, skipPaths []string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 检查是否跳过认证
			if tr, ok := transport.FromServerContext(ctx); ok {
				if ht, ok := tr.(*http.Transport); ok {
					path := ht.Request().URL.Path
					for _, skip := range skipPaths {
						if strings.HasPrefix(path, skip) {
							return handler(ctx, req)
						}
					}
				}
			}

			// 提取 token
			var tokenString string
			if tr, ok := transport.FromServerContext(ctx); ok {
				authHeader := tr.RequestHeader().Get("Authorization")
				if authHeader == "" {
					return nil, errors.Unauthorized("UNAUTHORIZED", "missing authorization header")
				}
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
					return nil, errors.Unauthorized("UNAUTHORIZED", "invalid authorization format")
				}
				tokenString = parts[1]
			}

			// 解析 token
			claims, err := jwtHelper.ParseToken(tokenString)
			if err != nil {
				return nil, errors.Unauthorized("UNAUTHORIZED", "invalid or expired token")
			}

			// 注入 claims 到 context
			ctx = context.WithValue(ctx, authKey{}, claims)
			return handler(ctx, req)
		}
	}
}
