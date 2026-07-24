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
	globalKey := c.getKeyWithPrefix(fmt.Sprintf("ratelimit:global:%s", request.RouteName))
	member := uuid.NewString()
	expirationMilliseconds := max(request.Rate.Milliseconds(), int64(1))
	globalExpirationMilliseconds := max(request.GlobalRate.Milliseconds(), int64(1))
	allowed, err := redis.Int(redis.DoContext(
		connection,
		cctx,
		"EVAL",
		rateLimitScript,
		3,
		identityKey,
		ipKey,
		globalKey,
		member,
		expirationMilliseconds,
		request.Limit,
		request.IPLimit,
		globalExpirationMilliseconds,
		request.GlobalLimit,
	))
	if err != nil {
		return nil, fmt.Errorf("error checking and setting rate limit: %w", err)
	}
	if allowed == rateLimitDenied {
		return &vibe.RateLimitResult{RetryAfter: request.Rate}, nil
	}
	if allowed == rateLimitGlobalDenied {
		return &vibe.RateLimitResult{RetryAfter: request.GlobalRate}, nil
	}

	return &vibe.RateLimitResult{Allowed: true}, nil
}

const rateLimitScript = `
if redis.call("HLEN", KEYS[1]) >= tonumber(ARGV[3]) then
	return 0
end
if redis.call("HLEN", KEYS[2]) >= tonumber(ARGV[4]) then
	return 0
end
if tonumber(ARGV[6]) > 0 and redis.call("HLEN", KEYS[3]) >= tonumber(ARGV[6]) then
	return 2
end
redis.call("HSETEX", KEYS[1], "PX", ARGV[2], "FIELDS", 1, ARGV[1], 1)
redis.call("HSETEX", KEYS[2], "PX", ARGV[2], "FIELDS", 1, ARGV[1], 1)
if tonumber(ARGV[6]) > 0 then
	redis.call("HSETEX", KEYS[3], "PX", ARGV[5], "FIELDS", 1, ARGV[1], 1)
end
return 1
`

const rateLimitDenied = 0

const rateLimitGlobalDenied = 2
