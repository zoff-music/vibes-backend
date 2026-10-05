package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareListAdminRoomListenerUsageStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH windows_q (period, starts_at, granularity) AS (
			VALUES
				('hour', DATE_TRUNC('hour', NOW(), 'UTC'), 'minute'),
				('day', DATE_TRUNC('day', NOW(), 'UTC'), 'hour'),
				('week', DATE_TRUNC('week', NOW(), 'UTC'), 'day'),
				('month', DATE_TRUNC('month', NOW(), 'UTC'), 'day')
		), peaks_q AS (
			SELECT DISTINCT ON (w.period, DATE_TRUNC(w.granularity, u.created_at, 'UTC'))
				w.period,
				DATE_TRUNC(w.granularity, u.created_at, 'UTC') AS bucket,
				u.created_at
			FROM windows_q w
			JOIN listener_usage u ON u.created_at >= w.starts_at
			ORDER BY
				w.period,
				DATE_TRUNC(w.granularity, u.created_at, 'UTC'),
				u.listener_count DESC,
				u.created_at DESC
		)
		SELECT p.period, p.bucket, r.room_id, r.listener_count
		FROM peaks_q p
		JOIN room_listener_usage r ON r.created_at = p.created_at
		ORDER BY 1, 2, 3
	`)
	if err != nil {
		return fmt.Errorf("error preparing ListAdminRoomListenerUsageStatement: %w", err)
	}

	c.ListAdminRoomListenerUsageStatement = stmt

	return nil
}

func (c *Client) ListAdminRoomListenerUsage(ctx context.Context) ([]vibe.RoomListenerUsagePoint, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ListAdminRoomListenerUsage")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.ListAdminRoomListenerUsageStatement.QueryContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error querying room listener usage: %w", err)
	}

	defer rows.Close()

	points := make([]vibe.RoomListenerUsagePoint, 0)
	for rows.Next() {
		var row roomListenerUsageRow
		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning room listener usage: %w", err)
		}

		point, err := row.toRoomListenerUsagePoint()
		if err != nil {
			return nil, fmt.Errorf("error mapping room listener usage: %w", err)
		}

		points = append(points, *point)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating room listener usage: %w", err)
	}

	return points, nil
}

type roomListenerUsageRow struct {
	Window    sql.NullString
	Timestamp sql.NullTime
	RoomID    sql.NullString
	Listeners sql.NullInt64
}

func (r *roomListenerUsageRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(&r.Window, &r.Timestamp, &r.RoomID, &r.Listeners)
	if err != nil {
		return fmt.Errorf("error scanning room listener usage row: %w", err)
	}

	return nil
}

func (r *roomListenerUsageRow) toRoomListenerUsagePoint() (*vibe.RoomListenerUsagePoint, error) {
	return &vibe.RoomListenerUsagePoint{
		RoomID: r.RoomID.String,
		ListenerUsagePoint: vibe.ListenerUsagePoint{
			Window:    r.Window.String,
			Timestamp: r.Timestamp.Time,
			Listeners: int(r.Listeners.Int64),
		},
	}, nil
}
