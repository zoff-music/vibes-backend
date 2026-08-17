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

func (c *Client) GetCachedAdminListenerUsage(
	ctx context.Context,
) (*vibe.CachedAdminListenerUsage, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedAdminListenerUsage")
	defer span.End()

	cachedUsage := &vibe.CachedAdminListenerUsage{}
	if c.Redis == nil {
		return cachedUsage, nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting redis connection in GetCachedAdminListenerUsage: %w",
			err,
		)
	}
	defer connection.Close()

	body, err := redis.Bytes(
		redis.DoContext(
			connection,
			cctx,
			"GET",
			c.adminListenerUsageCacheKey(),
		),
	)
	if errors.Is(err, redis.ErrNil) {
		return cachedUsage, nil
	}
	if err != nil {
		return nil, fmt.Errorf(
			"error getting cached admin listener usage in GetCachedAdminListenerUsage: %w",
			err,
		)
	}

	err = json.Unmarshal(body, &cachedUsage.Usage)
	if err != nil {
		return nil, fmt.Errorf(
			"error unmarshaling cached admin listener usage in GetCachedAdminListenerUsage: %w",
			err,
		)
	}
	cachedUsage.Found = true

	return cachedUsage, nil
}

func (c *Client) CacheAdminListenerUsage(
	ctx context.Context,
	usage vibe.AdminListenerUsage,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheAdminListenerUsage")
	defer span.End()

	if c.Redis == nil {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return fmt.Errorf(
			"error getting redis connection in CacheAdminListenerUsage: %w",
			err,
		)
	}
	defer connection.Close()

	body, err := json.Marshal(usage)
	if err != nil {
		return fmt.Errorf(
			"error marshaling cached admin listener usage in CacheAdminListenerUsage: %w",
			err,
		)
	}

	_, err = redis.DoContext(
		connection,
		cctx,
		"SET",
		c.adminListenerUsageCacheKey(),
		body,
		"EX",
		int(adminUsageCacheExpiration.Seconds()),
	)
	if err != nil {
		return fmt.Errorf(
			"error storing cached admin listener usage in CacheAdminListenerUsage: %w",
			err,
		)
	}

	return nil
}

func (c *Client) adminListenerUsageCacheKey() string {
	key := c.getKeyWithPrefix("admin:listeners:usage")
	return key
}
