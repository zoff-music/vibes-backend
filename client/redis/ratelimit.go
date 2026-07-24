package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) CheckRateLimit(ctx context.Context, request vibe.RateLimitRequest) (*vibe.RateLimitResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CheckRateLimit")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection: %w", err)
	}
	defer connection.Close()

	identityKey := c.getKeyWithPrefix(fmt.Sprintf("ratelimit:identity:{%s}:%s", request.IdentityHash, request.RouteName))
	ipKey := c.getKeyWithPrefix(fmt.Sprintf("ratelimit:ip:{%s}:%s", request.IPIdentityHash, request.RouteName))

	identityCount, err := getRateLimitCount(cctx, connection, identityKey)
	if err != nil {
		return nil, fmt.Errorf("error counting identity rate limits: %w", err)
	}
	if identityCount >= int64(request.Limit) {
		return &vibe.RateLimitResult{RetryAfter: request.Rate}, nil
	}

	ipCount, err := getRateLimitCount(cctx, connection, ipKey)
	if err != nil {
		return nil, fmt.Errorf("error counting IP rate limits: %w", err)
	}
	if ipCount >= int64(request.IPLimit) {
		return &vibe.RateLimitResult{RetryAfter: request.Rate}, nil
	}

	member := uuid.NewString()
	err = setRateLimit(cctx, connection, identityKey, member, request.Rate)
	if err != nil {
		return nil, fmt.Errorf("error setting identity rate limit: %w", err)
	}
	err = setRateLimit(cctx, connection, ipKey, member, request.Rate)
	if err != nil {
		return nil, fmt.Errorf("error setting IP rate limit: %w", err)
	}

	return &vibe.RateLimitResult{Allowed: true}, nil
}

func getRateLimitCount(ctx context.Context, connection redis.Conn, key string) (int64, error) {
	count, err := redis.Int64(redis.DoContext(connection, ctx, "HLEN", key))
	if err != nil {
		return 0, fmt.Errorf("error getting redis hash length: %w", err)
	}

	return count, nil
}

func setRateLimit(
	ctx context.Context,
	connection redis.Conn,
	key string,
	member string,
	expiration time.Duration,
) error {
	expirationMilliseconds := max(expiration.Milliseconds(), int64(1))
	_, err := redis.DoContext(connection, ctx, "HSETEX", key, "PX", expirationMilliseconds, "FIELDS", 1, member, 1)
	if err != nil {
		return fmt.Errorf("error setting expiring redis hash field: %w", err)
	}

	return nil
}
