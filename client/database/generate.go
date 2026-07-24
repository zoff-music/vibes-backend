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
		INSERT INTO room_generations (
			room_id,
			prompt,
			attempt,
			created_at,
			updated_at
		)
		SELECT $1, $2, 0, NOW(), NOW()
		FROM rooms
		WHERE id = $1
		AND (
			SELECT COUNT(*)
			FROM songs
			WHERE room_id = $1
		) <= $3
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

	result, err := c.CreateRoomGenerationStatement.ExecContext(
		cctx,
		roomID,
		prompt,
		vibe.RoomGenerationMaxExistingSongs,
	)
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

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting affected rows in CreateRoomGeneration: %w", err)
	}
	if rowsAffected == 0 {
		return internalerror.ErrRoomGenerationSongLimit{
			Err: fmt.Errorf(
				"error validating song count in CreateRoomGeneration: room has more than %d songs",
				vibe.RoomGenerationMaxExistingSongs,
			),
		}
	}

	return nil
}

const roomGenerationSingleActiveConstraint = "room_generations_single_active_idx"

func (c *Client) prepareProcessNextRoomGenerationStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH locked_generation_q AS (
			SELECT
				a.room_id,
				a.prompt,
				a.attempt
			FROM room_generations a
			WHERE a.attempt >= $1
			OR a.attempt = 0
			OR a.updated_at <= NOW() - INTERVAL '5 minutes'
			ORDER BY a.updated_at ASC, a.created_at ASC
			LIMIT 1
			FOR UPDATE OF a SKIP LOCKED
		),
		claimed_generation_q AS (
			UPDATE room_generations a
			SET attempt = a.attempt + 1,
				updated_at = NOW()
			FROM locked_generation_q b
			WHERE a.room_id = b.room_id
			AND b.attempt < $1
			RETURNING
				a.room_id,
				a.prompt,
				a.attempt,
				FALSE AS exhausted
		),
		deleted_generation_q AS (
			DELETE FROM room_generations a
			USING locked_generation_q b
			WHERE a.room_id = b.room_id
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
		FROM deleted_generation_q
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

func (c *Client) prepareDeleteRoomGenerationStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM room_generations
		WHERE room_id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeleteRoomGenerationStatement: %w", err)
	}

	c.DeleteRoomGenerationStatement = stmt

	return nil
}

func (c *Client) DeleteRoomGeneration(ctx context.Context, roomID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "DeleteRoomGeneration")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.DeleteRoomGenerationStatement.ExecContext(cctx, roomID)
	if err != nil {
		return fmt.Errorf("error deleting room generation: %w", err)
	}

	return nil
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
