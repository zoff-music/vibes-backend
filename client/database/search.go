package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareCreateSearchUsagesStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH deleted_q AS (
			DELETE FROM search_usage
			WHERE created_at < NOW() - INTERVAL '32 days'
			RETURNING id
		), deleted_rooms_q AS (
			DELETE FROM room_search_usage
			WHERE created_at < NOW() - INTERVAL '32 days'
			RETURNING room_id
		), recorded_q AS (
			INSERT INTO search_usage (
				provider,
				query_hash,
				cached,
				search_count,
				created_at
			)
			VALUES ($1, $2, $3, 1, DATE_TRUNC('hour', NOW(), 'UTC'))
			ON CONFLICT (provider, query_hash, cached, created_at)
			DO UPDATE SET
				search_count = search_usage.search_count + EXCLUDED.search_count
			RETURNING provider, created_at
		), recorded_rooms_q AS (
			INSERT INTO room_search_usage (room_id, provider, created_at, search_count, cached_count)
			SELECT $4, provider, created_at, 1, CASE WHEN $3 THEN 1 ELSE 0 END
			FROM recorded_q
			ON CONFLICT (room_id, provider, created_at) DO UPDATE SET
				search_count = room_search_usage.search_count + 1,
				cached_count = room_search_usage.cached_count + EXCLUDED.cached_count
			RETURNING room_id
		)
		SELECT COUNT(*) FROM recorded_rooms_q
		UNION ALL SELECT COUNT(*) FROM deleted_q
		UNION ALL SELECT COUNT(*) FROM deleted_rooms_q
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateSearchUsagesStatement: %w", err)
	}

	c.CreateSearchUsagesStatement = stmt
	return nil
}

func (c *Client) CreateSearchUsages(
	ctx context.Context,
	usages []vibe.SearchUsage,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateSearchUsages")
	defer span.End()

	if len(usages) == 0 {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for _, usage := range usages {
		_, err := c.CreateSearchUsagesStatement.ExecContext(cctx, usage.Provider, usage.QueryHash, usage.Cached, usage.RoomID)
		if err != nil {
			return fmt.Errorf("error creating search usages in CreateSearchUsages: %w", err)
		}
	}

	return nil
}

func (c *Client) prepareListAdminSearchUsageStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH usage_q AS (
			SELECT
				'hour'::text AS aggregation_window,
				DATE_TRUNC('hour', created_at, 'UTC') AS bucket,
				provider,
				SUM(search_count) AS total,
				COUNT(DISTINCT query_hash) AS unique_count,
				COALESCE(
					SUM(search_count) FILTER (WHERE cached),
					0
				) AS cached_count,
				COALESCE(
					SUM(search_count) FILTER (WHERE NOT cached),
					0
				) AS live_count
			FROM search_usage
			WHERE created_at >=
				DATE_TRUNC('hour', NOW(), 'UTC') - INTERVAL '23 hours'
			GROUP BY DATE_TRUNC('hour', created_at, 'UTC'), provider

			UNION ALL

			SELECT
				'day'::text AS aggregation_window,
				DATE_TRUNC('day', created_at, 'UTC') AS bucket,
				provider,
				SUM(search_count) AS total,
				COUNT(DISTINCT query_hash) AS unique_count,
				COALESCE(
					SUM(search_count) FILTER (WHERE cached),
					0
				) AS cached_count,
				COALESCE(
					SUM(search_count) FILTER (WHERE NOT cached),
					0
				) AS live_count
			FROM search_usage
			WHERE created_at >=
				DATE_TRUNC('day', NOW(), 'UTC') - INTERVAL '29 days'
			GROUP BY DATE_TRUNC('day', created_at, 'UTC'), provider
		)
		SELECT
			aggregation_window,
			bucket,
			provider,
			total,
			unique_count,
			cached_count,
			live_count
		FROM usage_q
		ORDER BY aggregation_window, bucket, provider
	`)
	if err != nil {
		return fmt.Errorf("error preparing ListAdminSearchUsageStatement: %w", err)
	}

	c.ListAdminSearchUsageStatement = stmt
	return nil
}

func (c *Client) ListAdminSearchUsage(
	ctx context.Context,
) ([]vibe.AdminSearchUsagePoint, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ListAdminSearchUsage")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.ListAdminSearchUsageStatement.QueryContext(cctx)
	if err != nil {
		return nil, fmt.Errorf(
			"error listing admin search usage in ListAdminSearchUsage: %w",
			err,
		)
	}
	defer rows.Close()

	points := make([]vibe.AdminSearchUsagePoint, 0)
	for rows.Next() {
		var row adminSearchUsageRow
		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"error scanning admin search usage in ListAdminSearchUsage: %w",
				err,
			)
		}

		point, err := row.toAdminSearchUsagePoint()
		if err != nil {
			return nil, fmt.Errorf("error converting search usage in ListAdminSearchUsage: %w", err)
		}

		points = append(points, *point)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf(
			"error iterating admin search usage in ListAdminSearchUsage: %w",
			err,
		)
	}

	return points, nil
}

type adminSearchUsageRow struct {
	Window    sql.NullString
	Timestamp sql.NullTime
	Provider  sql.NullString
	Total     sql.NullInt64
	Unique    sql.NullInt64
	Cached    sql.NullInt64
	Live      sql.NullInt64
}

func (r *adminSearchUsageRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(
		&r.Window,
		&r.Timestamp,
		&r.Provider,
		&r.Total,
		&r.Unique,
		&r.Cached,
		&r.Live,
	)
	if err != nil {
		return fmt.Errorf("error scanning admin search usage row: %w", err)
	}

	return nil
}

func (r *adminSearchUsageRow) toAdminSearchUsagePoint() (*vibe.AdminSearchUsagePoint, error) {
	return &vibe.AdminSearchUsagePoint{
		Window:    r.Window.String,
		Timestamp: r.Timestamp.Time,
		Provider:  r.Provider.String,
		Total:     int(r.Total.Int64),
		Unique:    int(r.Unique.Int64),
		Cached:    int(r.Cached.Int64),
		Live:      int(r.Live.Int64),
	}, nil
}
