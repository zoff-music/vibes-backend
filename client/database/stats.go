package database

import (
	"context"
	"database/sql"
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
		(SELECT COUNT(*) FROM songs) AS total_songs,
		(SELECT COUNT(*) FROM rooms) AS total_rooms
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

	var statsRow statsRow
	err := statsRow.scan(row)
	if err != nil {
		return nil, fmt.Errorf("error scanning stats in GetStats: %w", err)
	}

	stats := statsRow.toStats()

	return stats, nil
}

type statsRow struct {
	TotalListeners sql.NullInt64
	TotalSongs     sql.NullInt64
	TotalRooms     sql.NullInt64
}

func (s *statsRow) scan(row *sql.Row) error {
	err := row.Scan(
		&s.TotalListeners,
		&s.TotalSongs,
		&s.TotalRooms,
	)
	if err != nil {
		return fmt.Errorf("error scanning stats in scan: %w", err)
	}

	return nil
}

func (s *statsRow) toStats() *vibe.Stats {
	return &vibe.Stats{
		TotalListeners: int(s.TotalListeners.Int64),
		TotalSongs:     int(s.TotalSongs.Int64),
		TotalRooms:     int(s.TotalRooms.Int64),
	}
}
