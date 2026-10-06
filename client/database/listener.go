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
			RETURNING created_at
		), deleted_rooms_q AS (
			DELETE FROM room_listener_usage
			WHERE created_at < NOW() - INTERVAL '32 days'
			RETURNING room_id
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
		), recorded_q AS (
			INSERT INTO listener_usage (listener_count, created_at)
			SELECT listener_count, DATE_TRUNC('minute', NOW())
			FROM listener_usage_q
			WHERE listener_count > 0
			ON CONFLICT (created_at) DO NOTHING
			RETURNING created_at
		), recorded_rooms_q AS (
			INSERT INTO room_listener_usage (room_id, created_at, listener_count)
			SELECT a.room_id, b.created_at,
				CASE
					WHEN a.active_listeners = 0 AND a.active_cast_receivers > 0 THEN 1
					ELSE a.active_listeners
				END
			FROM room_listener_counts_q a
			CROSS JOIN recorded_q b
			RETURNING room_id
		)
		SELECT COUNT(*) FROM recorded_q
		UNION ALL SELECT COUNT(*) FROM recorded_rooms_q
		UNION ALL SELECT COUNT(*) FROM deleted_q
		UNION ALL SELECT COUNT(*) FROM deleted_rooms_q
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
						DATE_TRUNC('hour', NOW(), 'UTC'),
						'minute'::text,
						1
					),
					(
						'day'::text,
						DATE_TRUNC('day', NOW(), 'UTC'),
						'hour'::text,
						2
					),
					(
						'week'::text,
						DATE_TRUNC('week', NOW(), 'UTC'),
						'day'::text,
						3
					),
					(
						'month'::text,
						DATE_TRUNC('month', NOW(), 'UTC'),
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
			DATE_TRUNC(a.granularity, b.created_at, 'UTC') AS recorded_at,
			MAX(b.listener_count) AS listener_count
		FROM windows_q a
		JOIN listener_usage b ON b.created_at >= a.starts_at
		GROUP BY
			a.period,
			a.window_order,
			DATE_TRUNC(a.granularity, b.created_at, 'UTC')
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

		point, err := row.toListenerUsagePoint()
		if err != nil {
			return nil, fmt.Errorf("error converting listener usage in ListAdminListenerUsage: %w", err)
		}

		points = append(points, *point)
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

func (r *listenerUsageRow) toListenerUsagePoint() (*vibe.ListenerUsagePoint, error) {
	return &vibe.ListenerUsagePoint{
		Window:    r.Window.String,
		Timestamp: r.Timestamp.Time,
		Listeners: int(r.Listeners.Int64),
	}, nil
}

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
		SELECT p.period, p.bucket, r.room_id, r.listener_count, rooms.room_type
		FROM peaks_q p
		JOIN room_listener_usage r ON r.created_at = p.created_at
		LEFT JOIN rooms ON rooms.id = r.room_id
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
	RoomType  sql.NullString
	Window    sql.NullString
	Timestamp sql.NullTime
	RoomID    sql.NullString
	Listeners sql.NullInt64
}

func (r *roomListenerUsageRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(&r.Window, &r.Timestamp, &r.RoomID, &r.Listeners, &r.RoomType)
	if err != nil {
		return fmt.Errorf("error scanning room listener usage row: %w", err)
	}

	return nil
}

func (r *roomListenerUsageRow) toRoomListenerUsagePoint() (*vibe.RoomListenerUsagePoint, error) {
	return &vibe.RoomListenerUsagePoint{
		RoomType: vibe.RoomType(r.RoomType.String),
		RoomID:   r.RoomID.String,
		ListenerUsagePoint: vibe.ListenerUsagePoint{
			Window:    r.Window.String,
			Timestamp: r.Timestamp.Time,
			Listeners: int(r.Listeners.Int64),
		},
	}, nil
}
