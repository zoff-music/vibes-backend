package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareCreatePlaylistImportItemStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO playlist_import_items (
			id, import_id, position, source_type, source_id, provider_url,
			playback_restriction, title, publisher, thumbnail_url, duration
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreatePlaylistImportItemStatement: %w", err)
	}

	c.CreatePlaylistImportItemStatement = stmt

	return nil
}

func (c *Client) CreatePlaylistImportItem(ctx context.Context, importID string, position int, playlistItem vibe.PlaylistItem) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreatePlaylistImportItem")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, playlistImportDatabaseTimeout)
	defer cancel()

	_, err := c.CreatePlaylistImportItemStatement.ExecContext(
		cctx, playlistItem.ID, importID, position, playlistItem.SourceType, playlistItem.SourceID,
		playlistItem.ProviderURL, playlistItem.PlaybackRestriction, playlistItem.Title, playlistItem.Publisher,
		playlistItem.ThumbnailURL, playlistItem.Duration,
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
		return fmt.Errorf("error creating playlist import: playlist has no items")
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
		WITH expired_imports_q AS (
			DELETE FROM playlist_imports
			WHERE created_at < NOW() - INTERVAL '1 day'
			RETURNING id
		)
		DELETE FROM playlist_import_items a
		WHERE a.created_at < NOW() - INTERVAL '1 day'
		OR a.import_id IN (SELECT id FROM expired_imports_q)
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
			b.publisher,
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
	PlaylistItemID      sql.NullString
	SourceType          sql.NullString
	SourceID            sql.NullString
	ProviderURL         sql.NullString
	PlaybackRestriction sql.NullString
	Title               sql.NullString
	Publisher           sql.NullString
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
		&r.PlaylistItemID,
		&r.SourceType,
		&r.SourceID,
		&r.ProviderURL,
		&r.PlaybackRestriction,
		&r.Title,
		&r.Publisher,
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
		PlaylistItem: vibe.PlaylistItem{
			ID:                  r.PlaylistItemID.String,
			RoomID:              r.RoomID.String,
			SourceType:          r.SourceType.String,
			SourceID:            r.SourceID.String,
			ProviderURL:         r.ProviderURL.String,
			PlaybackRestriction: r.PlaybackRestriction.String,
			Title:               r.Title.String,
			Publisher:           r.Publisher.String,
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
func (c *Client) StartPlaylistPlayback(ctx context.Context, roomID string) (*vibe.PlaybackStateV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "StartPlaylistPlayback")
	defer span.End()

	state, err := c.startPlaybackIfIdleV2(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error starting playlist playback: %w", err)
	}

	return state, nil
}

const playlistImportMaxAttempts = 5

// Imports can contain large batches, so retain their longer database deadline.
const playlistImportDatabaseTimeout = 15 * time.Second

func (c *Client) prepareUpdatePlaylistItemPlaybackRestrictionStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE playlist_items
		SET playback_restriction = $3
		WHERE room_id = $1
		AND id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdatePlaylistItemPlaybackRestrictionStatement: %w", err)
	}

	c.UpdatePlaylistItemPlaybackRestrictionStatement = stmt

	return nil
}

func (c *Client) UpdatePlaylistItemPlaybackRestriction(
	ctx context.Context,
	roomID string,
	playlistItemID string,
	restriction string,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdatePlaylistItemPlaybackRestriction")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.UpdatePlaylistItemPlaybackRestrictionStatement.ExecContext(
		cctx,
		roomID,
		playlistItemID,
		restriction,
	)
	if err != nil {
		return fmt.Errorf("error updating playlist item playback restriction: %w", err)
	}

	return nil
}

// Expire a room at a time, independently of provider availability. The 25-day
// limit leaves headroom for search caches, import staging, and event replay.
func (c *Client) prepareExpirePlaylistItemMetadataStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH room_q AS (
			SELECT a.id
			FROM rooms a
			WHERE EXISTS (
				SELECT 1 FROM playlist_items b
				WHERE b.room_id = a.id
				AND b.source_type = 'youtube'
				AND b.metadata_updated_at <= NOW() - INTERVAL '25 days'
			)
			ORDER BY a.id
			LIMIT 1
			FOR UPDATE OF a SKIP LOCKED
		), expired_q AS MATERIALIZED (
			SELECT a.id, a.room_id
			FROM playlist_items a
			JOIN room_q b ON b.id = a.room_id
			WHERE a.source_type = 'youtube'
			AND a.metadata_updated_at <= NOW() - INTERVAL '25 days'
			FOR UPDATE OF a
		), deleted_votes_q AS (
			DELETE FROM playlist_item_votes a USING expired_q b
			WHERE a.room_id = b.room_id AND a.playlist_item_id = b.id
			RETURNING a.room_id
		), deleted_skips_q AS (
			DELETE FROM skip_votes a USING expired_q b
			WHERE a.room_id = b.room_id AND a.playlist_item_id = b.id
			RETURNING a.room_id
		), stopped_playback_q AS (
			UPDATE playback_state a
			SET current_playlist_item_id = NULL, is_playing = FALSE,
				position_ms = 0, updated_at = NOW()
			FROM expired_q b
			WHERE a.room_id = b.room_id AND a.current_playlist_item_id = b.id
			RETURNING a.room_id
		), deleted_playlist_items_q AS (
			DELETE FROM playlist_items a USING expired_q b WHERE a.id = b.id
			RETURNING a.room_id
		)
		SELECT room_id FROM deleted_votes_q
		UNION
		SELECT room_id FROM deleted_skips_q
		UNION
		SELECT room_id FROM stopped_playback_q
		UNION
		SELECT room_id FROM deleted_playlist_items_q
	`)
	if err != nil {
		return fmt.Errorf("error preparing ExpirePlaylistItemMetadataStatement: %w", err)
	}

	c.ExpirePlaylistItemMetadataStatement = stmt

	return nil
}

func (c *Client) ExpirePlaylistItemMetadata(ctx context.Context) (*vibe.PlaylistItemMetadataExpiry, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ExpirePlaylistItemMetadata")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.ExpirePlaylistItemMetadataStatement.QueryRowContext(cctx)

	var expiryRow playlistItemMetadataExpiryRow
	err := expiryRow.scan(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrExpected{Err: internalerror.ErrNonRecoverable{
				Err: fmt.Errorf("error expiring playlist item metadata in ExpirePlaylistItemMetadata: none expired"),
			}}
		}

		return nil, fmt.Errorf("error scanning expired metadata in ExpirePlaylistItemMetadata: %w", err)
	}

	expiry := expiryRow.toPlaylistItemMetadataExpiry()

	return expiry, nil
}

type playlistItemMetadataExpiryRow struct {
	RoomID string
}

func (r *playlistItemMetadataExpiryRow) scan(row *sql.Row) error {
	err := row.Scan(&r.RoomID)
	if err != nil {
		return fmt.Errorf("error scanning metadata expiry room: %w", err)
	}

	return nil
}

func (r *playlistItemMetadataExpiryRow) toPlaylistItemMetadataExpiry() *vibe.PlaylistItemMetadataExpiry {
	return &vibe.PlaylistItemMetadataExpiry{RoomID: r.RoomID}
}

func (c *Client) prepareClaimPlaylistItemMetadataRefreshStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH stale_playlist_item_q AS (
			SELECT a.id
			FROM playlist_items a
			WHERE a.source_type = $1
			AND a.metadata_refresh_after <= NOW()
			ORDER BY a.metadata_refresh_after ASC, a.id ASC
			LIMIT 1
			FOR UPDATE OF a SKIP LOCKED
		)
		UPDATE playlist_items a
		SET metadata_refresh_after = NOW() + make_interval(secs => $2)
		FROM stale_playlist_item_q b
		WHERE a.id = b.id
		RETURNING a.id, a.room_id, a.source_id
	`)
	if err != nil {
		return fmt.Errorf("error preparing ClaimPlaylistItemMetadataRefreshStatement: %w", err)
	}

	c.ClaimPlaylistItemMetadataRefreshStatement = stmt

	return nil
}

func (c *Client) ClaimPlaylistItemMetadataRefresh(
	ctx context.Context,
	provider string,
	retryAfter time.Duration,
) (*vibe.PlaylistItemMetadataRefresh, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ClaimPlaylistItemMetadataRefresh")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.ClaimPlaylistItemMetadataRefreshStatement.QueryRowContext(
		cctx,
		string(provider),
		int(retryAfter/time.Second),
	)

	var refreshRow playlistItemMetadataRefreshRow
	err := refreshRow.scan(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf(
						"error claiming playlist item metadata refresh in ClaimPlaylistItemMetadataRefresh: none ready for processing",
					),
				},
			}
		}

		return nil, fmt.Errorf(
			"error scanning playlist item metadata refresh in ClaimPlaylistItemMetadataRefresh: %w",
			err,
		)
	}

	refresh, err := refreshRow.toPlaylistItemMetadataRefresh()
	if err != nil {
		return nil, fmt.Errorf("error mapping refresh: %w", err)
	}

	return refresh, nil
}

type playlistItemMetadataRefreshRow struct {
	PlaylistItemID sql.NullString
	RoomID         sql.NullString
	SourceID       sql.NullString
}

func (r *playlistItemMetadataRefreshRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.PlaylistItemID,
		&r.RoomID,
		&r.SourceID,
	)
	if err != nil {
		return fmt.Errorf("error scanning playlist item metadata refresh in scan: %w", err)
	}

	return nil
}

func (r *playlistItemMetadataRefreshRow) toPlaylistItemMetadataRefresh() (*vibe.PlaylistItemMetadataRefresh, error) {
	return &vibe.PlaylistItemMetadataRefresh{
		PlaylistItemID: r.PlaylistItemID.String,
		RoomID:         r.RoomID.String,
		SourceID:       r.SourceID.String,
	}, nil
}

func (c *Client) prepareRefreshPlaylistItemMetadataStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE playlist_items
		SET title = $2,
			publisher = $3,
			thumbnail_url = $4,
			duration = $5,
			provider_url = $6,
			playback_restriction = $7,
			metadata_updated_at = NOW(),
			metadata_refresh_after = NOW() + make_interval(secs => $8)
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing RefreshPlaylistItemMetadataStatement: %w", err)
	}

	c.RefreshPlaylistItemMetadataStatement = stmt

	return nil
}

func (c *Client) RefreshPlaylistItemMetadata(
	ctx context.Context,
	refresh vibe.PlaylistItemMetadataRefresh,
	item vibe.ProviderItem,
	refreshInterval time.Duration,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "RefreshPlaylistItemMetadata")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.RefreshPlaylistItemMetadataStatement.ExecContext(
		cctx,
		refresh.PlaylistItemID,
		item.Title,
		item.Publisher,
		item.ThumbnailURL,
		item.DurationSeconds,
		item.ProviderURL,
		item.PlaybackRestriction,
		int(refreshInterval/time.Second),
	)
	if err != nil {
		return fmt.Errorf("error refreshing playlist item metadata in RefreshPlaylistItemMetadata: %w", err)
	}

	return nil
}

func (c *Client) prepareDeferPlaylistItemMetadataRefreshStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE playlist_items
		SET metadata_refresh_after = NOW() + make_interval(secs => $2)
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeferPlaylistItemMetadataRefreshStatement: %w", err)
	}

	c.DeferPlaylistItemMetadataRefreshStatement = stmt

	return nil
}

func (c *Client) DeferPlaylistItemMetadataRefresh(
	ctx context.Context,
	playlistItemID string,
	retryAfter time.Duration,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "DeferPlaylistItemMetadataRefresh")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.DeferPlaylistItemMetadataRefreshStatement.ExecContext(
		cctx,
		playlistItemID,
		int(retryAfter/time.Second),
	)
	if err != nil {
		return fmt.Errorf(
			"error deferring playlist item metadata refresh in DeferPlaylistItemMetadataRefresh: %w",
			err,
		)
	}

	return nil
}

// prepareGetPlaylistItemsStmt prepares the GetPlaylistItemsStatement.
func (c *Client) prepareGetPlaylistItemsStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			a.id,
			a.room_id,
			a.source_type,
			a.source_id,
			a.provider_url,
			a.playback_restriction,
			a.title,
			a.publisher,
			a.thumbnail_url,
			a.duration,
			a.added_by,
			COALESCE(
				(SELECT c.name FROM sessions c WHERE c.id = a.added_by),
				a.added_by_nickname
			) AS added_by_nickname,
			a.added_at,
			COUNT(b.user_id) as vote_count
		FROM playlist_items a
		LEFT JOIN playlist_item_votes b
		ON a.id = b.playlist_item_id
		AND a.room_id = b.room_id
		WHERE a.room_id = $1
		AND a.source_type = ANY($2::text[])
		AND NOT (a.source_type = 'youtube' AND a.duration <= 0)
		GROUP BY a.id, a.room_id, a.source_type, a.source_id, a.provider_url, a.playback_restriction, a.title, a.publisher, a.thumbnail_url, a.duration, a.added_by, a.added_by_nickname, a.added_at
		ORDER BY vote_count DESC, MAX(b.created_at) ASC, a.added_at ASC
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetPlaylistItemsStatement: %w", err)
	}

	c.GetPlaylistItemsStatement = stmt

	return nil
}

// GetPlaylistItems fetches all playlistItems in a room's queue.
func (c *Client) GetPlaylistItems(ctx context.Context, roomID string) ([]vibe.PlaylistItem, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetPlaylistItems")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.GetPlaylistItemsStatement.QueryContext(
		cctx,
		roomID,
		c.enabledProviders,
	)
	if err != nil {
		return nil, fmt.Errorf("error fetching playlistItems: %w", err)
	}
	defer rows.Close()

	playlistItems := []vibe.PlaylistItem{}
	for rows.Next() {
		var row playlistItemRow

		err := row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning playlistItem row: %w", err)
		}

		playlistItem, err := row.toPlaylistItem()
		if err != nil {
			return nil, fmt.Errorf("error mapping playlistItem: %w", err)
		}

		playlistItems = append(playlistItems, *playlistItem)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating playlistItem rows: %w", err)
	}

	return playlistItems, nil
}

type playlistItemRow struct {
	ID                  sql.NullString
	RoomID              sql.NullString
	SourceType          sql.NullString
	SourceID            sql.NullString
	ProviderURL         sql.NullString
	PlaybackRestriction sql.NullString
	Title               sql.NullString
	Publisher           sql.NullString
	ThumbnailURL        sql.NullString
	Duration            sql.NullInt64
	AddedBySessionID    sql.NullString
	AddedBy             sql.NullString
	AddedAt             sql.NullTime
	VoteCount           sql.NullInt64
}

func (r *playlistItemRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(
		&r.ID,
		&r.RoomID,
		&r.SourceType,
		&r.SourceID,
		&r.ProviderURL,
		&r.PlaybackRestriction,
		&r.Title,
		&r.Publisher,
		&r.ThumbnailURL,
		&r.Duration,
		&r.AddedBySessionID,
		&r.AddedBy,
		&r.AddedAt,
		&r.VoteCount,
	)
	if err != nil {
		return fmt.Errorf("error scanning playlistItem row: %w", err)
	}

	return nil
}

func (r *playlistItemRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.ID,
		&r.RoomID,
		&r.SourceType,
		&r.SourceID,
		&r.ProviderURL,
		&r.PlaybackRestriction,
		&r.Title,
		&r.Publisher,
		&r.ThumbnailURL,
		&r.Duration,
		&r.AddedBySessionID,
		&r.AddedBy,
		&r.AddedAt,
		&r.VoteCount,
	)
	if err != nil {
		return fmt.Errorf("error scanning playlistItem row: %w", err)
	}

	return nil
}

func (r *playlistItemRow) toPlaylistItem() (*vibe.PlaylistItem, error) {
	return &vibe.PlaylistItem{
		ID:                  r.ID.String,
		RoomID:              r.RoomID.String,
		SourceType:          r.SourceType.String,
		SourceID:            r.SourceID.String,
		ProviderURL:         r.ProviderURL.String,
		PlaybackRestriction: r.PlaybackRestriction.String,
		Title:               r.Title.String,
		Publisher:           r.Publisher.String,
		ThumbnailURL:        r.ThumbnailURL.String,
		Duration:            int(r.Duration.Int64),
		AddedBySessionID:    r.AddedBySessionID.String,
		AddedBy:             r.AddedBy.String,
		AddedAt:             r.AddedAt.Time,
		VoteCount:           int(r.VoteCount.Int64),
	}, nil
}

// prepareGetPlaylistItemStmt prepares the GetPlaylistItemStatement.
func (c *Client) prepareGetPlaylistItemStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			a.id,
			a.room_id,
			a.source_type,
			a.source_id,
			a.provider_url,
			a.playback_restriction,
			a.title,
			a.publisher,
			a.thumbnail_url,
			a.duration,
			a.added_by,
			COALESCE(
				(SELECT c.name FROM sessions c WHERE c.id = a.added_by),
				a.added_by_nickname
			) AS added_by_nickname,
			a.added_at,
			COUNT(b.user_id) as vote_count
		FROM playlist_items a
		LEFT JOIN playlist_item_votes b
		ON a.id = b.playlist_item_id
		AND a.room_id = b.room_id
		WHERE a.room_id = $1
		AND a.id = $2
		AND a.source_type = ANY($3::text[])
		AND NOT (a.source_type = 'youtube' AND a.duration <= 0)
		GROUP BY a.id, a.room_id, a.source_type, a.source_id, a.provider_url, a.playback_restriction, a.title, a.publisher, a.thumbnail_url, a.duration, a.added_by, a.added_by_nickname, a.added_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetPlaylistItemStatement: %w", err)
	}

	c.GetPlaylistItemStatement = stmt

	return nil
}

// GetPlaylistItem fetches a single playlistItem by ID.
func (c *Client) GetPlaylistItem(ctx context.Context, roomID, playlistItemID string) (*vibe.PlaylistItem, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetPlaylistItem")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetPlaylistItemStatement.QueryRowContext(
		cctx,
		roomID,
		playlistItemID,
		c.enabledProviders,
	)

	var row playlistItemRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.PlaylistItem{}, nil
		}

		return nil, fmt.Errorf("error fetching playlistItem: %w", err)
	}

	playlistItem, err := row.toPlaylistItem()
	if err != nil {
		return nil, fmt.Errorf("error mapping playlistItem: %w", err)
	}

	return playlistItem, nil
}

func (c *Client) prepareAddPlaylistItemStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH room_config_q AS (
			SELECT
				a.id AS room_id,
				COALESCE(b.allow_duplicates, FALSE) AS allow_duplicates
			FROM rooms a
			LEFT JOIN room_settings b ON b.room_id = a.id
			WHERE a.id = $1
			AND $2 = ANY($12::text[])
			FOR KEY SHARE OF a
		),
		existing_session_q AS MATERIALIZED (
			SELECT name
			FROM sessions
			WHERE id = $8
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
					hashtextextended($8, 0) & 9223372036854775807,
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
				$8,
				SPLIT_PART(a.name, '-', 1) || '-' || SPLIT_PART(a.name, '-', 2)
			FROM generated_name_q a
			WHERE a.name IS NOT NULL
			ON CONFLICT (id) DO UPDATE
			SET name = sessions.name
			RETURNING name
		),
		session_profile_q AS (
			SELECT name
			FROM existing_session_q
			UNION ALL
			SELECT name
			FROM inserted_session_q
			LIMIT 1
		),
		upserted_playlist_item_q AS (
			INSERT INTO playlist_items (
				id,
				room_id,
				source_type,
				source_id,
				provider_url,
				playback_restriction,
				title,
				publisher,
				thumbnail_url,
				duration,
				added_by,
				added_by_nickname,
				added_at,
				duplicate_guard
			)
			SELECT
				$10,
				a.room_id,
				$2,
				$3,
				$13,
				$14,
				$4,
				$5,
				$6,
				$7,
				$8,
				COALESCE((SELECT name FROM session_profile_q), $9),
				NOW(),
				NOT a.allow_duplicates
			FROM room_config_q a
			ON CONFLICT (room_id, source_type, source_id)
			WHERE duplicate_guard
			DO UPDATE SET source_id = EXCLUDED.source_id
			RETURNING
				id,
				room_id,
				source_type,
				source_id,
				provider_url,
				playback_restriction,
				title,
				publisher,
				thumbnail_url,
				duration,
				added_by,
				added_by_nickname,
				added_at,
				id = $10 AS inserted
		),
		inserted_vote_q AS (
			INSERT INTO playlist_item_votes (room_id, playlist_item_id, user_id)
			SELECT a.room_id, a.id, $8
			FROM upserted_playlist_item_q a
			WHERE $11
			ON CONFLICT (room_id, playlist_item_id, user_id) DO NOTHING
			RETURNING 1
		)
		SELECT
			CASE
				WHEN a.inserted THEN 'added'
				WHEN EXISTS (SELECT 1 FROM inserted_vote_q) THEN 'duplicate_voted'
				ELSE 'duplicate_already_voted'
			END AS result,
			a.id,
			a.room_id,
			a.source_type,
			a.source_id,
			a.provider_url,
			a.playback_restriction,
			a.title,
			a.publisher,
			a.thumbnail_url,
			a.duration,
			a.added_by,
			COALESCE(
				(SELECT b.name FROM sessions b WHERE b.id = a.added_by),
				a.added_by_nickname
			) AS added_by_nickname,
			a.added_at,
			(
				SELECT COUNT(*)
				FROM playlist_item_votes b
				WHERE b.room_id = a.room_id
				AND b.playlist_item_id = a.id
			) + (SELECT COUNT(*) FROM inserted_vote_q) AS vote_count
		FROM upserted_playlist_item_q a
		UNION ALL
		SELECT
			CASE
				WHEN $2 = ANY($12::text[]) THEN 'room_not_found'
				ELSE 'provider_disabled'
			END AS result,
			NULL::TEXT AS id,
			NULL::TEXT AS room_id,
			NULL::TEXT AS source_type,
			NULL::TEXT AS source_id,
			NULL::TEXT AS provider_url,
			NULL::TEXT AS playback_restriction,
			NULL::TEXT AS title,
			NULL::TEXT AS publisher,
			NULL::TEXT AS thumbnail_url,
			NULL::INTEGER AS duration,
			NULL::TEXT AS added_by,
			NULL::TEXT AS added_by_nickname,
			NULL::TIMESTAMP AS added_at,
			0::BIGINT AS vote_count
		WHERE NOT EXISTS (SELECT 1 FROM upserted_playlist_item_q)
	`)
	if err != nil {
		return fmt.Errorf("error preparing AddPlaylistItemStatement: %w", err)
	}

	c.AddPlaylistItemStatement = stmt

	return nil
}

// AddPlaylistItem adds a playlistItem to the queue.
func (c *Client) AddPlaylistItem(ctx context.Context, playlistItem *vibe.PlaylistItem) (*vibe.AddPlaylistItemResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AddPlaylistItem")
	defer span.End()

	result, err := c.addPlaylistItem(ctx, playlistItem, true)
	if err != nil {
		return nil, fmt.Errorf("error adding playlistItem in AddPlaylistItem: %w", err)
	}

	return result, nil
}

func (c *Client) AddImportedPlaylistItem(
	ctx context.Context,
	playlistItem *vibe.PlaylistItem,
) (*vibe.AddPlaylistItemResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AddImportedPlaylistItem")
	defer span.End()

	result, err := c.addPlaylistItem(ctx, playlistItem, false)
	if err != nil {
		return nil, fmt.Errorf("error adding imported playlist item: %w", err)
	}

	return result, nil
}

func (c *Client) addPlaylistItem(
	ctx context.Context,
	playlistItem *vibe.PlaylistItem,
	addVote bool,
) (*vibe.AddPlaylistItemResult, error) {
	if vibe.IsLiveVideo(playlistItem.SourceType, playlistItem.Duration) {
		return nil, internalerror.ErrLiveVideo{
			Err: fmt.Errorf("error adding playlistItem in addPlaylistItem: live videos are not supported"),
		}
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.AddPlaylistItemStatement.QueryRowContext(cctx,
		playlistItem.RoomID,
		playlistItem.SourceType,
		playlistItem.SourceID,
		playlistItem.Title,
		playlistItem.Publisher,
		playlistItem.ThumbnailURL,
		playlistItem.Duration,
		playlistItem.AddedBySessionID,
		playlistItem.AddedBy,
		playlistItem.ID,
		addVote,
		c.enabledProviders,
		playlistItem.ProviderURL,
		playlistItem.PlaybackRestriction,
	)

	var row addPlaylistItemRow
	err := row.scan(r)
	if err != nil {
		return nil, fmt.Errorf("error adding playlistItem atomically in addPlaylistItem: %w", err)
	}

	if row.Result.String == addPlaylistItemResultRoomNotFound {
		return nil, fmt.Errorf("error adding playlistItem in addPlaylistItem: room %s not found", playlistItem.RoomID)
	}
	if row.Result.String == addPlaylistItemResultProviderDisabled {
		return nil, fmt.Errorf("error adding playlistItem in addPlaylistItem: provider %s is disabled", playlistItem.SourceType)
	}

	outcome := row.Result.String
	if outcome != vibe.AddPlaylistItemOutcomeAdded &&
		outcome != vibe.AddPlaylistItemOutcomeDuplicateVoted &&
		outcome != vibe.AddPlaylistItemOutcomeDuplicateAlreadyVoted {
		return nil, fmt.Errorf("error adding playlistItem in addPlaylistItem: unknown outcome %s", outcome)
	}

	convertedPlaylistItem, err := row.toPlaylistItem()
	if err != nil {
		return nil, fmt.Errorf("error mapping added playlistItem: %w", err)
	}

	return &vibe.AddPlaylistItemResult{
		PlaylistItem: *convertedPlaylistItem,
		Outcome:      outcome,
	}, nil
}

type addPlaylistItemRow struct {
	Result              sql.NullString
	ID                  sql.NullString
	RoomID              sql.NullString
	SourceType          sql.NullString
	SourceID            sql.NullString
	ProviderURL         sql.NullString
	PlaybackRestriction sql.NullString
	Title               sql.NullString
	Publisher           sql.NullString
	ThumbnailURL        sql.NullString
	Duration            sql.NullInt64
	AddedBySessionID    sql.NullString
	AddedBy             sql.NullString
	AddedAt             sql.NullTime
	VoteCount           sql.NullInt64
}

func (r *addPlaylistItemRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.Result,
		&r.ID,
		&r.RoomID,
		&r.SourceType,
		&r.SourceID,
		&r.ProviderURL,
		&r.PlaybackRestriction,
		&r.Title,
		&r.Publisher,
		&r.ThumbnailURL,
		&r.Duration,
		&r.AddedBySessionID,
		&r.AddedBy,
		&r.AddedAt,
		&r.VoteCount,
	)
	if err != nil {
		return fmt.Errorf("error scanning added playlistItem row: %w", err)
	}

	return nil
}

func (r *addPlaylistItemRow) toPlaylistItem() (*vibe.PlaylistItem, error) {
	return &vibe.PlaylistItem{
		ID:                  r.ID.String,
		RoomID:              r.RoomID.String,
		SourceType:          r.SourceType.String,
		SourceID:            r.SourceID.String,
		ProviderURL:         r.ProviderURL.String,
		PlaybackRestriction: r.PlaybackRestriction.String,
		Title:               r.Title.String,
		Publisher:           r.Publisher.String,
		ThumbnailURL:        r.ThumbnailURL.String,
		Duration:            int(r.Duration.Int64),
		AddedBySessionID:    r.AddedBySessionID.String,
		AddedBy:             r.AddedBy.String,
		AddedAt:             r.AddedAt.Time,
		VoteCount:           int(r.VoteCount.Int64),
	}, nil
}

// prepareRemovePlaylistItemStmt prepares the RemovePlaylistItemStatement.
func (c *Client) prepareRemovePlaylistItemStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH deleted_skip_votes_q AS (
			DELETE FROM skip_votes
			WHERE room_id = $1
			AND playlist_item_id = $2
		),
		deleted_playlist_item_votes_q AS (
			DELETE FROM playlist_item_votes
			WHERE room_id = $1
			AND playlist_item_id = $2
		),
		stopped_playback_q AS (
			UPDATE playback_state
			SET current_playlist_item_id = NULL, is_playing = FALSE,
				position_ms = 0, updated_at = NOW()
			WHERE room_id = $1 AND current_playlist_item_id = $2
		)
		DELETE FROM playlist_items
		WHERE room_id = $1
		AND id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing RemovePlaylistItemStatement: %w", err)
	}

	c.RemovePlaylistItemStatement = stmt

	return nil
}

// RemovePlaylistItem removes a playlistItem from the queue.
func (c *Client) RemovePlaylistItem(ctx context.Context, roomID, playlistItemID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "RemovePlaylistItem")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.RemovePlaylistItemStatement.ExecContext(cctx, roomID, playlistItemID)
	if err != nil {
		return fmt.Errorf("error removing playlistItem: %w", err)
	}

	return nil
}

// prepareVotePlaylistItemStmt prepares the VotePlaylistItemStatement.
func (c *Client) prepareVotePlaylistItemStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH playlist_item_q AS (
			SELECT a.room_id, a.id
			FROM playlist_items a
			WHERE a.room_id = $1
			AND a.id = $2
			FOR KEY SHARE OF a
		),
		inserted_vote_q AS (
			INSERT INTO playlist_item_votes (room_id, playlist_item_id, user_id)
			SELECT a.room_id, a.id, $3
			FROM playlist_item_q a
			ON CONFLICT(room_id, playlist_item_id, user_id) DO NOTHING
			RETURNING 1
		)
		SELECT CASE
			WHEN NOT EXISTS (SELECT 1 FROM playlist_item_q)
				THEN 'playlistItem_not_found'
			WHEN EXISTS (SELECT 1 FROM inserted_vote_q)
				THEN 'created'
			ELSE 'already_voted'
		END
	`)
	if err != nil {
		return fmt.Errorf("error preparing VotePlaylistItemStatement: %w", err)
	}

	c.VotePlaylistItemStatement = stmt

	return nil
}

// VotePlaylistItem adds a vote for a playlistItem.
func (c *Client) VotePlaylistItem(ctx context.Context, roomID, playlistItemID, userID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "VotePlaylistItem")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.VotePlaylistItemStatement.QueryRowContext(cctx, roomID, playlistItemID, userID)

	var vote votePlaylistItemRow
	err := vote.scan(row)
	if err != nil {
		return fmt.Errorf("error scanning vote playlistItem outcome: %w", err)
	}

	if vote.Outcome.String == votePlaylistItemNotFound {
		return fmt.Errorf("error voting for playlistItem: playlistItem not found")
	}
	if vote.Outcome.String == votePlaylistItemAlreadyVoted {
		return internalerror.ErrAlreadyVoted{
			Err: fmt.Errorf("error voting for playlistItem: vote already exists"),
		}
	}
	if vote.Outcome.String != votePlaylistItemCreated {
		return fmt.Errorf(
			"error voting for playlistItem: unknown outcome %s",
			vote.Outcome.String,
		)
	}

	return nil
}

type votePlaylistItemRow struct {
	Outcome sql.NullString
}

func (r *votePlaylistItemRow) scan(row *sql.Row) error {
	err := row.Scan(&r.Outcome)
	if err != nil {
		return fmt.Errorf("error scanning playlistItem vote row: %w", err)
	}

	return nil
}

func (c *Client) prepareClearVotesPlaylistItemStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM playlist_item_votes WHERE room_id = $1 AND playlist_item_id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing ClearVotesPlaylistItemStatement: %w", err)
	}

	c.ClearVotesPlaylistItemStatement = stmt

	return nil
}

// clearVotesPlaylistItem clears all votes for a playlistItem.
func (c *Client) clearVotesPlaylistItem(ctx context.Context, roomID, playlistItemID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "clearVotesPlaylistItem")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.Printf("[DEBUG-VOTES] Clearing votes for playlistItem %s in room %s", playlistItemID, roomID)

	result, err := c.ClearVotesPlaylistItemStatement.ExecContext(cctx, roomID, playlistItemID)
	if err != nil {
		log.Printf("[DEBUG-VOTES] Error clearing votes for playlistItem %s in room %s: %v", playlistItemID, roomID, err)
		return fmt.Errorf("error clearing votes for playlistItem: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[DEBUG-VOTES] Error getting rows affected for playlistItem %s in room %s: %v", playlistItemID, roomID, err)
		return nil
	}
	log.Printf("[DEBUG-VOTES] Cleared %d votes for playlistItem %s in room %s", rowsAffected, playlistItemID, roomID)

	return nil
}

func (c *Client) prepareUpdatePlaylistItemAddedAtStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE playlist_items
		SET added_at = NOW()
		WHERE room_id = $1 AND id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdatePlaylistItemAddedAtStatement: %w", err)
	}

	c.UpdatePlaylistItemAddedAtStatement = stmt

	return nil
}

// updatePlaylistItemAddedAt updates the added_at timestamp for a playlistItem to treat it as "new"
func (c *Client) updatePlaylistItemAddedAt(ctx context.Context, roomID, playlistItemID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "updatePlaylistItemAddedAt")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.Printf("[DEBUG-VOTES] Updating added_at timestamp for playlistItem %s in room %s", playlistItemID, roomID)

	result, err := c.UpdatePlaylistItemAddedAtStatement.ExecContext(cctx, roomID, playlistItemID)
	if err != nil {
		log.Printf("[DEBUG-VOTES] Error updating added_at for playlistItem %s in room %s: %v", playlistItemID, roomID, err)
		return fmt.Errorf("error updating playlistItem added_at: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[DEBUG-VOTES] Error getting rows affected for playlistItem %s in room %s: %v", playlistItemID, roomID, err)
		return nil
	}
	log.Printf("[DEBUG-VOTES] Updated added_at for %d playlistItem(s) %s in room %s", rowsAffected, playlistItemID, roomID)

	return nil
}

const votePlaylistItemAlreadyVoted = "already_voted"
const votePlaylistItemCreated = "created"
const votePlaylistItemNotFound = "playlistItem_not_found"
const addPlaylistItemResultRoomNotFound = "room_not_found"
const addPlaylistItemResultProviderDisabled = "provider_disabled"
