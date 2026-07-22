package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) ConsumeRateLimit(ctx context.Context, request vibe.RateLimitRequest) (vibe.RateLimitResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ConsumeRateLimit")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	keyPrefix := fmt.Sprintf("vibes:ratelimit:{%s}:%s", request.IPIdentityHash, request.RouteName)
	deviceKey := fmt.Sprintf("%s:device:%s", keyPrefix, request.DeviceIdentityHash)
	ipKey := keyPrefix + ":ip"

	values, err := c.consumeScript.Run(
		cctx,
		c.client,
		[]string{deviceKey, ipKey},
		request.Rate.Microseconds(),
		request.DeviceLimit,
		request.IPLimit,
		uuid.NewString(),
	).Int64Slice()
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error consuming redis rate limit: %w", err)
	}
	if len(values) != rateLimitResultLength {
		return vibe.RateLimitResult{}, fmt.Errorf("error consuming redis rate limit: %w", vibe.RateLimitResultError{
			ExpectedLength: rateLimitResultLength,
			ActualLength:   len(values),
		})
	}

	return vibe.RateLimitResult{
		Allowed:    values[0] == 1,
		RetryAfter: time.Duration(values[1]) * time.Microsecond,
	}, nil
}

const rateLimitResultLength = 2

const consumeRateLimitScript = `
local device_key = KEYS[1]
local ip_key = KEYS[2]
local window_us = tonumber(ARGV[1])
local device_limit = tonumber(ARGV[2])
local ip_limit = tonumber(ARGV[3])
local nonce = ARGV[4]

local redis_time = redis.call('TIME')
local now_us = tonumber(redis_time[1]) * 1000000 + tonumber(redis_time[2])
local cutoff_us = now_us - window_us

redis.call('ZREMRANGEBYSCORE', device_key, '-inf', cutoff_us)
redis.call('ZREMRANGEBYSCORE', ip_key, '-inf', cutoff_us)

local function expire_key(key)
    local latest = redis.call('ZRANGE', key, -1, -1, 'WITHSCORES')
    if latest[2] then
        local ttl_us = tonumber(latest[2]) + window_us - now_us
        if ttl_us > 0 then
            redis.call('PEXPIRE', key, math.max(1, math.ceil(ttl_us / 1000)))
        end
    end
end

local function retry_after(key)
    local earliest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    return math.max(1, math.ceil(tonumber(earliest[2]) + window_us - now_us))
end

if redis.call('ZCARD', device_key) >= device_limit then
    expire_key(device_key)
    expire_key(ip_key)
    return {0, retry_after(device_key)}
end

if redis.call('ZCARD', ip_key) >= ip_limit then
    expire_key(device_key)
    expire_key(ip_key)
    return {0, retry_after(ip_key)}
end

local member = tostring(now_us) .. ':' .. nonce
redis.call('ZADD', device_key, now_us, member)
redis.call('ZADD', ip_key, now_us, member)
expire_key(device_key)
expire_key(ip_key)

return {1, 0}
`
