package redis

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) GetCachedStats(ctx context.Context) (*vibe.CachedStats, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedStats")
	defer span.End()

	cachedStats := &vibe.CachedStats{}
	if c.Redis == nil {
		return cachedStats, nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection in GetCachedStats: %w", err)
	}
	defer connection.Close()

	body, err := redis.Bytes(
		redis.DoContext(connection, cctx, "GET", c.statsCacheKey()),
	)
	if errors.Is(err, redis.ErrNil) {
		return cachedStats, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting cached stats in GetCachedStats: %w", err)
	}

	err = json.Unmarshal(body, &cachedStats.Stats)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling cached stats in GetCachedStats: %w", err)
	}
	cachedStats.Found = true

	return cachedStats, nil
}

func (c *Client) CacheStats(ctx context.Context, stats vibe.Stats) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheStats")
	defer span.End()

	if c.Redis == nil {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return fmt.Errorf("error getting redis connection in CacheStats: %w", err)
	}
	defer connection.Close()

	body, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("error marshaling cached stats in CacheStats: %w", err)
	}

	_, err = redis.DoContext(
		connection,
		cctx,
		"SET",
		c.statsCacheKey(),
		body,
		"EX",
		int(statsCacheExpiration.Seconds()),
	)
	if err != nil {
		return fmt.Errorf("error storing cached stats in CacheStats: %w", err)
	}

	return nil
}

func (c *Client) statsCacheKey() string {
	key := c.getKeyWithPrefix("stats")

	return key
}

const statsCacheExpiration = 5 * time.Second
