package vibe

import (
	"context"
	"time"
)

type RateLimitPolicy struct {
	Rate        time.Duration
	Limit       int
	IPLimit     int
	GlobalRate  time.Duration
	GlobalLimit int
}

type RateLimitRequest struct {
	RouteName      string
	IdentityHash   string
	IPIdentityHash string
	Rate           time.Duration
	Limit          int
	IPLimit        int
}

type RateLimitResult struct {
	Allowed    bool
	RetryAfter time.Duration
}

type RateLimitChecker interface {
	CheckRateLimit(ctx context.Context, request RateLimitRequest) (*RateLimitResult, error)
}
