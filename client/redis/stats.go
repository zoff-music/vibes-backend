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

func (c *Client) GetCachedStatsV2(ctx context.Context) (*vibe.CachedStatsV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedStatsV2")
	defer span.End()

	cachedStats := &vibe.CachedStatsV2{}
	if c.Redis == nil {
		return cachedStats, nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection in GetCachedStatsV2: %w", err)
	}
	defer connection.Close()

	body, err := redis.Bytes(
		redis.DoContext(connection, cctx, "GET", c.statsV2CacheKey()),
	)
	if errors.Is(err, redis.ErrNil) {
		return cachedStats, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting cached stats in GetCachedStatsV2: %w", err)
	}

	err = json.Unmarshal(body, &cachedStats.Stats)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling cached stats in GetCachedStatsV2: %w", err)
	}
	cachedStats.Found = true

	return cachedStats, nil
}

func (c *Client) CacheStatsV2(ctx context.Context, stats vibe.StatsV2) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheStatsV2")
	defer span.End()

	if c.Redis == nil {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return fmt.Errorf("error getting redis connection in CacheStatsV2: %w", err)
	}
	defer connection.Close()

	body, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("error marshaling cached stats in CacheStatsV2: %w", err)
	}

	_, err = redis.DoContext(
		connection,
		cctx,
		"SET",
		c.statsV2CacheKey(),
		body,
		"EX",
		int(statsCacheExpiration.Seconds()),
	)
	if err != nil {
		return fmt.Errorf("error storing cached stats in CacheStatsV2: %w", err)
	}

	return nil
}

func (c *Client) statsV2CacheKey() string {
	key := c.getKeyWithPrefix("stats:v2")

	return key
}

const statsCacheExpiration = 5 * time.Second
