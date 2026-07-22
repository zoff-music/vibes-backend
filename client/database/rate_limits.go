package database

import (
	"context"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareConsumeRateLimitStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH device_q AS (
			INSERT INTO rate_limits (
				route_name,
				scope,
				identity_hash,
				request_count,
				request_limit,
				window_started_at,
				expires_at
			)
			VALUES (
				$1,
				'device',
				$2,
				1,
				$5,
				CURRENT_TIMESTAMP AT TIME ZONE 'UTC',
				(CURRENT_TIMESTAMP AT TIME ZONE 'UTC') + ($4::bigint * INTERVAL '1 millisecond')
			)
			ON CONFLICT (route_name, scope, identity_hash) DO UPDATE
			SET request_count = CASE
					WHEN rate_limits.expires_at <= CURRENT_TIMESTAMP AT TIME ZONE 'UTC' THEN 1
					ELSE LEAST(rate_limits.request_count + 1, EXCLUDED.request_limit + 1)
				END,
				request_limit = EXCLUDED.request_limit,
				window_started_at = CASE
					WHEN rate_limits.expires_at <= CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
						THEN CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
					ELSE rate_limits.window_started_at
				END,
				expires_at = CASE
					WHEN rate_limits.expires_at <= CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
						THEN (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') + ($4::bigint * INTERVAL '1 millisecond')
					ELSE rate_limits.expires_at
				END
			RETURNING request_count, request_limit, expires_at
		),
		ip_q AS (
			INSERT INTO rate_limits (
				route_name,
				scope,
				identity_hash,
				request_count,
				request_limit,
				window_started_at,
				expires_at
			)
			SELECT
				$1,
				'ip',
				$3,
				1,
				$6,
				CURRENT_TIMESTAMP AT TIME ZONE 'UTC',
				(CURRENT_TIMESTAMP AT TIME ZONE 'UTC') + ($4::bigint * INTERVAL '1 millisecond')
			FROM device_q
			WHERE request_count <= request_limit
			ON CONFLICT (route_name, scope, identity_hash) DO UPDATE
			SET request_count = CASE
					WHEN rate_limits.expires_at <= CURRENT_TIMESTAMP AT TIME ZONE 'UTC' THEN 1
					ELSE LEAST(rate_limits.request_count + 1, EXCLUDED.request_limit + 1)
				END,
				request_limit = EXCLUDED.request_limit,
				window_started_at = CASE
					WHEN rate_limits.expires_at <= CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
						THEN CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
					ELSE rate_limits.window_started_at
				END,
				expires_at = CASE
					WHEN rate_limits.expires_at <= CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
						THEN (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') + ($4::bigint * INTERVAL '1 millisecond')
					ELSE rate_limits.expires_at
				END
			RETURNING request_count, request_limit, expires_at
		),
		consumed_q AS (
			SELECT request_count, request_limit, expires_at FROM device_q
			UNION ALL
			SELECT request_count, request_limit, expires_at FROM ip_q
		)
		SELECT
			COALESCE(BOOL_AND(request_count <= request_limit), TRUE),
			COALESCE(
				CEIL(EXTRACT(EPOCH FROM (
					MAX(expires_at - (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'))
						FILTER (WHERE request_count > request_limit)
				)))::bigint,
				0
			)
		FROM consumed_q
	`)
	if err != nil {
		return fmt.Errorf("error preparing ConsumeRateLimitStatement: %w", err)
	}
	c.ConsumeRateLimitStatement = stmt
	return nil
}

func (c *Client) ConsumeRateLimit(ctx context.Context, request vibe.RateLimitRequest) (vibe.RateLimitResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ConsumeRateLimit")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var result vibe.RateLimitResult
	var retryAfterSeconds int64
	err := c.ConsumeRateLimitStatement.QueryRowContext(
		cctx,
		request.RouteName,
		request.DeviceIdentityHash,
		request.IPIdentityHash,
		request.Rate.Milliseconds(),
		request.DeviceLimit,
		request.IPLimit,
	).Scan(&result.Allowed, &retryAfterSeconds)
	if err != nil {
		return vibe.RateLimitResult{}, fmt.Errorf("error consuming rate limit: %w", err)
	}

	result.RetryAfter = time.Duration(retryAfterSeconds) * time.Second
	return result, nil
}

func (c *Client) prepareDeleteExpiredRateLimitsStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM rate_limits
		WHERE expires_at <= CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeleteExpiredRateLimitsStatement: %w", err)
	}
	c.DeleteExpiredRateLimitsStatement = stmt
	return nil
}

func (c *Client) DeleteExpiredRateLimits(ctx context.Context) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "DeleteExpiredRateLimits")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.DeleteExpiredRateLimitsStatement.ExecContext(cctx)
	if err != nil {
		return fmt.Errorf("error deleting expired rate limits: %w", err)
	}

	return nil
}
