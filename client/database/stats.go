package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareGetStatsV2Stmt() error {
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
			WHERE last_seen_at > NOW() - INTERVAL '15 seconds'
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
		) AS total_listeners,
		(SELECT COUNT(*) FROM playlist_items) AS total_playlist_items,
		(SELECT COUNT(*) FROM rooms) AS total_rooms
		FROM room_listener_counts
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetStatsV2Statement: %w", err)
	}
	c.GetStatsV2Statement = stmt
	return nil
}

// GetStatsV2 returns public, service-wide usage statistics.
func (c *Client) GetStatsV2(ctx context.Context) (*vibe.StatsV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetStatsV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetStatsV2Statement.QueryRowContext(cctx)

	var statsRow statsRow
	err := statsRow.scan(row)
	if err != nil {
		return nil, fmt.Errorf("error scanning stats in GetStatsV2: %w", err)
	}

	stats, err := statsRow.toStatsV2()
	if err != nil {
		return nil, fmt.Errorf("error mapping stats: %w", err)
	}

	return stats, nil
}

type statsRow struct {
	TotalListeners     sql.NullInt64
	TotalPlaylistItems sql.NullInt64
	TotalRooms         sql.NullInt64
}

func (s *statsRow) scan(row *sql.Row) error {
	err := row.Scan(
		&s.TotalListeners,
		&s.TotalPlaylistItems,
		&s.TotalRooms,
	)
	if err != nil {
		return fmt.Errorf("error scanning stats in scan: %w", err)
	}

	return nil
}

func (s *statsRow) toStatsV2() (*vibe.StatsV2, error) {
	return &vibe.StatsV2{
		TotalListeners:     int(s.TotalListeners.Int64),
		TotalPlaylistItems: int(s.TotalPlaylistItems.Int64),
		TotalRooms:         int(s.TotalRooms.Int64),
	}, nil
}

func (c *Client) GetStats(ctx context.Context) (*vibe.Stats, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetStats")
	defer span.End()

	result, err := c.GetStatsV2(ctx)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy statistics: %w", err)
	}

	legacy := result.ToStats()
	return legacy, nil
}
