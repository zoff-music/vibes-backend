package redis

import (
	"context"
	"fmt"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) GetCachedStats(ctx context.Context) (*vibe.CachedStats, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedStats")
	defer span.End()
	cached, err := c.GetCachedStatsV2(ctx)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy cached statistics: %w", err)
	}
	legacy := cached.Stats.ToStats()
	return &vibe.CachedStats{Stats: *legacy, Found: cached.Found}, nil
}
func (c *Client) CacheStats(ctx context.Context, stats vibe.Stats) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheStats")
	defer span.End()
	canonical := stats.ToStatsV2()
	err := c.CacheStatsV2(ctx, *canonical)
	if err != nil {
		return fmt.Errorf("error mapping legacy statistics cache write: %w", err)
	}
	return nil
}
