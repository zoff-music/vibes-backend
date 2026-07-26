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
		)
		INSERT INTO search_usage (
			provider,
			query_hash,
			cached,
			search_count,
			created_at
		)
		SELECT
			a.provider,
			a.query_hash,
			a.cached,
			COUNT(*),
			DATE_TRUNC('hour', NOW())
		FROM UNNEST(
			$1::text[],
			$2::text[],
			$3::boolean[]
		) AS a(provider, query_hash, cached)
		GROUP BY
			a.provider,
			a.query_hash,
			a.cached
		ON CONFLICT (
			provider,
			query_hash,
			cached,
			created_at
		)
		DO UPDATE SET
			search_count = search_usage.search_count + EXCLUDED.search_count
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

	providers := make([]string, 0, len(usages))
	queryHashes := make([]string, 0, len(usages))
	cached := make([]bool, 0, len(usages))
	for _, usage := range usages {
		providers = append(providers, usage.Provider)
		queryHashes = append(queryHashes, usage.QueryHash)
		cached = append(cached, usage.Cached)
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.CreateSearchUsagesStatement.ExecContext(
		cctx,
		providers,
		queryHashes,
		cached,
	)
	if err != nil {
		return fmt.Errorf("error creating search usages in CreateSearchUsages: %w", err)
	}

	return nil
}

func (c *Client) prepareListAdminSearchUsageStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH windows_q AS (
			SELECT *
			FROM (
				VALUES
					('hour'::text, DATE_TRUNC('hour', NOW()), 1),
					('day'::text, DATE_TRUNC('day', NOW()), 2),
					('week'::text, DATE_TRUNC('week', NOW()), 3),
					('month'::text, DATE_TRUNC('month', NOW()), 4)
			) AS a(period, starts_at, window_order)
		)
		SELECT
			a.period,
			b.provider,
			SUM(b.search_count) AS total,
			COUNT(DISTINCT b.query_hash) AS unique_count,
			COALESCE(SUM(b.search_count) FILTER (WHERE b.cached), 0) AS cached_count,
			COALESCE(SUM(b.search_count) FILTER (WHERE NOT b.cached), 0) AS live_count
		FROM windows_q a
		JOIN search_usage b ON b.created_at >= a.starts_at
		GROUP BY a.period, a.window_order, b.provider
		ORDER BY a.window_order, b.provider
	`)
	if err != nil {
		return fmt.Errorf("error preparing ListAdminSearchUsageStatement: %w", err)
	}

	c.ListAdminSearchUsageStatement = stmt
	return nil
}

func (c *Client) ListAdminSearchUsage(
	ctx context.Context,
) ([]vibe.AdminSearchUsageSummary, error) {
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

	summaries := make([]vibe.AdminSearchUsageSummary, 0)
	for rows.Next() {
		var row adminSearchUsageRow
		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"error scanning admin search usage in ListAdminSearchUsage: %w",
				err,
			)
		}
		summaries = append(summaries, row.adminSearchUsageSummary())
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf(
			"error iterating admin search usage in ListAdminSearchUsage: %w",
			err,
		)
	}

	return summaries, nil
}

type adminSearchUsageRow struct {
	Window   sql.NullString
	Provider sql.NullString
	Total    sql.NullInt64
	Unique   sql.NullInt64
	Cached   sql.NullInt64
	Live     sql.NullInt64
}

func (r *adminSearchUsageRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(
		&r.Window,
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

func (r *adminSearchUsageRow) adminSearchUsageSummary() vibe.AdminSearchUsageSummary {
	return vibe.AdminSearchUsageSummary{
		Window:   r.Window.String,
		Provider: r.Provider.String,
		Total:    r.Total.Int64,
		Unique:   r.Unique.Int64,
		Cached:   r.Cached.Int64,
		Live:     r.Live.Int64,
	}
}
