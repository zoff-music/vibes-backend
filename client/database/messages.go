package database

import (
	"context"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareCreateMessageUsageStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO chat_usage (room_id, created_at, message_count)
		VALUES ($1, DATE_TRUNC('hour', $2::timestamptz, 'UTC'), 1)
		ON CONFLICT (room_id, created_at)
		DO UPDATE SET message_count = chat_usage.message_count + 1
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateMessageUsageStatement: %w", err)
	}
	c.CreateMessageUsageStatement = stmt
	return nil
}

func (c *Client) CreateMessageUsage(ctx context.Context, roomID string, sentAt time.Time) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateMessageUsage")
	defer span.End()
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := c.CreateMessageUsageStatement.ExecContext(cctx, roomID, sentAt)
	if err != nil {
		return fmt.Errorf("error recording message usage in CreateMessageUsage: %w", err)
	}
	return nil
}

func (c *Client) prepareListAdminMessageUsageStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH windows_q (period, starts_at) AS (
			VALUES
				('hour', DATE_TRUNC('hour', NOW(), 'UTC') - INTERVAL '23 hours'),
				('day', DATE_TRUNC('day', NOW(), 'UTC') - INTERVAL '29 days'),
				('month', DATE_TRUNC('month', NOW(), 'UTC') - INTERVAL '11 months')
		)
		SELECT w.period, DATE_TRUNC(w.period, u.created_at, 'UTC'), SUM(u.message_count)
		FROM windows_q w
		JOIN chat_usage u ON u.created_at >= w.starts_at
		WHERE ($1 = '' OR u.room_id = $1)
		GROUP BY w.period, DATE_TRUNC(w.period, u.created_at, 'UTC')
		UNION ALL
		SELECT 'total', NULL::timestamptz, COALESCE(SUM(message_count), 0)
		FROM chat_usage WHERE ($1 = '' OR room_id = $1)
		ORDER BY 1, 2
	`)
	if err != nil {
		return fmt.Errorf("error preparing ListAdminMessageUsageStatement: %w", err)
	}
	c.ListAdminMessageUsageStatement = stmt
	return nil
}

func (c *Client) ListAdminMessageUsage(ctx context.Context, roomID string) (*vibe.AdminMessageUsage, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ListAdminMessageUsage")
	defer span.End()
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := c.ListAdminMessageUsageStatement.QueryContext(cctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error querying usage in ListAdminMessageUsage: %w", err)
	}
	defer rows.Close()
	usage := &vibe.AdminMessageUsage{
		RoomID:      roomID,
		Points:      make([]vibe.MessageUsagePoint, 0),
		GeneratedAt: time.Now().UTC(),
	}
	for rows.Next() {
		var period string
		var timestamp *time.Time
		var count int64
		err = rows.Scan(&period, &timestamp, &count)
		if err != nil {
			return nil, fmt.Errorf("error scanning usage in ListAdminMessageUsage: %w", err)
		}
		if period == "total" {
			usage.Total = count
			continue
		}
		if timestamp != nil {
			usage.Points = append(usage.Points, vibe.MessageUsagePoint{Window: period, Timestamp: *timestamp, Messages: count})
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating usage in ListAdminMessageUsage: %w", err)
	}
	return usage, nil
}
