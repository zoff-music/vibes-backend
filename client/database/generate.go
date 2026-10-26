package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareCreateGenerationStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO room_generations (owner_id, room_type, prompt, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id::TEXT, ''::TEXT, 'generating'::TEXT, ''::TEXT
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateGenerationStatement: %w", err)
	}

	c.CreateGenerationStatement = stmt

	return nil
}

func (c *Client) CreateGeneration(ctx context.Context, ownerID string, request vibe.GeneratedRoomRequestV2) (*vibe.GenerationStatus, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateGeneration")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.CreateGenerationStatement.QueryRowContext(cctx, ownerID, request.RoomType, request.Prompt)
	var row generationStatusRow
	err := row.scan(r)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.ConstraintName == roomGenerationSingleActiveConstraint {
			return nil, internalerror.ErrRoomGenerationBusy{
				Err: fmt.Errorf("error creating generation: %w", err),
			}
		}

		return nil, fmt.Errorf("error scanning created generation: %w", err)
	}

	status, err := row.toGenerationStatus()
	if err != nil {
		return nil, fmt.Errorf("error mapping created generation: %w", err)
	}

	return status, nil
}

func (c *Client) prepareGetGenerationStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT a.id::TEXT,
			COALESCE(a.room_id, ''),
			CASE
				WHEN a.failed_at IS NOT NULL THEN 'failed'
				WHEN a.completed_at IS NOT NULL THEN 'completed'
				ELSE 'generating'
			END,
			COALESCE(a.failure_reason, '')
		FROM room_generations a
		WHERE a.id = $1 AND a.owner_id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetGenerationStatement: %w", err)
	}

	c.GetGenerationStatement = stmt

	return nil
}

func (c *Client) GetGeneration(ctx context.Context, id int64, ownerID string) (*vibe.GenerationStatus, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetGeneration")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetGenerationStatement.QueryRowContext(cctx, id, ownerID)
	var row generationStatusRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.GenerationStatus{}, nil
		}

		return nil, fmt.Errorf("error scanning generation: %w", err)
	}

	status, err := row.toGenerationStatus()
	if err != nil {
		return nil, fmt.Errorf("error mapping generation status: %w", err)
	}

	return status, nil
}

type generationStatusRow struct {
	ID     string
	RoomID string
	Status string
	Error  string
}

func (r *generationStatusRow) scan(row *sql.Row) error {
	err := row.Scan(&r.ID, &r.RoomID, &r.Status, &r.Error)
	if err != nil {
		return fmt.Errorf("error scanning generation status: %w", err)
	}

	return nil
}

func (r *generationStatusRow) toGenerationStatus() (*vibe.GenerationStatus, error) {
	return &vibe.GenerationStatus{
		ID:     r.ID,
		RoomID: r.RoomID,
		Status: r.Status,
		Error:  vibe.PublicGenerationError(r.Error),
	}, nil
}

func (c *Client) prepareCreateGenerationRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH locked_generation_q AS (
			SELECT a.id, a.owner_id, a.room_type, a.room_id
			FROM room_generations a
			WHERE a.id = $1 AND a.attempt = $2
			AND a.completed_at IS NULL AND a.failed_at IS NULL
			FOR UPDATE OF a
		),
		created_room_q AS (
			INSERT INTO rooms (id, name, mode, host_id, admin_password_hash, created_at, room_type)
			SELECT $3, $3, 'server', a.owner_id, '', NOW(), a.room_type
			FROM locked_generation_q a
			WHERE a.room_id IS NULL
			AND NOT EXISTS (
				SELECT 1 FROM room_name_pool b
				WHERE b.name = $3 AND b.consumed_at IS NOT NULL
			)
			AND NOT EXISTS (
				SELECT 1 FROM room_name_reservations c
				WHERE c.name = $3 AND c.expires_at > NOW()
				AND NOT (c.token::TEXT = $4 AND c.owner_id = a.owner_id)
			)
			ON CONFLICT (id) DO NOTHING
			RETURNING id
		),
		created_settings_q AS (
			INSERT INTO room_settings (
				room_id, skip_allowed, democratic_skip, skip_vote_threshold,
				max_continuous_adds, remove_on_play, allow_duplicates,
				enabled_sources, only_admin_add_playlist_items, is_public, playlist_import
			)
			SELECT a.id, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
			FROM created_room_q a
		),
		consumed_name_q AS (
			INSERT INTO room_name_pool (name, generated, consumed_at)
			SELECT a.id, FALSE, NOW() FROM created_room_q a
			ON CONFLICT (name) DO UPDATE SET consumed_at = EXCLUDED.consumed_at
		),
		deleted_reservation_q AS (
			DELETE FROM room_name_reservations a
			USING created_room_q b
			WHERE a.name = b.id AND a.token::TEXT = $4
		),
		updated_generation_q AS (
			UPDATE room_generations a
			SET room_id = b.id, updated_at = NOW()
			FROM created_room_q b
			WHERE a.id = $1
			RETURNING a.room_id
		)
		SELECT room_id FROM updated_generation_q
		UNION ALL
		SELECT room_id FROM locked_generation_q WHERE room_id IS NOT NULL
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateGenerationRoomStatement: %w", err)
	}

	c.CreateGenerationRoomStatement = stmt

	return nil
}

func (c *Client) CreateGenerationRoom(ctx context.Context, generation *vibe.RoomGeneration, names []string) (*vibe.RoomV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateGenerationRoom")
	defer span.End()

	settings, err := vibe.DefaultRoomSettingsV2()
	if err != nil {
		return nil, fmt.Errorf("error getting generated room settings: %w", err)
	}
	if generation.RoomType == vibe.RoomTypeWatch {
		settings.EnabledSources = []string{vibe.SourceTypeYouTube}
	}

	candidates := vibe.GeneratedRoomNames(names)
	// One final attempt uses the existing random-name reservation mechanism.
	candidates = append(candidates, "")

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for _, name := range candidates {
		token := ""
		if name == "" {
			reservation, err := c.ReserveSuggestedRoomName(ctx, generation.OwnerID)
			if err != nil {
				return nil, fmt.Errorf("error reserving fallback generation name: %w", err)
			}

			name = reservation.Name
			token = reservation.Token
		}

		r := c.CreateGenerationRoomStatement.QueryRowContext(cctx,
			generation.ID, generation.Attempt, name, token,
			settings.SkipAllowed, settings.DemocraticSkip, settings.SkipVoteThreshold,
			settings.MaxContinuousAdds, settings.RemoveOnPlay, settings.AllowDuplicates,
			strings.Join(settings.EnabledSources, ","), settings.OnlyAdminAddPlaylistItems,
			settings.Public, settings.PlaylistImport,
		)
		var row generationRoomRow
		err = row.scan(r)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error creating generation room: %w", err)
		}

		room, err := c.GetRoomV2(ctx, row.RoomID, generation.OwnerID)
		if err != nil {
			return nil, fmt.Errorf("error fetching created generation room: %w", err)
		}

		return room, nil
	}

	return nil, fmt.Errorf("error creating generation room: no available name or generation claim")
}

type generationRoomRow struct {
	RoomID string
}

func (r *generationRoomRow) scan(row *sql.Row) error {
	err := row.Scan(&r.RoomID)
	if err != nil {
		return fmt.Errorf("error scanning generation room: %w", err)
	}

	return nil
}

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
		return fmt.Errorf(
			"error preparing statement in prepareHasActiveRoomGenerationStmt: %w",
			err,
		)
	}

	c.HasActiveRoomGenerationStatement = stmt

	return nil
}

func (c *Client) HasActiveRoomGeneration(ctx context.Context) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "HasActiveRoomGeneration")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.HasActiveRoomGenerationStatement.QueryRowContext(
		cctx,
		c.roomGenerationMaxAttempts,
	)

	var row activeRoomGenerationRow
	err := row.scan(r)
	if err != nil {
		return false, fmt.Errorf(
			"error scanning active room generation in HasActiveRoomGeneration: %w",
			err,
		)
	}

	return row.Active, nil
}

type activeRoomGenerationRow struct {
	Active bool
}

func (r *activeRoomGenerationRow) scan(row *sql.Row) error {
	err := row.Scan(&r.Active)
	if err != nil {
		return fmt.Errorf("error scanning active room generation row: %w", err)
	}

	return nil
}

func (c *Client) prepareCreateRoomGenerationStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH room_q AS (
			SELECT
				a.id,
				(
					SELECT COUNT(*)
					FROM playlist_items b
					WHERE b.room_id = a.id
					AND b.source_type = ANY($5::TEXT[])
				) AS playlist_item_count,
				(
					SELECT COUNT(*)
					FROM room_generations c
					WHERE c.room_id = a.id
					AND c.created_at >= NOW() - INTERVAL '24 HOURS'
				) AS generation_count
			FROM rooms a
			WHERE a.id = $1
			FOR KEY SHARE OF a
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
			WHERE playlist_item_count <= $3
			AND generation_count < $4
			RETURNING room_id
		)
		SELECT CASE
			WHEN NOT EXISTS (SELECT 1 FROM room_q)
				THEN 'room_not_found'
			WHEN (SELECT playlist_item_count FROM room_q) > $3
				THEN 'playlist_item_limit'
			WHEN (SELECT generation_count FROM room_q) >= $4
				THEN 'daily_limit'
			WHEN EXISTS (SELECT 1 FROM created_generation_q)
				THEN 'created'
			ELSE 'unknown'
		END
	`)
	if err != nil {
		return fmt.Errorf(
			"error preparing statement in prepareCreateRoomGenerationStmt: %w",
			err,
		)
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

	r := c.CreateRoomGenerationStatement.QueryRowContext(
		cctx,
		roomID,
		prompt,
		c.roomGenerationMaxExistingPlaylistItems,
		c.roomGenerationMaxDailyCount,
		c.enabledProviders,
	)

	var row createRoomGenerationRow
	err := row.scan(r)
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

	outcome := row.Outcome
	if outcome == createRoomGenerationPlaylistItemLimit {
		return internalerror.ErrRoomGenerationPlaylistItemLimit{
			Err: fmt.Errorf(
				"error validating playlist item count in CreateRoomGeneration: room has more than %d playlist items",
				c.roomGenerationMaxExistingPlaylistItems,
			),
		}
	}
	if outcome == createRoomGenerationDailyLimit {
		return internalerror.ErrRoomGenerationDailyLimit{
			Err: fmt.Errorf(
				"error validating daily limit in CreateRoomGeneration: room has reached %d generations",
				c.roomGenerationMaxDailyCount,
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

type createRoomGenerationRow struct {
	Outcome string
}

func (r *createRoomGenerationRow) scan(row *sql.Row) error {
	err := row.Scan(&r.Outcome)
	if err != nil {
		return fmt.Errorf("error scanning created room generation row: %w", err)
	}

	return nil
}

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
				OR a.updated_at <= NOW() - INTERVAL '5 MINUTES'
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
				a.id,
				a.owner_id,
				a.room_type,
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
				a.id,
				a.owner_id,
				a.room_type,
				a.room_id,
				a.prompt,
				a.attempt,
				TRUE AS exhausted
		)
		SELECT id, owner_id, room_type, room_id, prompt, attempt, exhausted
		FROM claimed_generation_q
		UNION ALL
		SELECT id, owner_id, room_type, room_id, prompt, attempt, exhausted
		FROM failed_generation_q
	`)
	if err != nil {
		return fmt.Errorf(
			"error preparing statement in prepareProcessNextRoomGenerationStmt: %w",
			err,
		)
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
		c.roomGenerationMaxAttempts,
		vibe.RoomGenerationFailure,
	)

	var generationRow roomGenerationRow
	err := generationRow.scan(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf(
						"error getting room generation in ProcessNextRoomGeneration: none ready for processing",
					),
				},
			}
		}

		return nil, fmt.Errorf(
			"error scanning room generation in ProcessNextRoomGeneration: %w",
			err,
		)
	}

	generation, err := generationRow.toRoomGeneration()
	if err != nil {
		return nil, fmt.Errorf("error mapping generation: %w", err)
	}

	if generation.Exhausted || generation.RoomID == "" {
		return generation, nil
	}

	room, err := c.GetRoomV2(ctx, generation.RoomID, "")
	if err != nil {
		return nil, fmt.Errorf(
			"error getting room in ProcessNextRoomGeneration: %w",
			err,
		)
	}
	if room.IsEmpty() {
		return nil, fmt.Errorf(
			"error getting room in ProcessNextRoomGeneration: room is empty",
		)
	}

	playbackState, err := c.GetPlaybackStateV2(ctx, generation.RoomID)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting playback state in ProcessNextRoomGeneration: %w",
			err,
		)
	}

	playlistItems, err := c.GetPlaylistItems(ctx, generation.RoomID)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting playlist items in ProcessNextRoomGeneration: %w",
			err,
		)
	}

	generationWithContext := &vibe.RoomGeneration{
		ID:            generation.ID,
		OwnerID:       generation.OwnerID,
		RoomType:      generation.RoomType,
		Room:          *room,
		PlaybackState: *playbackState,
		PlaylistItems: playlistItems,
		RoomID:        generation.RoomID,
		Prompt:        generation.Prompt,
		Attempt:       generation.Attempt,
		Exhausted:     generation.Exhausted,
	}

	return generationWithContext, nil
}

type roomGenerationRow struct {
	ID        int64
	OwnerID   sql.NullString
	RoomType  string
	RoomID    sql.NullString
	Prompt    sql.NullString
	Attempt   sql.NullInt64
	Exhausted sql.NullBool
}

func (r *roomGenerationRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.ID,
		&r.OwnerID,
		&r.RoomType,
		&r.RoomID,
		&r.Prompt,
		&r.Attempt,
		&r.Exhausted,
	)
	if err != nil {
		return fmt.Errorf("error scanning room generation in scan: %w", err)
	}

	return nil
}

func (r *roomGenerationRow) toRoomGeneration() (*vibe.RoomGeneration, error) {
	return &vibe.RoomGeneration{
		ID:        r.ID,
		OwnerID:   r.OwnerID.String,
		RoomType:  vibe.RoomType(r.RoomType),
		RoomID:    r.RoomID.String,
		Prompt:    r.Prompt.String,
		Attempt:   int(r.Attempt.Int64),
		Exhausted: r.Exhausted.Bool,
	}, nil
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
		return fmt.Errorf(
			"error preparing statement in prepareCompleteRoomGenerationStmt: %w",
			err,
		)
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
		return fmt.Errorf(
			"error preparing statement in prepareFailRoomGenerationStmt: %w",
			err,
		)
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
		c.roomGenerationMaxAttempts,
		vibe.PublicGenerationError(reason),
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
		return fmt.Errorf(
			"error preparing statement in prepareDeleteExpiredRoomGenerationsStmt: %w",
			err,
		)
	}

	c.DeleteExpiredRoomGenerationsStatement = stmt

	return nil
}

func (c *Client) DeleteExpiredRoomGenerations(
	ctx context.Context,
	olderThan time.Duration,
) (int, error) {
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

	count := int(rowsAffected)
	return count, nil
}

func (c *Client) AddGeneratedPlaylistItem(
	ctx context.Context,
	playlistItem *vibe.PlaylistItem,
) (*vibe.PlaylistItem, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AddGeneratedPlaylistItem")
	defer span.End()

	result, err := c.addPlaylistItem(ctx, playlistItem, false)
	if err != nil {
		return nil, fmt.Errorf("error adding generated playlist item in AddGeneratedPlaylistItem: %w", err)
	}
	if result.Outcome != vibe.AddPlaylistItemOutcomeAdded {
		return &vibe.PlaylistItem{}, nil
	}

	return &result.PlaylistItem, nil
}

const createRoomGenerationCreated = "created"
const createRoomGenerationDailyLimit = "daily_limit"
const createRoomGenerationPlaylistItemLimit = "playlist_item_limit"
const roomGenerationSingleActiveConstraint = "room_generations_single_active_idx"
