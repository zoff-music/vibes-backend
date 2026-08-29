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

func (c *Client) prepareCreatePlaylistImportStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH inserted_import_q AS (
			INSERT INTO playlist_imports (id, room_id, added_by)
			VALUES ($1, $2, $3)
			RETURNING id
		)
		INSERT INTO playlist_import_items (
			id,
			import_id,
			position,
			source_type,
			source_id,
			provider_url,
			playback_restriction,
			title,
			artist,
			thumbnail_url,
			duration
		)
		SELECT
			a.id,
			b.id,
			a.position - 1,
			a.source_type,
			a.source_id,
			a.provider_url,
			a.playback_restriction,
			a.title,
			a.artist,
			a.thumbnail_url,
			a.duration
		FROM inserted_import_q b
		CROSS JOIN UNNEST(
			$4::text[],
			$5::text[],
			$6::text[],
			$7::text[],
			$8::text[],
			$9::text[],
			$10::text[],
			$11::text[],
			$12::integer[]
		) WITH ORDINALITY AS a(
			id,
			source_type,
			source_id,
			provider_url,
			playback_restriction,
			title,
			artist,
			thumbnail_url,
			duration,
			position
		)
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreatePlaylistImportStatement: %w", err)
	}

	c.CreatePlaylistImportStatement = stmt

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

func (c *Client) prepareCompletePlaylistImportItemStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH deleted_item_q AS (
			DELETE FROM playlist_import_items
			WHERE import_id = $1
			AND position = $2
			RETURNING import_id
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
			AND EXISTS (
				SELECT 1
				FROM playlist_import_items c
				WHERE c.import_id = a.id
				AND c.position != $2
			)
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
			AND NOT EXISTS (
				SELECT 1
				FROM playlist_import_items c
				WHERE c.import_id = a.id
				AND c.position != $2
			)
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

func (c *Client) CreatePlaylistImport(
	ctx context.Context,
	importID string,
	songs []*vibe.Song,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreatePlaylistImport")
	defer span.End()

	if len(songs) == 0 {
		return fmt.Errorf("error creating playlist import: playlist has no songs")
	}

	ids := make([]string, 0, len(songs))
	sourceTypes := make([]string, 0, len(songs))
	sourceIDs := make([]string, 0, len(songs))
	providerURLs := make([]string, 0, len(songs))
	playbackRestrictions := make([]string, 0, len(songs))
	titles := make([]string, 0, len(songs))
	artists := make([]string, 0, len(songs))
	thumbnailURLs := make([]string, 0, len(songs))
	durations := make([]int, 0, len(songs))
	for _, song := range songs {
		if song == nil {
			return fmt.Errorf("error creating playlist import: playlist contains an empty song")
		}
		if song.RoomID != songs[0].RoomID ||
			song.AddedBySessionID != songs[0].AddedBySessionID {
			return fmt.Errorf("error creating playlist import: songs do not share room and session")
		}

		ids = append(ids, song.ID)
		sourceTypes = append(sourceTypes, song.SourceType)
		sourceIDs = append(sourceIDs, song.SourceID)
		providerURLs = append(providerURLs, song.ProviderURL)
		playbackRestrictions = append(playbackRestrictions, song.PlaybackRestriction)
		titles = append(titles, song.Title)
		artists = append(artists, song.Artist)
		thumbnailURLs = append(thumbnailURLs, song.ThumbnailURL)
		durations = append(durations, song.Duration)
	}

	cctx, cancel := context.WithTimeout(ctx, playlistImportDatabaseTimeout)
	defer cancel()

	_, err := c.CreatePlaylistImportStatement.ExecContext(
		cctx,
		importID,
		songs[0].RoomID,
		songs[0].AddedBySessionID,
		ids,
		sourceTypes,
		sourceIDs,
		providerURLs,
		playbackRestrictions,
		titles,
		artists,
		thumbnailURLs,
		durations,
	)
	if err != nil {
		return fmt.Errorf("error creating playlist import in CreatePlaylistImport: %w", err)
	}

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

	return playlistImportRow.toPlaylistImport(), nil
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

func (r *playlistImportRow) toPlaylistImport() *vibe.PlaylistImport {
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
	}
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

	var updatedCount int
	err := c.CompletePlaylistImportItemStatement.QueryRowContext(
		cctx,
		importID,
		position,
	).Scan(&updatedCount)
	if err != nil {
		return fmt.Errorf("error completing playlist import item in CompletePlaylistImportItem: %w", err)
	}
	if updatedCount != 1 {
		return fmt.Errorf("error completing playlist import item: import item was not claimed")
	}

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

const playlistImportDatabaseTimeout = 15 * time.Second

const playlistImportMaxAttempts = 5
