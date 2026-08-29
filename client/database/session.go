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

func (c *Client) prepareGetOrCreateSessionProfileStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH existing_session_q AS MATERIALIZED (
			SELECT name
			FROM sessions
			WHERE id = $1
		),
		generated_range_q AS MATERIALIZED (
			SELECT id AS max_id
			FROM room_name_pool
			WHERE generated
			AND NOT EXISTS (SELECT 1 FROM existing_session_q)
			ORDER BY id DESC
			LIMIT 1
		),
		target_q AS MATERIALIZED (
			SELECT
				MOD(
					hashtextextended($1, 0) & 9223372036854775807,
					max_id
				) + 1 AS id
			FROM generated_range_q
		),
		generated_name_q AS MATERIALIZED (
			SELECT COALESCE(
				(
					SELECT name
					FROM room_name_pool
					WHERE generated
					AND id >= (SELECT id FROM target_q)
					ORDER BY id
					LIMIT 1
				),
				(
					SELECT name
					FROM room_name_pool
					WHERE generated
					AND EXISTS (SELECT 1 FROM target_q)
					ORDER BY id
					LIMIT 1
				)
			) AS name
		),
		inserted_session_q AS (
			INSERT INTO sessions (id, name)
			SELECT
				$1,
				SPLIT_PART(a.name, '-', 1) || '-' || SPLIT_PART(a.name, '-', 2)
			FROM generated_name_q a
			WHERE a.name IS NOT NULL
			ON CONFLICT (id) DO UPDATE
			SET name = sessions.name
			RETURNING name
		)
		SELECT name
		FROM existing_session_q
		UNION ALL
		SELECT name
		FROM inserted_session_q
		LIMIT 1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetOrCreateSessionProfileStatement: %w", err)
	}

	c.GetOrCreateSessionProfileStatement = stmt

	return nil
}

func (c *Client) GetOrCreateSessionProfile(
	ctx context.Context,
	id string,
) (*vibe.SessionProfile, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetOrCreateSessionProfile")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetOrCreateSessionProfileStatement.QueryRowContext(cctx, id)

	var profile vibe.SessionProfile
	err := row.Scan(&profile.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("error creating session profile in GetOrCreateSessionProfile: no generated names available")
		}

		return nil, fmt.Errorf("error scanning session profile in GetOrCreateSessionProfile: %w", err)
	}

	return &profile, nil
}

func (c *Client) prepareUpdateSessionProfileStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO sessions (id, name)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
			updated_at = NOW()
		RETURNING name
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdateSessionProfileStatement: %w", err)
	}

	c.UpdateSessionProfileStatement = stmt

	return nil
}

func (c *Client) UpdateSessionProfile(
	ctx context.Context,
	id string,
	name string,
) (*vibe.SessionProfile, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdateSessionProfile")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.UpdateSessionProfileStatement.QueryRowContext(cctx, id, name)

	var profile vibe.SessionProfile
	err := row.Scan(&profile.Name)
	if err != nil {
		return nil, fmt.Errorf("error scanning session profile in UpdateSessionProfile: %w", err)
	}

	return &profile, nil
}
