package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareCreateListenerUsageStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH deleted_q AS (
			DELETE FROM listener_usage
			WHERE created_at < NOW() - INTERVAL '32 days'
		),
		room_listener_counts_q AS (
			SELECT
				room_id,
				COUNT(*) FILTER (
					WHERE is_active_listener
					AND NOT is_cast_receiver
				) AS active_listeners,
				COUNT(*) FILTER (
					WHERE is_cast_receiver
				) AS active_cast_receivers
			FROM room_users
			WHERE last_seen_at > NOW() - INTERVAL '15 seconds'
			GROUP BY room_id
		),
		listener_usage_q AS (
			SELECT COALESCE(
				SUM(
					CASE
						WHEN active_listeners = 0
							AND active_cast_receivers > 0
							THEN 1
						ELSE active_listeners
					END
				),
				0
			) AS listener_count
			FROM room_listener_counts_q
		)
		INSERT INTO listener_usage (
			listener_count,
			created_at
		)
		SELECT
			listener_count,
			DATE_TRUNC('minute', NOW())
		FROM listener_usage_q
		WHERE listener_count > 0
		ON CONFLICT (created_at)
		DO UPDATE SET
			listener_count = EXCLUDED.listener_count
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateListenerUsageStatement: %w", err)
	}

	c.CreateListenerUsageStatement = stmt
	return nil
}

func (c *Client) CreateListenerUsage(ctx context.Context) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateListenerUsage")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.CreateListenerUsageStatement.ExecContext(cctx)
	if err != nil {
		return fmt.Errorf(
			"error creating listener usage in CreateListenerUsage: %w",
			err,
		)
	}

	return nil
}

func (c *Client) prepareListAdminListenerUsageStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH windows_q AS (
			SELECT *
			FROM (
				VALUES
					(
						'hour'::text,
						DATE_TRUNC('hour', NOW()),
						'minute'::text,
						1
					),
					(
						'day'::text,
						DATE_TRUNC('day', NOW()),
						'hour'::text,
						2
					),
					(
						'week'::text,
						DATE_TRUNC('week', NOW()),
						'day'::text,
						3
					),
					(
						'month'::text,
						DATE_TRUNC('month', NOW()),
						'day'::text,
						4
					)
			) AS a(
				period,
				starts_at,
				granularity,
				window_order
			)
		)
		SELECT
			a.period,
			DATE_TRUNC(a.granularity, b.created_at) AS recorded_at,
			MAX(b.listener_count) AS listener_count
		FROM windows_q a
		JOIN listener_usage b ON b.created_at >= a.starts_at
		GROUP BY
			a.period,
			a.window_order,
			DATE_TRUNC(a.granularity, b.created_at)
		ORDER BY
			a.window_order,
			recorded_at
	`)
	if err != nil {
		return fmt.Errorf(
			"error preparing ListAdminListenerUsageStatement: %w",
			err,
		)
	}

	c.ListAdminListenerUsageStatement = stmt
	return nil
}

func (c *Client) ListAdminListenerUsage(
	ctx context.Context,
) ([]vibe.ListenerUsagePoint, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ListAdminListenerUsage")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.ListAdminListenerUsageStatement.QueryContext(cctx)
	if err != nil {
		return nil, fmt.Errorf(
			"error listing admin listener usage in ListAdminListenerUsage: %w",
			err,
		)
	}
	defer rows.Close()

	points := make([]vibe.ListenerUsagePoint, 0)
	for rows.Next() {
		var row listenerUsageRow
		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"error scanning admin listener usage in ListAdminListenerUsage: %w",
				err,
			)
		}
		points = append(points, row.listenerUsagePoint())
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf(
			"error iterating admin listener usage in ListAdminListenerUsage: %w",
			err,
		)
	}

	return points, nil
}

type listenerUsageRow struct {
	Window    sql.NullString
	Timestamp sql.NullTime
	Listeners sql.NullInt64
}

func (r *listenerUsageRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(
		&r.Window,
		&r.Timestamp,
		&r.Listeners,
	)
	if err != nil {
		return fmt.Errorf("error scanning listener usage row: %w", err)
	}

	return nil
}

func (r *listenerUsageRow) listenerUsagePoint() vibe.ListenerUsagePoint {
	return vibe.ListenerUsagePoint{
		Window:    r.Window.String,
		Timestamp: r.Timestamp.Time,
		Listeners: r.Listeners.Int64,
	}
}
