package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"golang.org/x/time/rate"
)

// RateLimit 限流中间件
func RateLimit(limiter *rate.Limiter) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if !limiter.Allow() {
				return nil, errors.New(429, "RATE_LIMITED", "too many requests")
			}
			return handler(ctx, req)
		}
	}
}
