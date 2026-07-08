package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/opentracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareUpsertAuthTokenStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO auth_tokens (user_id, provider, code, state, expires_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, provider) DO UPDATE SET
			code = EXCLUDED.code,
			state = EXCLUDED.state,
			expires_at = EXCLUDED.expires_at,
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("error in db: prepare upsert auth token statement: %w", err)
	}
	c.UpsertAuthTokenStatement = stmt
	return nil
}

// UpsertAuthToken stores or updates initial auth token data (code/state) for a user.
func (c *Client) UpsertAuthToken(ctx context.Context, userID, provider, code, state string, expiresAt time.Time) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UpsertAuthToken")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.UpsertAuthTokenStatement.ExecContext(cctx, userID, provider, code, state, expiresAt)
	if err != nil {
		return fmt.Errorf("error in db: upsert auth token: %w", err)
	}

	return nil
}

func (c *Client) prepareGetAuthProvidersStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT provider FROM access_tokens WHERE user_id = $1 AND refresh_expires_at > CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetAuthProvidersStatement: %w", err)
	}
	c.GetAuthProvidersStatement = stmt
	return nil
}

// GetAuthProviders retrieves a list of providers the user has authorized.
func (c *Client) GetAuthProviders(ctx context.Context, userID string) ([]string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetAuthProviders")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.GetAuthProvidersStatement.QueryContext(cctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error in db: get auth providers: %w", err)
	}

	defer rows.Close()

	var providers []string
	for rows.Next() {
		var provider string
		err := rows.Scan(&provider)
		if err != nil {
			return nil, fmt.Errorf("error in db: scan auth provider: %w", err)
		}
		providers = append(providers, provider)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error in db: iterate auth providers: %w", err)
	}

	return providers, nil
}

func (c *Client) prepareUpsertAccessTokenStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO access_tokens (user_id, provider, access_token, refresh_token, expires_at, refresh_expires_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, provider) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			expires_at = EXCLUDED.expires_at,
			refresh_expires_at = EXCLUDED.refresh_expires_at,
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("error in db: prepare upsert access token statement: %w", err)
	}
	c.UpsertAccessTokenStatement = stmt
	return nil
}

// UpsertAccessToken stores or updates access token data for a user.
func (c *Client) UpsertAccessToken(ctx context.Context, userID, provider, accessToken, refreshToken string, expiresAt, refreshExpiresAt time.Time) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UpsertAccessToken")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.UpsertAccessTokenStatement.ExecContext(cctx, userID, provider, accessToken, refreshToken, expiresAt, refreshExpiresAt)
	if err != nil {
		return fmt.Errorf("error in db: upsert access token: %w", err)
	}

	return nil
}

func (c *Client) prepareGetAccessTokenStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT user_id, provider, access_token, refresh_token, expires_at, refresh_expires_at
		FROM access_tokens
		WHERE user_id = $1 AND provider = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetAccessTokenStatement: %w", err)
	}
	c.GetAccessTokenStatement = stmt
	return nil
}

// GetAccessToken retrieves specific access tokens for a user.
func (c *Client) GetAccessToken(ctx context.Context, userID, provider string) (*vibe.AccessToken, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetAccessToken")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetAccessTokenStatement.QueryRowContext(cctx, userID, provider)

	var row accessTokenRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrAccessTokenNotFound{
				Err: fmt.Errorf("access token not found for user %s and provider %s", userID, provider),
			}
		}
		return nil, fmt.Errorf("error in db: get access token: %w", err)
	}
	return row.toAccessToken(), nil
}

type accessTokenRow struct {
	UserID           sql.NullString
	Provider         sql.NullString
	AccessToken      sql.NullString
	RefreshToken     sql.NullString
	ExpiresAt        sql.NullTime
	RefreshExpiresAt sql.NullTime
}

func (a *accessTokenRow) scan(rows *sql.Row) error {
	return rows.Scan(
		&a.UserID,
		&a.Provider,
		&a.AccessToken,
		&a.RefreshToken,
		&a.ExpiresAt,
		&a.RefreshExpiresAt,
	)
}

func (a *accessTokenRow) toAccessToken() *vibe.AccessToken {
	return &vibe.AccessToken{
		UserID:           a.UserID.String,
		Provider:         a.Provider.String,
		AccessToken:      a.AccessToken.String,
		RefreshToken:     a.RefreshToken.String,
		ExpiresAt:        a.ExpiresAt.Time,
		RefreshExpiresAt: a.RefreshExpiresAt.Time,
	}
}

func (c *Client) prepareDeleteExpiredAuthTokensStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM auth_tokens WHERE expires_at <= CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeleteExpiredAuthTokensStatement: %w", err)
	}
	c.DeleteExpiredAuthTokensStatement = stmt
	return nil
}

// DeleteExpiredAuthTokens removes all expired auth tokens records from the database.
func (c *Client) DeleteExpiredAuthTokens(ctx context.Context) (int64, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "DeleteExpiredAuthTokens")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.DeleteExpiredAuthTokensStatement.ExecContext(cctx)
	if err != nil {
		return 0, fmt.Errorf("error in db: delete expired auth tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("error in db: get rows affected: %w", err)
	}

	return rowsAffected, nil
}

func (c *Client) prepareDeleteExpiredAccessTokensStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM access_tokens WHERE refresh_expires_at <= CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeleteExpiredAccessTokensStatement: %w", err)
	}
	c.DeleteExpiredAccessTokensStatement = stmt
	return nil
}

// DeleteExpiredAccessTokens removes all expired access tokens records from the database.
func (c *Client) DeleteExpiredAccessTokens(ctx context.Context) (int64, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "DeleteExpiredAccessTokens")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.DeleteExpiredAccessTokensStatement.ExecContext(cctx)
	if err != nil {
		return 0, fmt.Errorf("error in db: delete expired access tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("error in db: get rows affected: %w", err)
	}

	return rowsAffected, nil
}

func (c *Client) prepareSavePendingOAuthStateStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO pending_oauth_state (user_id, state, code_verifier, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT(user_id, state) DO UPDATE SET
			code_verifier = excluded.code_verifier,
			expires_at = excluded.expires_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing SavePendingOAuthStateStatement: %w", err)
	}
	c.SavePendingOAuthStateStatement = stmt
	return nil
}

// SavePendingOAuthState stores a pending OAuth state for a user.
func (c *Client) SavePendingOAuthState(ctx context.Context, userID, state, codeVerifier string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SavePendingOAuthState")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	expiresAt := time.Now().Add(10 * time.Minute)
	_, err := c.SavePendingOAuthStateStatement.ExecContext(cctx, userID, state, codeVerifier, expiresAt)
	if err != nil {
		return fmt.Errorf("error in db: save pending oauth state: %w", err)
	}
	return nil
}

func (c *Client) prepareValidatePendingOAuthStateStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT user_id, COALESCE(code_verifier, '') FROM pending_oauth_state 
		WHERE state = $1 AND expires_at > CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("error preparing ValidatePendingOAuthStateStatement (Select): %w", err)
	}
	c.ValidatePendingOAuthStateStatement = stmt
	return nil
}

func (c *Client) validatePendingOAuthState(ctx context.Context, state string) (*vibe.PendingOAuthState, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validatePendingOAuthState")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.ValidatePendingOAuthStateStatement.QueryRowContext(cctx, state)

	var userID sql.NullString
	var codeVerifier sql.NullString
	err := r.Scan(&userID, &codeVerifier)
	if err != nil {
		if err == sql.ErrNoRows {
			return &vibe.PendingOAuthState{}, nil
		}

		return nil, fmt.Errorf("error in db: validate pending oauth state: %w", err)
	}

	return &vibe.PendingOAuthState{
		UserID:       userID.String,
		CodeVerifier: codeVerifier.String,
	}, nil
}

func (c *Client) prepareDeletePendingOAuthStateStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM pending_oauth_state WHERE user_id = $1 AND state = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeletePendingOAuthStateStatement: %w", err)
	}
	c.DeletePendingOAuthStateStatement = stmt
	return nil
}

func (c *Client) deletePendingOAuthState(ctx context.Context, userID, state string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "deletePendingOAuthState")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.DeletePendingOAuthStateStatement.ExecContext(cctx, userID, state)
	if err != nil {
		return fmt.Errorf("error in db: delete pending oauth state: %w", err)
	}
	return nil
}

// ValidateAndDeletePendingOAuthState checks if the state exists and is valid, then deletes it.
func (c *Client) ValidateAndDeletePendingOAuthState(ctx context.Context, state string) (*vibe.PendingOAuthState, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ValidateAndDeletePendingOAuthState")
	defer span.Finish()

	pendingState, err := c.validatePendingOAuthState(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("error in db: validate pending oauth state: %w", err)
	}

	if !pendingState.IsEmpty() {
		err = c.deletePendingOAuthState(ctx, pendingState.UserID, state)
		if err != nil {
			return nil, fmt.Errorf("error in db: delete pending oauth state: %w", err)
		}
	}

	return pendingState, nil
}

func (c *Client) prepareDeleteExpiredPendingOAuthStatesStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM pending_oauth_state WHERE expires_at <= CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeleteExpiredPendingOAuthStatesStatement: %w", err)
	}
	c.DeleteExpiredPendingOAuthStatesStatement = stmt
	return nil
}

// DeleteExpiredPendingOAuthStates removes all expired pending OAuth states.
func (c *Client) DeleteExpiredPendingOAuthStates(ctx context.Context) (int64, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "DeleteExpiredPendingOAuthStates")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.DeleteExpiredPendingOAuthStatesStatement.ExecContext(cctx)
	if err != nil {
		return 0, fmt.Errorf("error in db: delete expired pending oauth states: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("error in db: get rows affected: %w", err)
	}

	return rowsAffected, nil
}

func (c *Client) prepareClaimAndGetExpiredTokenForRefreshStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH claimed_token AS (
			SELECT user_id, provider, access_token, refresh_token, expires_at, refresh_expires_at
			FROM access_tokens
			WHERE provider = $1
			AND expires_at <= CURRENT_TIMESTAMP
				AND refresh_expires_at > CURRENT_TIMESTAMP
				AND (last_checked IS NULL OR last_checked <= NOW() - INTERVAL '2 minutes')
				ORDER BY last_checked ASC
				LIMIT 1
				FOR UPDATE SKIP LOCKED
			)
		UPDATE access_tokens
		SET last_checked = CURRENT_TIMESTAMP
		WHERE (user_id, provider) IN (SELECT user_id, provider FROM claimed_token)
		RETURNING user_id, provider, access_token, refresh_token, expires_at, refresh_expires_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing ClaimAndGetExpiredTokenForRefreshStatement: %w", err)
	}
	c.ClaimAndGetExpiredTokenForRefreshStatement = stmt
	return nil
}

// ClaimAndGetExpiredTokenForRefresh finds an expired token, claims it by updating last_checked, and returns it.
func (c *Client) ClaimAndGetExpiredTokenForRefresh(ctx context.Context, provider string) (*vibe.AccessToken, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ClaimAndGetExpiredTokenForRefresh")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.ClaimAndGetExpiredTokenForRefreshStatement.QueryRowContext(cctx, provider)

	var row accessTokenRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf("no expired token found for refresh"),
				},
			}
		}
		return nil, fmt.Errorf("error in db: get expired token for refresh: %w", err)
	}

	token := row.toAccessToken()

	return token, nil
}
