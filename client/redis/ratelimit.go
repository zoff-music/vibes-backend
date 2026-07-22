package redis

import (
	"context"
	"errors"
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

	keyPrefix := c.getKeyWithPrefix(fmt.Sprintf("ratelimit:{%s}:%s", request.IPIdentityHash, request.RouteName))
	deviceKey := fmt.Sprintf("%s:device:%s", keyPrefix, request.DeviceIdentityHash)
	ipKey := keyPrefix + ":ip"

	for attempt := 0; attempt < rateLimitTransactionAttempts; attempt++ {
		result, consumeErr := consumeRateLimitTransaction(cctx, connection, request, deviceKey, ipKey)
		if consumeErr == nil {
			return result, nil
		}

		var conflictError rateLimitTransactionConflictError
		if !errors.As(consumeErr, &conflictError) {
			return vibe.RateLimitResult{}, fmt.Errorf("error consuming redis rate limit: %w", consumeErr)
		}
	}

	return vibe.RateLimitResult{}, fmt.Errorf(
		"error consuming redis rate limit after %d attempts: %w",
		rateLimitTransactionAttempts,
		rateLimitTransactionConflictError{Reason: "rate limit keys changed"},
	)
}

func consumeRateLimitTransaction(
	ctx context.Context,
	connection redis.Conn,
	request vibe.RateLimitRequest,
	deviceKey string,
	ipKey string,
) (vibe.RateLimitResult, error) {
	_, err := redis.DoContext(connection, ctx, "WATCH", deviceKey, ipKey)
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error watching redis rate limits: %w", err)
	}

	deviceCount, err := redis.Int64(redis.DoContext(connection, ctx, "HLEN", deviceKey))
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error counting device rate limits: %w", err)
	}
	ipCount, err := redis.Int64(redis.DoContext(connection, ctx, "HLEN", ipKey))
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error counting IP rate limits: %w", err)
	}

	result := vibe.RateLimitResult{Allowed: true}
	limitedKey := ""
	if deviceCount >= int64(request.DeviceLimit) {
		result.Allowed = false
		limitedKey = deviceKey
	} else if ipCount >= int64(request.IPLimit) {
		result.Allowed = false
		limitedKey = ipKey
	}

	if !result.Allowed {
		result.RetryAfter, err = rateLimitRetryAfter(ctx, connection, limitedKey)
		if err != nil {
			return vibe.RateLimitResult{}, fmt.Errorf("error getting rate limit retry duration: %w", err)
		}
	}

	err = connection.Send("MULTI")
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error starting redis rate limit transaction: %w", err)
	}

	if result.Allowed {
		err = queueRateLimit(connection, request, deviceKey, ipKey)
		if err != nil {
			return vibe.RateLimitResult{}, fmt.Errorf("error queueing redis rate limit: %w", err)
		}
	} else {
		err = connection.Send("HLEN", limitedKey)
		if err != nil {
			return vibe.RateLimitResult{}, fmt.Errorf("error queueing redis rate limit check: %w", err)
		}
	}

	reply, err := redis.DoContext(connection, ctx, "EXEC")
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error executing redis rate limit transaction: %w", err)
	}
	if reply == nil {
		return vibe.RateLimitResult{}, rateLimitTransactionConflictError{Reason: "rate limit keys changed"}
	}

	err = validateRateLimitTransaction(reply)
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error validating redis rate limit transaction: %w", err)
	}

	return result, nil
}

func queueRateLimit(connection redis.Conn, request vibe.RateLimitRequest, deviceKey string, ipKey string) error {
	member := uuid.NewString()
	expirationMilliseconds := max(request.Rate.Milliseconds(), int64(1))

	err := connection.Send("HSET", deviceKey, member, 1)
	if err != nil {
		return fmt.Errorf("error queueing device rate limit: %w", err)
	}
	err = connection.Send("HPEXPIRE", deviceKey, expirationMilliseconds, "FIELDS", 1, member)
	if err != nil {
		return fmt.Errorf("error queueing device rate limit expiration: %w", err)
	}
	err = connection.Send("HSET", ipKey, member, 1)
	if err != nil {
		return fmt.Errorf("error queueing IP rate limit: %w", err)
	}
	err = connection.Send("HPEXPIRE", ipKey, expirationMilliseconds, "FIELDS", 1, member)
	if err != nil {
		return fmt.Errorf("error queueing IP rate limit expiration: %w", err)
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

func validateRateLimitTransaction(reply interface{}) error {
	replies, err := redis.Values(reply, nil)
	if err != nil {
		return fmt.Errorf("error reading redis transaction response: %w", err)
	}
	for _, commandReply := range replies {
		commandError, ok := commandReply.(redis.Error)
		if ok {
			return fmt.Errorf("error executing queued redis command: %w", commandError)
		}
	}

	return nil
}

type rateLimitTransactionConflictError struct {
	Reason string
}

func (e rateLimitTransactionConflictError) Error() string {
	return e.Reason
}

const rateLimitTransactionAttempts = 16
