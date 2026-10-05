package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareListAdminRoomSearchUsageStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH windows_q (period, starts_at) AS (
			VALUES
				('hour', DATE_TRUNC('hour', NOW(), 'UTC') - INTERVAL '23 hours'),
				('day', DATE_TRUNC('day', NOW(), 'UTC') - INTERVAL '29 days')
		)
		SELECT
			w.period,
			DATE_TRUNC(w.period, u.created_at, 'UTC'),
			u.room_id,
			u.provider,
			SUM(u.search_count),
			SUM(u.cached_count)
		FROM windows_q w
		JOIN room_search_usage u ON u.created_at >= w.starts_at
		GROUP BY
			w.period,
			DATE_TRUNC(w.period, u.created_at, 'UTC'),
			u.room_id,
			u.provider
		ORDER BY 1, 2, 3, 4
	`)
	if err != nil {
		return fmt.Errorf("error preparing ListAdminRoomSearchUsageStatement: %w", err)
	}

	c.ListAdminRoomSearchUsageStatement = stmt

	return nil
}

func (c *Client) ListAdminRoomSearchUsage(ctx context.Context) ([]vibe.RoomSearchUsagePoint, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ListAdminRoomSearchUsage")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.ListAdminRoomSearchUsageStatement.QueryContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error querying room search usage: %w", err)
	}

	defer rows.Close()

	points := make([]vibe.RoomSearchUsagePoint, 0)
	for rows.Next() {
		var row roomSearchUsageRow
		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning room search usage: %w", err)
		}

		point, err := row.toRoomSearchUsagePoint()
		if err != nil {
			return nil, fmt.Errorf("error mapping room search usage: %w", err)
		}

		points = append(points, *point)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating room search usage: %w", err)
	}

	return points, nil
}

type roomSearchUsageRow struct {
	Window    sql.NullString
	Timestamp sql.NullTime
	RoomID    sql.NullString
	Provider  sql.NullString
	Total     sql.NullInt64
	Cached    sql.NullInt64
}

func (r *roomSearchUsageRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(&r.Window, &r.Timestamp, &r.RoomID, &r.Provider, &r.Total, &r.Cached)
	if err != nil {
		return fmt.Errorf("error scanning room search usage row: %w", err)
	}

	return nil
}

func (r *roomSearchUsageRow) toRoomSearchUsagePoint() (*vibe.RoomSearchUsagePoint, error) {
	return &vibe.RoomSearchUsagePoint{
		RoomID:    r.RoomID.String,
		Window:    r.Window.String,
		Timestamp: r.Timestamp.Time,
		Provider:  r.Provider.String,
		Total:     int(r.Total.Int64),
		Cached:    int(r.Cached.Int64),
		Live:      int(r.Total.Int64 - r.Cached.Int64),
	}, nil
}
