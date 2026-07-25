package database

import (
	"context"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareGetStatsStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH room_listener_counts AS (
			SELECT
				room_id,
				COUNT(*) FILTER (
					WHERE is_active_listener AND NOT is_cast_receiver
				) AS active_listeners,
				COUNT(*) FILTER (
					WHERE is_cast_receiver
				) AS active_cast_receivers
			FROM room_users
			GROUP BY room_id
		)
		SELECT COALESCE(
			SUM(
				CASE
					WHEN active_listeners = 0 AND active_cast_receivers > 0 THEN 1
					ELSE active_listeners
				END
			),
			0
		)
		FROM room_listener_counts
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetStatsStatement: %w", err)
	}
	c.GetStatsStatement = stmt
	return nil
}

// GetStats returns public, service-wide usage statistics.
func (c *Client) GetStats(ctx context.Context) (*vibe.Stats, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetStats")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetStatsStatement.QueryRowContext(cctx)

	var stats vibe.Stats
	err := row.Scan(&stats.TotalListeners)
	if err != nil {
		return nil, fmt.Errorf("error scanning stats in GetStats: %w", err)
	}

	return &stats, nil
}
