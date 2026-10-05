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
