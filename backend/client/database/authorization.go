package database

import (
	"context"
	"fmt"
	"time"

	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"
)

func (c *Client) prepareUpsertExternalAuthStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO external_auth (user_id, provider, code, state, expires_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, provider) DO UPDATE SET
			code = excluded.code,
			state = excluded.state,
			expires_at = excluded.expires_at,
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("error in db: prepare upsert external auth statement: %w", err)
	}
	c.UpsertExternalAuthStatement = stmt
	return nil
}

// UpsertExternalAuth stores or updates external authentication data for a user.
func (c *Client) UpsertExternalAuth(ctx context.Context, userID, provider, code, state string, expiresAt time.Time) error {
	_, err := c.UpsertExternalAuthStatement.ExecContext(ctx, userID, provider, code, state, expiresAt)
	if err != nil {
		return fmt.Errorf("error in db: upsert external auth: %w", err)
	}
	return nil
}

// GetExternalAuths retrieves a list of providers the user has authorized.
func (c *Client) GetExternalAuths(ctx context.Context, userID string) ([]string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetExternalAuths")
	defer span.Finish()

	rows, err := c.DB.QueryContext(ctx, `
		SELECT provider FROM external_auth WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("error in db: get external auths: %w", err)
	}
	defer rows.Close()

	var providers []string
	for rows.Next() {
		var provider string
		if err := rows.Scan(&provider); err != nil {
			return nil, fmt.Errorf("error in db: scan external auth provider: %w", err)
		}
		providers = append(providers, provider)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error in db: iterate external auth providers: %w", err)
	}

	return providers, nil
}

// GetExternalAuth retrieves a specific provider auth for a user.
func (c *Client) GetExternalAuth(ctx context.Context, userID, provider string) (*vibe.ExternalAuth, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetExternalAuth")
	defer span.Finish()

	var auth vibe.ExternalAuth
	err := c.DB.QueryRowContext(ctx, `
		SELECT user_id, provider, code, state, expires_at
		FROM external_auth
		WHERE user_id = ? AND provider = ?
	`, userID, provider).Scan(&auth.UserID, &auth.Provider, &auth.Code, &auth.State, &auth.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("error in db: get external auth: %w", err)
	}
	return &auth, nil
}
