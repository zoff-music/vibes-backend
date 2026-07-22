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

func (c *Client) ConsumeRateLimit(ctx context.Context, request vibe.RateLimitRequest) (vibe.RateLimitResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ConsumeRateLimit")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error getting redis connection: %w", err)
	}
	defer connection.Close()

	identityKey := c.getKeyWithPrefix(fmt.Sprintf("ratelimit:identity:{%s}:%s", request.IdentityHash, request.RouteName))
	ipKey := c.getKeyWithPrefix(fmt.Sprintf("ratelimit:ip:{%s}:%s", request.IPIdentityHash, request.RouteName))

	identityCount, err := getRateLimitCount(cctx, connection, identityKey)
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error counting identity rate limits: %w", err)
	}
	if identityCount >= int64(request.Limit) {
		return getLimitedResult(cctx, connection, identityKey)
	}

	ipCount, err := getRateLimitCount(cctx, connection, ipKey)
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error counting IP rate limits: %w", err)
	}
	if ipCount >= int64(request.IPLimit) {
		return getLimitedResult(cctx, connection, ipKey)
	}

	member := uuid.NewString()
	err = setRateLimit(cctx, connection, identityKey, member, request.Rate)
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error setting identity rate limit: %w", err)
	}
	err = setRateLimit(cctx, connection, ipKey, member, request.Rate)
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error setting IP rate limit: %w", err)
	}

	return vibe.RateLimitResult{Allowed: true}, nil
}

func getRateLimitCount(ctx context.Context, connection redis.Conn, key string) (int64, error) {
	count, err := redis.Int64(redis.DoContext(connection, ctx, "HLEN", key))
	if err != nil {
		return 0, fmt.Errorf("error getting redis hash length: %w", err)
	}

	return count, nil
}

func getLimitedResult(ctx context.Context, connection redis.Conn, key string) (vibe.RateLimitResult, error) {
	retryAfter, err := rateLimitRetryAfter(ctx, connection, key)
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error getting rate limit retry duration: %w", err)
	}

	return vibe.RateLimitResult{RetryAfter: retryAfter}, nil
}

func setRateLimit(
	ctx context.Context,
	connection redis.Conn,
	key string,
	member string,
	expiration time.Duration,
) error {
	expirationMilliseconds := max(expiration.Milliseconds(), int64(1))
	args := redis.Args{}.
		Add(key).
		Add("PX").
		Add(expirationMilliseconds).
		Add("FIELDS").
		Add(1).
		Add(member).
		Add(1)

	_, err := redis.DoContext(connection, ctx, "HSETEX", args...)
	if err != nil {
		return fmt.Errorf("error setting expiring redis hash field: %w", err)
	}

	return nil
}

func rateLimitRetryAfter(ctx context.Context, connection redis.Conn, key string) (time.Duration, error) {
	fields, err := redis.Strings(redis.DoContext(connection, ctx, "HKEYS", key))
	if err != nil {
		return 0, fmt.Errorf("error getting redis rate limit entries: %w", err)
	}
	if len(fields) == 0 {
		return time.Millisecond, nil
	}

	args := redis.Args{}.Add(key).Add("FIELDS").Add(len(fields)).AddFlat(fields)
	ttls, err := redis.Int64s(redis.DoContext(connection, ctx, "HPTTL", args...))
	if err != nil {
		return 0, fmt.Errorf("error getting redis rate limit expirations: %w", err)
	}

	retryAfter := time.Duration(0)
	for _, ttl := range ttls {
		ttlDuration := time.Duration(ttl) * time.Millisecond
		if ttlDuration > 0 && (retryAfter == 0 || ttlDuration < retryAfter) {
			retryAfter = ttlDuration
		}
	}
	if retryAfter == 0 {
		return time.Millisecond, nil
	}

	return retryAfter, nil
}
