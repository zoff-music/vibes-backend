package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareHasActiveRoomGenerationStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT EXISTS (
			SELECT 1
			FROM room_generations
			WHERE attempt < $1
			AND completed_at IS NULL
			AND failed_at IS NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("error preparing HasActiveRoomGenerationStatement: %w", err)
	}

	c.HasActiveRoomGenerationStatement = stmt

	return nil
}

func (c *Client) HasActiveRoomGeneration(ctx context.Context) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "HasActiveRoomGeneration")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.HasActiveRoomGenerationStatement.QueryRowContext(
		cctx,
		vibe.RoomGenerationMaxAttempts,
	)

	var hasActiveGeneration bool
	err := row.Scan(&hasActiveGeneration)
	if err != nil {
		return false, fmt.Errorf("error scanning active room generation: %w", err)
	}

	return hasActiveGeneration, nil
}

func (c *Client) prepareCreateRoomGenerationStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH room_q AS (
			SELECT
				a.id,
				(
					SELECT COUNT(*)
					FROM songs b
					WHERE b.room_id = a.id
				) AS song_count,
				(
					SELECT COUNT(*)
					FROM room_generations c
					WHERE c.room_id = a.id
					AND c.created_at >= NOW() - INTERVAL '24 hours'
				) AS generation_count
			FROM rooms a
			WHERE a.id = $1
		),
		created_generation_q AS (
			INSERT INTO room_generations (
				room_id,
				prompt,
				attempt,
				created_at,
				updated_at
			)
			SELECT id, $2, 0, NOW(), NOW()
			FROM room_q
			WHERE song_count <= $3
			AND generation_count < $4
			RETURNING room_id
		)
		SELECT CASE
			WHEN NOT EXISTS (SELECT 1 FROM room_q)
				THEN 'room_not_found'
			WHEN (SELECT song_count FROM room_q) > $3
				THEN 'song_limit'
			WHEN (SELECT generation_count FROM room_q) >= $4
				THEN 'daily_limit'
			WHEN EXISTS (SELECT 1 FROM created_generation_q)
				THEN 'created'
			ELSE 'unknown'
		END
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateRoomGenerationStatement: %w", err)
	}

	c.CreateRoomGenerationStatement = stmt

	return nil
}

func (c *Client) CreateRoomGeneration(
	ctx context.Context,
	roomID string,
	prompt string,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateRoomGeneration")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.CreateRoomGenerationStatement.QueryRowContext(
		cctx,
		roomID,
		prompt,
		vibe.RoomGenerationMaxExistingSongs,
		vibe.RoomGenerationMaxDailyCount,
	)

	var outcome string
	err := row.Scan(&outcome)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.ConstraintName == roomGenerationSingleActiveConstraint {
			return internalerror.ErrRoomGenerationBusy{
				Err: fmt.Errorf(
					"error creating room generation in CreateRoomGeneration: %w",
					err,
				),
			}
		}

		return fmt.Errorf(
			"error creating room generation in CreateRoomGeneration: %w",
			err,
		)
	}

	if outcome == createRoomGenerationSongLimit {
		return internalerror.ErrRoomGenerationSongLimit{
			Err: fmt.Errorf(
				"error validating song count in CreateRoomGeneration: room has more than %d songs",
				vibe.RoomGenerationMaxExistingSongs,
			),
		}
	}
	if outcome == createRoomGenerationDailyLimit {
		return internalerror.ErrRoomGenerationDailyLimit{
			Err: fmt.Errorf(
				"error validating daily limit in CreateRoomGeneration: room has reached %d generations",
				vibe.RoomGenerationMaxDailyCount,
			),
		}
	}
	if outcome != createRoomGenerationCreated {
		return fmt.Errorf(
			"error validating outcome in CreateRoomGeneration: received %s",
			outcome,
		)
	}

	return nil
}

const createRoomGenerationCreated = "created"

const createRoomGenerationDailyLimit = "daily_limit"

const createRoomGenerationSongLimit = "song_limit"

const roomGenerationSingleActiveConstraint = "room_generations_single_active_idx"

func (c *Client) prepareProcessNextRoomGenerationStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH locked_generation_q AS (
			SELECT
				a.id,
				a.room_id,
				a.prompt,
				a.attempt
			FROM room_generations a
			WHERE a.completed_at IS NULL
			AND a.failed_at IS NULL
			AND (
				a.attempt >= $1
				OR a.attempt = 0
				OR a.updated_at <= NOW() - INTERVAL '5 minutes'
			)
			ORDER BY a.updated_at ASC, a.created_at ASC
			LIMIT 1
			FOR UPDATE OF a SKIP LOCKED
		),
		claimed_generation_q AS (
			UPDATE room_generations a
			SET attempt = a.attempt + 1,
				updated_at = NOW()
			FROM locked_generation_q b
			WHERE a.id = b.id
			AND b.attempt < $1
			RETURNING
				a.room_id,
				a.prompt,
				a.attempt,
				FALSE AS exhausted
		),
		failed_generation_q AS (
			UPDATE room_generations a
			SET failed_at = NOW(),
				failure_reason = $2,
				updated_at = NOW()
			FROM locked_generation_q b
			WHERE a.id = b.id
			AND b.attempt >= $1
			RETURNING
				a.room_id,
				a.prompt,
				a.attempt,
				TRUE AS exhausted
		)
		SELECT room_id, prompt, attempt, exhausted
		FROM claimed_generation_q
		UNION ALL
		SELECT room_id, prompt, attempt, exhausted
		FROM failed_generation_q
	`)
	if err != nil {
		return fmt.Errorf("error preparing ProcessNextRoomGenerationStatement: %w", err)
	}

	c.ProcessNextRoomGenerationStatement = stmt

	return nil
}

func (c *Client) ProcessNextRoomGeneration(
	ctx context.Context,
) (*vibe.RoomGeneration, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ProcessNextRoomGeneration")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.ProcessNextRoomGenerationStatement.QueryRowContext(
		cctx,
		vibe.RoomGenerationMaxAttempts,
		vibe.RoomGenerationFailure,
	)

	var generationRow roomGenerationRow
	err := generationRow.scan(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf("error no room generation ready for processing"),
				},
			}
		}

		return nil, fmt.Errorf("error scanning room generation: %w", err)
	}

	generation := generationRow.toRoomGeneration()

	return generation, nil
}

type roomGenerationRow struct {
	RoomID    sql.NullString
	Prompt    sql.NullString
	Attempt   sql.NullInt64
	Exhausted sql.NullBool
}

func (r *roomGenerationRow) scan(row *sql.Row) error {
	return row.Scan(
		&r.RoomID,
		&r.Prompt,
		&r.Attempt,
		&r.Exhausted,
	)
}

func (r *roomGenerationRow) toRoomGeneration() *vibe.RoomGeneration {
	return &vibe.RoomGeneration{
		RoomID:    r.RoomID.String,
		Prompt:    r.Prompt.String,
		Attempt:   int(r.Attempt.Int64),
		Exhausted: r.Exhausted.Bool,
	}
}

func (c *Client) prepareCompleteRoomGenerationStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE room_generations
		SET completed_at = NOW(),
			updated_at = NOW()
		WHERE room_id = $1
		AND completed_at IS NULL
		AND failed_at IS NULL
	`)
	if err != nil {
		return fmt.Errorf("error preparing CompleteRoomGenerationStatement: %w", err)
	}

	c.CompleteRoomGenerationStatement = stmt

	return nil
}

func (c *Client) CompleteRoomGeneration(ctx context.Context, roomID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CompleteRoomGeneration")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.CompleteRoomGenerationStatement.ExecContext(cctx, roomID)
	if err != nil {
		return fmt.Errorf("error completing room generation in CompleteRoomGeneration: %w", err)
	}

	return nil
}

func (c *Client) prepareFailRoomGenerationStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE room_generations
		SET attempt = $2,
			failed_at = NOW(),
			failure_reason = $3,
			updated_at = NOW()
		WHERE room_id = $1
		AND completed_at IS NULL
		AND failed_at IS NULL
	`)
	if err != nil {
		return fmt.Errorf("error preparing FailRoomGenerationStatement: %w", err)
	}

	c.FailRoomGenerationStatement = stmt

	return nil
}

func (c *Client) FailRoomGeneration(
	ctx context.Context,
	roomID string,
	reason string,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "FailRoomGeneration")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.FailRoomGenerationStatement.ExecContext(
		cctx,
		roomID,
		vibe.RoomGenerationMaxAttempts,
		reason,
	)
	if err != nil {
		return fmt.Errorf("error failing room generation in FailRoomGeneration: %w", err)
	}

	return nil
}

func (c *Client) prepareDeleteExpiredRoomGenerationsStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM room_generations
		WHERE completed_at <= $1
		OR failed_at <= $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeleteExpiredRoomGenerationsStatement: %w", err)
	}

	c.DeleteExpiredRoomGenerationsStatement = stmt

	return nil
}

func (c *Client) DeleteExpiredRoomGenerations(
	ctx context.Context,
	olderThan time.Duration,
) (int64, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "DeleteExpiredRoomGenerations")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-olderThan)
	result, err := c.DeleteExpiredRoomGenerationsStatement.ExecContext(cctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf(
			"error deleting expired room generations in DeleteExpiredRoomGenerations: %w",
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf(
			"error getting affected rows in DeleteExpiredRoomGenerations: %w",
			err,
		)
	}

	return rowsAffected, nil
}

func (c *Client) AddGeneratedSong(
	ctx context.Context,
	song *vibe.Song,
) (*vibe.Song, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AddGeneratedSong")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.AddSongStatement.QueryRowContext(
		cctx,
		song.RoomID,
		string(song.SourceType),
		song.SourceID,
		song.Title,
		song.Artist,
		song.ThumbnailURL,
		song.Duration,
		song.AddedBy,
		song.AddedByNickname,
		song.ID,
		false,
	)

	var rowData addSongRow
	err := rowData.scan(row)
	if err != nil {
		return nil, fmt.Errorf("error scanning generated song: %w", err)
	}

	if rowData.Result.String == addSongResultRoomNotFound {
		return nil, fmt.Errorf(
			"error adding generated song: room %s not found",
			song.RoomID,
		)
	}
	if vibe.AddSongOutcome(rowData.Result.String) != vibe.AddSongOutcomeAdded {
		return &vibe.Song{}, nil
	}

	generatedSong := rowData.toSong()

	return &generatedSong, nil
}
