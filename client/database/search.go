package database

import (
	"context"
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
