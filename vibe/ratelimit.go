package vibe

import (
	"context"
	"time"
)

type RateLimitPolicy struct {
	Rate  time.Duration
	Limit int
}

type RateLimitRequest struct {
	RouteName          string
	DeviceIdentityHash string
	IPIdentityHash     string
	Rate               time.Duration
	DeviceLimit        int
	IPLimit            int
}

type RateLimitResult struct {
	Allowed    bool
	RetryAfter time.Duration
}

type RateLimitConsumer interface {
	ConsumeRateLimit(ctx context.Context, request RateLimitRequest) (RateLimitResult, error)
}
