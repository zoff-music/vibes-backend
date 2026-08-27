package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
)

func (c *Client) GetProviderQuotaReset(
	ctx context.Context,
	provider string,
	operation string,
) (time.Time, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetProviderQuotaReset")
	defer span.End()

	key := c.providerQuotaKey(provider, operation)
	if c.Redis == nil || key == "" {
		return time.Time{}, nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"error getting redis connection in GetProviderQuotaReset: %w",
			err,
		)
	}
	defer connection.Close()

	resetMilliseconds, err := redigo.Int64(redigo.DoContext(
		connection,
		cctx,
		"GET",
		key,
	))
	if errors.Is(err, redigo.ErrNil) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"error getting provider quota reset in GetProviderQuotaReset: %w",
			err,
		)
	}

	return time.UnixMilli(resetMilliseconds), nil
}

func (c *Client) CacheProviderQuotaReset(
	ctx context.Context,
	provider string,
	operation string,
	resetAt time.Time,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheProviderQuotaReset")
	defer span.End()

	key := c.providerQuotaKey(provider, operation)
	if c.Redis == nil || key == "" || !resetAt.After(time.Now()) {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return fmt.Errorf(
			"error getting redis connection in CacheProviderQuotaReset: %w",
			err,
		)
	}
	defer connection.Close()

	_, err = redigo.DoContext(
		connection,
		cctx,
		"SET",
		key,
		resetAt.UnixMilli(),
		"PXAT",
		resetAt.UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf(
			"error caching provider quota reset in CacheProviderQuotaReset: %w",
			err,
		)
	}

	return nil
}

func (c *Client) providerQuotaKey(provider string, operation string) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	operation = strings.TrimSpace(strings.ToLower(operation))
	if provider == "" || operation == "" {
		return ""
	}

	return c.getKeyWithPrefix(
		providerQuotaKeyPrefix + ":" + provider + ":" + operation,
	)
}

const providerQuotaKeyPrefix = "provider:quota"
