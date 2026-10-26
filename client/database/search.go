package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareCreateSearchUsagesStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH deleted_q AS (
			DELETE FROM search_usage
			WHERE created_at < NOW() - INTERVAL '32 DAYS'
			RETURNING id
		), deleted_rooms_q AS (
			DELETE FROM room_search_usage
			WHERE created_at < NOW() - INTERVAL '32 DAYS'
			RETURNING room_id
		), recorded_q AS (
			INSERT INTO search_usage (
				provider,
				query_hash,
				cached,
				search_count,
				created_at
			)
			VALUES ($1, $2, $3, 1, DATE_TRUNC('HOUR', NOW(), 'UTC'))
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

func (c *Client) prepareClaimSearchStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH deleted_q AS (
			DELETE FROM search_claims
			WHERE expires_at < NOW() - INTERVAL '1 DAY'
			AND query_hash != $1
		)
		INSERT INTO search_claims (query_hash, token, expires_at)
		VALUES ($1, $2, NOW() + INTERVAL '60 SECONDS')
		ON CONFLICT (query_hash) DO UPDATE SET
			token = EXCLUDED.token,
			expires_at = EXCLUDED.expires_at
		WHERE search_claims.expires_at <= NOW()
		RETURNING query_hash, token
	`)
	if err != nil {
		return fmt.Errorf("error preparing ClaimSearchStatement: %w", err)
	}

	c.ClaimSearchStatement = stmt

	return nil
}

func (c *Client) ClaimSearch(ctx context.Context, key string, token string) (*vibe.SearchClaim, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ClaimSearch")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.ClaimSearchStatement.QueryRowContext(cctx, key, token)
	var data searchClaimRow
	err := data.scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return &vibe.SearchClaim{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error claiming search: %w", err)
	}

	result := &vibe.SearchClaim{Key: data.Key, Token: data.Token}

	return result, nil
}

type searchClaimRow struct {
	Key   string
	Token string
}

func (r *searchClaimRow) scanRow(row *sql.Row) error {
	err := row.Scan(&r.Key, &r.Token)
	if err != nil {
		return fmt.Errorf("error scanning search claim: %w", err)
	}

	return nil
}

func (c *Client) prepareReleaseSearchStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM search_claims
		WHERE query_hash = $1 AND token = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing ReleaseSearchStatement: %w", err)
	}

	c.ReleaseSearchStatement = stmt

	return nil
}

func (c *Client) ReleaseSearch(ctx context.Context, claim vibe.SearchClaim) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "ReleaseSearch")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.ReleaseSearchStatement.ExecContext(cctx, claim.Key, claim.Token)
	if err != nil {
		return fmt.Errorf("error releasing search claim: %w", err)
	}

	return nil
}

func (c *Client) prepareReserveSearchStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH clock_q AS (
			SELECT (NOW() AT TIME ZONE 'America/Los_Angeles')::DATE AS quota_day,
				((NOW() AT TIME ZONE 'America/Los_Angeles')::DATE::TIMESTAMP AT TIME ZONE 'America/Los_Angeles') AS day_start,
				(((NOW() AT TIME ZONE 'America/Los_Angeles')::DATE + 1)::TIMESTAMP AT TIME ZONE 'America/Los_Angeles') AS reset_at
		),
		claim_q AS (
			SELECT a.id
			FROM search_budgets a
			FOR UPDATE
		),
		budget_q AS (
			UPDATE search_budgets a SET
				quota_day = c.quota_day,
				search_count = CASE WHEN a.quota_day = c.quota_day THEN a.search_count + 1 ELSE 1 END,
				generation_count = CASE WHEN a.quota_day = c.quota_day THEN a.generation_count ELSE 0 END + CASE WHEN $5 = 'generation' THEN 1 ELSE 0 END,
				started_at = NOW()
			FROM claim_q b, clock_q c
			WHERE a.id = b.id
			AND (a.quota_day != c.quota_day OR a.search_count < $6)
			AND ($5 != 'generation' OR (CASE WHEN a.quota_day = c.quota_day THEN a.generation_count ELSE 0 END) < $7)
			AND (SELECT COUNT(*) FROM search_attempts d WHERE d.user_id = $4 AND d.created_at >= c.day_start) < $8
			AND (SELECT COUNT(*) FROM search_attempts e WHERE e.room_id = $3 AND e.created_at >= c.day_start) < $9
			RETURNING a.id
		),
		attempt_q AS (
			INSERT INTO search_attempts (id, query_hash, room_id, user_id, purpose)
			SELECT $1, $2, $3, $4, $5 FROM budget_q
			RETURNING id
		),
		deleted_q AS (
			DELETE FROM search_attempts
			WHERE created_at < NOW() - INTERVAL '30 DAYS'
		)
		SELECT EXISTS(SELECT 1 FROM attempt_q), a.reset_at
		FROM clock_q a
	`)
	if err != nil {
		return fmt.Errorf("error preparing ReserveSearchStatement: %w", err)
	}

	c.ReserveSearchStatement = stmt

	return nil
}

func (c *Client) ReserveSearch(ctx context.Context, attempt vibe.SearchAttempt) (*vibe.SearchAllowance, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ReserveSearch")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.ReserveSearchStatement.QueryRowContext(cctx,
		attempt.ID, attempt.QueryHash, attempt.RoomID, attempt.UserID, attempt.Purpose,
		c.searchDailyLimit, c.generationSearchDailyLimit, c.userSearchDailyLimit, c.roomSearchDailyLimit,
	)
	var data searchAllowanceRow
	err := data.scanRow(row)
	if err != nil {
		return nil, fmt.Errorf("error reserving search: %w", err)
	}

	result := &vibe.SearchAllowance{Allowed: data.Allowed, ResetAt: data.ResetAt}

	return result, nil
}

type searchAllowanceRow struct {
	Allowed bool
	ResetAt time.Time
}

func (r *searchAllowanceRow) scanRow(row *sql.Row) error {
	err := row.Scan(&r.Allowed, &r.ResetAt)
	if err != nil {
		return fmt.Errorf("error scanning search allowance: %w", err)
	}

	return nil
}

func (c *Client) prepareUpdateSearchAttemptStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE search_attempts
		SET outcome = $2
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdateSearchAttemptStatement: %w", err)
	}

	c.UpdateSearchAttemptStatement = stmt

	return nil
}

func (c *Client) UpdateSearchAttempt(ctx context.Context, id string, outcome string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdateSearchAttempt")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.UpdateSearchAttemptStatement.ExecContext(cctx, id, outcome)
	if err != nil {
		return fmt.Errorf("error updating search attempt: %w", err)
	}

	return nil
}
