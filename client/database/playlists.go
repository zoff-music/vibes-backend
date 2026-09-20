package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareCreatePlaylistImportItemStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO playlist_import_items (
			id, import_id, position, source_type, source_id, provider_url,
			playback_restriction, title, artist, thumbnail_url, duration
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreatePlaylistImportItemStatement: %w", err)
	}

	c.CreatePlaylistImportItemStatement = stmt

	return nil
}

func (c *Client) CreatePlaylistImportItem(ctx context.Context, importID string, position int, song vibe.Song) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreatePlaylistImportItem")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, playlistImportDatabaseTimeout)
	defer cancel()

	_, err := c.CreatePlaylistImportItemStatement.ExecContext(
		cctx, song.ID, importID, position, song.SourceType, song.SourceID,
		song.ProviderURL, song.PlaybackRestriction, song.Title, song.Artist,
		song.ThumbnailURL, song.Duration,
	)
	if err != nil {
		return fmt.Errorf("error creating playlist import item: %w", err)
	}

	return nil
}

func (c *Client) prepareCreatePlaylistImportStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO playlist_imports (id, room_id, added_by)
		SELECT $1, $2, $3
		FROM playlist_import_items
		WHERE import_id = $1
		HAVING COUNT(*) = $4 AND MIN(position) = 0 AND MAX(position) = $4 - 1
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreatePlaylistImportStatement: %w", err)
	}

	c.CreatePlaylistImportStatement = stmt

	return nil
}

// CreatePlaylistImport makes a fully staged import visible to the app-event worker.
func (c *Client) CreatePlaylistImport(ctx context.Context, importID string, roomID string, userID string, count int) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreatePlaylistImport")
	defer span.End()

	if count <= 0 {
		return fmt.Errorf("error creating playlist import: playlist has no songs")
	}

	cctx, cancel := context.WithTimeout(ctx, playlistImportDatabaseTimeout)
	defer cancel()

	result, err := c.CreatePlaylistImportStatement.ExecContext(cctx, importID, roomID, userID, count)
	if err != nil {
		return fmt.Errorf("error creating playlist import: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error reading created playlist import count: %w", err)
	}

	if affected != 1 {
		return fmt.Errorf("error creating playlist import: staged items are incomplete")
	}

	return nil
}

func (c *Client) prepareDeleteAbandonedPlaylistImportItemsStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM playlist_import_items a
		WHERE a.created_at < NOW() - INTERVAL '1 day'
		AND NOT EXISTS (SELECT 1 FROM playlist_imports b WHERE b.id = a.import_id)
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeleteAbandonedPlaylistImportItemsStatement: %w", err)
	}

	c.DeleteAbandonedPlaylistImportItemsStatement = stmt

	return nil
}

func (c *Client) DeleteAbandonedPlaylistImportItems(ctx context.Context) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "DeleteAbandonedPlaylistImportItems")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.DeleteAbandonedPlaylistImportItemsStatement.ExecContext(cctx)
	if err != nil {
		return fmt.Errorf("error deleting abandoned playlist import items: %w", err)
	}

	return nil
}

func (c *Client) prepareProcessNextPlaylistImportStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH claimed_import_q AS (
			SELECT a.id
			FROM playlist_imports a
			WHERE a.updated_at <= NOW()
			AND EXISTS (
				SELECT 1
				FROM playlist_import_items b
				WHERE b.import_id = a.id
				AND b.position = a.next_position
			)
			ORDER BY a.updated_at ASC, a.created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		),
		updated_import_q AS (
			UPDATE playlist_imports a
			SET
				attempts = a.attempts + 1,
				updated_at = NOW() + ($2 * INTERVAL '1 millisecond')
			FROM claimed_import_q b
			WHERE a.id = b.id
			AND a.attempts < $1
			RETURNING
				a.id,
				a.room_id,
				a.added_by,
				a.next_position,
				a.attempts,
				FALSE AS exhausted
		),
		exhausted_import_q AS (
			SELECT
				a.id,
				a.room_id,
				a.added_by,
				a.next_position,
				a.attempts,
				TRUE AS exhausted
			FROM playlist_imports a
			JOIN claimed_import_q b ON b.id = a.id
			WHERE a.attempts >= $1
		),
		processed_import_q AS (
			SELECT * FROM updated_import_q
			UNION ALL
			SELECT * FROM exhausted_import_q
		)
		SELECT
			a.id,
			a.room_id,
			a.added_by,
			a.next_position,
			a.attempts,
			a.exhausted,
			b.id,
			b.source_type,
			b.source_id,
			b.provider_url,
			b.playback_restriction,
			b.title,
			b.artist,
			b.thumbnail_url,
			b.duration,
			b.created_at
		FROM processed_import_q a
		JOIN playlist_import_items b
		ON b.import_id = a.id
		AND b.position = a.next_position
	`)
	if err != nil {
		return fmt.Errorf("error preparing ProcessNextPlaylistImportStatement: %w", err)
	}

	c.ProcessNextPlaylistImportStatement = stmt

	return nil
}

func (c *Client) ProcessNextPlaylistImport(
	ctx context.Context,
	retryAfter time.Duration,
) (*vibe.PlaylistImport, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ProcessNextPlaylistImport")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, playlistImportDatabaseTimeout)
	defer cancel()

	row := c.ProcessNextPlaylistImportStatement.QueryRowContext(
		cctx,
		playlistImportMaxAttempts,
		retryAfter.Milliseconds(),
	)

	var playlistImportRow playlistImportRow
	err := playlistImportRow.scan(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf("error processing playlist import: no imports ready"),
				},
			}
		}

		return nil, fmt.Errorf("error scanning playlist import in ProcessNextPlaylistImport: %w", err)
	}

	playlistImport, err := playlistImportRow.toPlaylistImport()
	if err != nil {
		return nil, fmt.Errorf("error converting playlist import in ProcessNextPlaylistImport: %w", err)
	}

	return playlistImport, nil
}

type playlistImportRow struct {
	ID                  sql.NullString
	RoomID              sql.NullString
	AddedBy             sql.NullString
	NextPosition        sql.NullInt64
	Attempts            sql.NullInt64
	Exhausted           sql.NullBool
	SongID              sql.NullString
	SourceType          sql.NullString
	SourceID            sql.NullString
	ProviderURL         sql.NullString
	PlaybackRestriction sql.NullString
	Title               sql.NullString
	Artist              sql.NullString
	ThumbnailURL        sql.NullString
	Duration            sql.NullInt64
	AddedAt             sql.NullTime
}

func (r *playlistImportRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.ID,
		&r.RoomID,
		&r.AddedBy,
		&r.NextPosition,
		&r.Attempts,
		&r.Exhausted,
		&r.SongID,
		&r.SourceType,
		&r.SourceID,
		&r.ProviderURL,
		&r.PlaybackRestriction,
		&r.Title,
		&r.Artist,
		&r.ThumbnailURL,
		&r.Duration,
		&r.AddedAt,
	)
	if err != nil {
		return fmt.Errorf("error scanning playlist import row in scan: %w", err)
	}

	return nil
}

func (r *playlistImportRow) toPlaylistImport() (*vibe.PlaylistImport, error) {
	return &vibe.PlaylistImport{
		ID:           r.ID.String,
		RoomID:       r.RoomID.String,
		AddedBy:      r.AddedBy.String,
		NextPosition: int(r.NextPosition.Int64),
		Attempts:     int(r.Attempts.Int64),
		Exhausted:    r.Exhausted.Bool,
		Song: vibe.Song{
			ID:                  r.SongID.String,
			RoomID:              r.RoomID.String,
			SourceType:          r.SourceType.String,
			SourceID:            r.SourceID.String,
			ProviderURL:         r.ProviderURL.String,
			PlaybackRestriction: r.PlaybackRestriction.String,
			Title:               r.Title.String,
			Artist:              r.Artist.String,
			ThumbnailURL:        r.ThumbnailURL.String,
			Duration:            int(r.Duration.Int64),
			AddedBySessionID:    r.AddedBy.String,
			AddedAt:             r.AddedAt.Time,
		},
	}, nil
}

func (c *Client) prepareCompletePlaylistImportItemStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH deleted_item_q AS (
			DELETE FROM playlist_import_items
			WHERE import_id = $1
			AND position = $2
			RETURNING import_id
		),
		remaining_item_q AS MATERIALIZED (
			SELECT 1
			FROM playlist_import_items
			WHERE import_id = $1
			AND position != $2
			LIMIT 1
		),
		updated_import_q AS (
			UPDATE playlist_imports a
			SET
				next_position = a.next_position + 1,
				attempts = 0,
				updated_at = NOW()
			WHERE a.id = $1
			AND EXISTS (
				SELECT 1
				FROM deleted_item_q b
				WHERE b.import_id = a.id
			)
			AND EXISTS (SELECT 1 FROM remaining_item_q)
			RETURNING a.id
		),
		deleted_import_q AS (
			DELETE FROM playlist_imports a
			WHERE a.id = $1
			AND EXISTS (
				SELECT 1
				FROM deleted_item_q b
				WHERE b.import_id = a.id
			)
			AND NOT EXISTS (SELECT 1 FROM remaining_item_q)
			RETURNING a.id
		)
		SELECT
			(SELECT COUNT(*) FROM updated_import_q) +
			(SELECT COUNT(*) FROM deleted_import_q)
	`)
	if err != nil {
		return fmt.Errorf("error preparing CompletePlaylistImportItemStatement: %w", err)
	}

	c.CompletePlaylistImportItemStatement = stmt

	return nil
}

func (c *Client) CompletePlaylistImportItem(
	ctx context.Context,
	importID string,
	position int,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CompletePlaylistImportItem")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, playlistImportDatabaseTimeout)
	defer cancel()

	r := c.CompletePlaylistImportItemStatement.QueryRowContext(
		cctx,
		importID,
		position,
	)

	var row completedPlaylistImportRow
	err := row.scan(r)
	if err != nil {
		return fmt.Errorf("error completing playlist import item in CompletePlaylistImportItem: %w", err)
	}

	if row.UpdatedCount != 1 {
		return fmt.Errorf("error completing playlist import item: import item was not claimed")
	}

	return nil
}

type completedPlaylistImportRow struct {
	UpdatedCount int
}

func (r *completedPlaylistImportRow) scan(row *sql.Row) error {
	err := row.Scan(&r.UpdatedCount)
	if err != nil {
		return fmt.Errorf("error scanning completed playlist import row: %w", err)
	}

	return nil
}

func (c *Client) prepareDeletePlaylistImportStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH deleted_items_q AS (
			DELETE FROM playlist_import_items
			WHERE import_id = $1
		)
		DELETE FROM playlist_imports
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeletePlaylistImportStatement: %w", err)
	}

	c.DeletePlaylistImportStatement = stmt

	return nil
}

func (c *Client) DeletePlaylistImport(ctx context.Context, importID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "DeletePlaylistImport")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, playlistImportDatabaseTimeout)
	defer cancel()

	_, err := c.DeletePlaylistImportStatement.ExecContext(cctx, importID)
	if err != nil {
		return fmt.Errorf("error deleting playlist import in DeletePlaylistImport: %w", err)
	}

	return nil
}

// StartPlaylistPlayback returns a state only when the import starts idle playback.
func (c *Client) StartPlaylistPlayback(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "StartPlaylistPlayback")
	defer span.End()

	state, err := c.startPlaybackIfIdle(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error starting playlist playback: %w", err)
	}

	return state, nil
}

const playlistImportMaxAttempts = 5

// Imports can contain large batches, so retain their longer database deadline.
const playlistImportDatabaseTimeout = 15 * time.Second
