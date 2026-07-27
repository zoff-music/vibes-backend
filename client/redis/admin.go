package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) GetCachedAdminSearchUsage(
	ctx context.Context,
) (*vibe.CachedAdminSearchUsage, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedAdminSearchUsage")
	defer span.End()

	cachedUsage := &vibe.CachedAdminSearchUsage{}
	if c.Redis == nil {
		return cachedUsage, nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting redis connection in GetCachedAdminSearchUsage: %w",
			err,
		)
	}
	defer connection.Close()

	body, err := redis.Bytes(
		redis.DoContext(
			connection,
			cctx,
			"GET",
			c.adminSearchUsageCacheKey(),
		),
	)
	if errors.Is(err, redis.ErrNil) {
		return cachedUsage, nil
	}
	if err != nil {
		return nil, fmt.Errorf(
			"error getting cached admin search usage in GetCachedAdminSearchUsage: %w",
			err,
		)
	}

	err = json.Unmarshal(body, &cachedUsage.Usage)
	if err != nil {
		return nil, fmt.Errorf(
			"error unmarshaling cached admin search usage in GetCachedAdminSearchUsage: %w",
			err,
		)
	}
	cachedUsage.Found = true

	return cachedUsage, nil
}

func (c *Client) CacheAdminSearchUsage(
	ctx context.Context,
	usage vibe.AdminSearchUsage,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheAdminSearchUsage")
	defer span.End()

	if c.Redis == nil {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return fmt.Errorf(
			"error getting redis connection in CacheAdminSearchUsage: %w",
			err,
		)
	}
	defer connection.Close()

	body, err := json.Marshal(usage)
	if err != nil {
		return fmt.Errorf(
			"error marshaling cached admin search usage in CacheAdminSearchUsage: %w",
			err,
		)
	}

	_, err = redis.DoContext(
		connection,
		cctx,
		"SET",
		c.adminSearchUsageCacheKey(),
		body,
		"EX",
		int(adminSearchUsageCacheExpiration.Seconds()),
	)
	if err != nil {
		return fmt.Errorf(
			"error storing cached admin search usage in CacheAdminSearchUsage: %w",
			err,
		)
	}

	return nil
}

func (c *Client) adminSearchUsageCacheKey() string {
	return c.getKeyWithPrefix("admin:searches:usage")
}

const adminSearchUsageCacheExpiration = 30 * time.Second
