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

// prepareGetPlaybackStateV2Stmt prepares the GetPlaybackStateV2Statement.
func (c *Client) prepareGetPlaybackStateV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT room_id, current_playlist_item_id, is_playing, position_ms, updated_at
		FROM playback_state
		WHERE room_id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetPlaybackStateV2Statement: %w", err)
	}

	c.GetPlaybackStateV2Statement = stmt

	return nil
}

// getPlaybackStateV2 fetches the playback state for a room (internal).
func (c *Client) getPlaybackStateV2(ctx context.Context, roomID string) (*vibe.PlaybackStateV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getPlaybackStateV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetPlaybackStateV2Statement.QueryRowContext(cctx, roomID)

	var row playbackStateRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.PlaybackStateV2{
				RoomID:              roomID,
				CurrentPlaylistItem: nil,
				IsPlaying:           false,
				PositionMs:          0,
				UpdatedAt:           time.Now(),
				ServerTimeMs:        int(time.Now().UnixMilli()),
			}, nil
		}

		return nil, fmt.Errorf("error fetching playback state: %w", err)
	}

	state, err := row.toPlaybackStateV2()
	if err != nil {
		return nil, fmt.Errorf("error converting playback state row: %w", err)
	}

	if !row.CurrentPlaylistItemID.Valid || row.CurrentPlaylistItemID.String == "" {
		return state, nil
	}

	playlistItem, err := c.GetPlaylistItem(ctx, state.RoomID, row.CurrentPlaylistItemID.String)
	if err != nil {
		return nil, fmt.Errorf("error get current playlistItem %s: %w", row.CurrentPlaylistItemID.String, err)
	}

	if playlistItem.IsEmpty() {
		return state, nil
	}

	state.CurrentPlaylistItem = playlistItem
	if !state.IsPlaying {
		return state, nil
	}

	elapsed := time.Since(state.UpdatedAt).Milliseconds()
	currentPosition := state.PositionMs + int(elapsed)

	if state.CurrentPlaylistItem.Duration > 0 {
		duration := state.CurrentPlaylistItem.Duration * 1000
		if currentPosition > duration {
			currentPosition = duration
		}
	}
	if currentPosition < 0 {
		currentPosition = 0
	}
	state.PositionMs = currentPosition
	state.UpdatedAt = time.Now()
	state.ServerTimeMs = int(state.UpdatedAt.UnixMilli())

	return state, nil
}

// GetPlaybackStateV2 fetches the playback state for a room.
// It will automatically attempt to start playback if the room is idle.
func (c *Client) GetPlaybackStateV2(ctx context.Context, roomID string) (*vibe.PlaybackStateV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetPlaybackStateV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	state, err := c.getPlaybackStateV2(cctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error get playback state: %w", err)
	}

	if state.CurrentPlaylistItem != nil {
		return state, nil
	}

	newState, err := c.StartPlaybackIfIdleV2(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error auto-start playback: %w", err)
	}

	return newState, nil
}

type playbackStateRow struct {
	RoomID                sql.NullString
	CurrentPlaylistItemID sql.NullString
	IsPlaying             sql.NullBool
	PositionMs            sql.NullInt64
	UpdatedAt             sql.NullTime
}

func (r *playbackStateRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.RoomID,
		&r.CurrentPlaylistItemID,
		&r.IsPlaying,
		&r.PositionMs,
		&r.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error scanning playback state row: %w", err)
	}

	return nil
}

func (r *playbackStateRow) toPlaybackStateV2() (*vibe.PlaybackStateV2, error) {
	return &vibe.PlaybackStateV2{
		RoomID:              r.RoomID.String,
		CurrentPlaylistItem: nil,
		IsPlaying:           r.IsPlaying.Bool,
		PositionMs:          int(r.PositionMs.Int64),
		UpdatedAt:           r.UpdatedAt.Time,
		ServerTimeMs:        int(time.Now().UnixMilli()),
	}, nil
}

type playbackPlaylistItemRow struct {
	PlaybackRoomID                sql.NullString
	PlaybackCurrentPlaylistItemID sql.NullString
	PlaybackIsPlaying             sql.NullBool
	PlaybackPositionMs            sql.NullInt64
	PlaybackUpdatedAt             sql.NullTime
	PlaylistItem                  playlistItemRow
}

type playbackAdvanceRow struct {
	Playback               playbackPlaylistItemRow
	PreviousPlaylistItemID sql.NullString
}

func (r *playbackAdvanceRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.Playback.PlaybackRoomID,
		&r.Playback.PlaybackCurrentPlaylistItemID,
		&r.Playback.PlaybackIsPlaying,
		&r.Playback.PlaybackPositionMs,
		&r.Playback.PlaybackUpdatedAt,
		&r.Playback.PlaylistItem.ID,
		&r.Playback.PlaylistItem.RoomID,
		&r.Playback.PlaylistItem.SourceType,
		&r.Playback.PlaylistItem.SourceID,
		&r.Playback.PlaylistItem.ProviderURL,
		&r.Playback.PlaylistItem.PlaybackRestriction,
		&r.Playback.PlaylistItem.Title,
		&r.Playback.PlaylistItem.Publisher,
		&r.Playback.PlaylistItem.ThumbnailURL,
		&r.Playback.PlaylistItem.Duration,
		&r.Playback.PlaylistItem.AddedBySessionID,
		&r.Playback.PlaylistItem.AddedBy,
		&r.Playback.PlaylistItem.AddedAt,
		&r.Playback.PlaylistItem.VoteCount,
		&r.PreviousPlaylistItemID,
	)
	if err != nil {
		return fmt.Errorf("error scanning playback advance row: %w", err)
	}

	return nil
}

func (r *playbackPlaylistItemRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.PlaybackRoomID,
		&r.PlaybackCurrentPlaylistItemID,
		&r.PlaybackIsPlaying,
		&r.PlaybackPositionMs,
		&r.PlaybackUpdatedAt,
		&r.PlaylistItem.ID,
		&r.PlaylistItem.RoomID,
		&r.PlaylistItem.SourceType,
		&r.PlaylistItem.SourceID,
		&r.PlaylistItem.ProviderURL,
		&r.PlaylistItem.PlaybackRestriction,
		&r.PlaylistItem.Title,
		&r.PlaylistItem.Publisher,
		&r.PlaylistItem.ThumbnailURL,
		&r.PlaylistItem.Duration,
		&r.PlaylistItem.AddedBySessionID,
		&r.PlaylistItem.AddedBy,
		&r.PlaylistItem.AddedAt,
		&r.PlaylistItem.VoteCount,
	)
	if err != nil {
		return fmt.Errorf("error scanning playback playlistItem row: %w", err)
	}

	return nil
}

func (r *playbackPlaylistItemRow) toPlaybackStateV2() (*vibe.PlaybackStateV2, error) {
	state := &vibe.PlaybackStateV2{
		RoomID:              r.PlaybackRoomID.String,
		CurrentPlaylistItem: nil,
		IsPlaying:           r.PlaybackIsPlaying.Bool,
		PositionMs:          int(r.PlaybackPositionMs.Int64),
		UpdatedAt:           r.PlaybackUpdatedAt.Time,
		ServerTimeMs:        int(time.Now().UnixMilli()),
	}

	if !r.PlaylistItem.ID.Valid || r.PlaylistItem.ID.String == "" {
		return state, nil
	}

	playlistItem, err := r.PlaylistItem.toPlaylistItem()
	if err != nil {
		return nil, fmt.Errorf("error mapping playlistItem: %w", err)
	}

	state.CurrentPlaylistItem = playlistItem

	return state, nil
}

// prepareProcessNextExpiredPlaybackV2Stmt prepares the ProcessNextExpiredPlaybackV2Statement.
func (c *Client) prepareProcessNextExpiredPlaybackV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		WITH locked_playback_q AS (
			SELECT
				a.room_id,
				a.current_playlist_item_id,
				COALESCE(d.remove_on_play, FALSE) AS remove_on_play
			FROM playback_state a
			JOIN playlist_items b ON a.current_playlist_item_id = b.id
			JOIN rooms c ON a.room_id = c.id
			JOIN room_settings d ON d.room_id = c.id
			WHERE a.is_playing
			AND (c.mode = 'server' OR c.mode = 'host')
			AND ((EXTRACT(EPOCH FROM (NOW() - a.updated_at)) * 1000) + a.position_ms) >= (b.duration * 1000 - 500)
			ORDER BY (
				SELECT COUNT(*)
				FROM room_users e
				WHERE e.room_id = a.room_id
				AND e.last_seen_at > NOW() - INTERVAL '15 seconds'
				AND (
					COALESCE(e.is_active_listener, FALSE)
					OR COALESCE(e.is_cast_receiver, FALSE)
				)
			) DESC,
			a.updated_at ASC
			LIMIT 1
			FOR UPDATE OF a SKIP LOCKED
		),
		cleared_skip_votes_q AS (
			DELETE FROM skip_votes a
			USING locked_playback_q b
			WHERE a.room_id = b.room_id
			AND a.playlist_item_id = b.current_playlist_item_id
			RETURNING 1
		),
		cleared_playlist_item_votes_q AS (
			DELETE FROM playlist_item_votes a
			USING locked_playback_q b
			WHERE a.room_id = b.room_id
			AND a.playlist_item_id = b.current_playlist_item_id
			RETURNING 1
		),
		removed_playlist_item_q AS (
			DELETE FROM playlist_items a
			USING locked_playback_q b
			WHERE b.remove_on_play
			AND a.room_id = b.room_id
			AND a.id = b.current_playlist_item_id
			RETURNING 1
		),
		requeued_playlist_item_q AS (
			UPDATE playlist_items a
			SET added_at = NOW()
			FROM locked_playback_q b
			WHERE NOT b.remove_on_play
			AND a.room_id = b.room_id
			AND a.id = b.current_playlist_item_id
			RETURNING 1
		),
		next_playlist_item_q AS (
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
					(SELECT d.name FROM sessions d WHERE d.id = a.added_by),
					a.added_by_nickname
				) AS added_by_nickname,
				CASE
					WHEN a.id = c.current_playlist_item_id AND NOT c.remove_on_play THEN NOW()
					ELSE a.added_at
				END AS added_at,
				COUNT(b.user_id) AS vote_count
			FROM playlist_items a
			JOIN locked_playback_q c ON c.room_id = a.room_id
			LEFT JOIN playlist_item_votes b
			ON a.id = b.playlist_item_id
			AND a.room_id = b.room_id
			AND a.id IS DISTINCT FROM c.current_playlist_item_id
			WHERE NOT (c.remove_on_play AND a.id = c.current_playlist_item_id)
			AND a.source_type = ANY($1::text[])
			GROUP BY a.id, a.room_id, a.source_type, a.source_id, a.provider_url, a.playback_restriction, a.title, a.publisher, a.thumbnail_url, a.duration, a.added_by, a.added_by_nickname, a.added_at, c.current_playlist_item_id, c.remove_on_play
			ORDER BY vote_count DESC, MAX(b.created_at) ASC, added_at ASC
			LIMIT 1
		),
		updated_playback_q AS (
			UPDATE playback_state a
			SET current_playlist_item_id = b.id,
			is_playing = b.id IS NOT NULL,
			position_ms = 0,
			updated_at = NOW()
			FROM locked_playback_q c
			LEFT JOIN next_playlist_item_q b ON b.room_id = c.room_id
			WHERE a.room_id = c.room_id
			RETURNING a.room_id, a.current_playlist_item_id, a.is_playing, a.position_ms, a.updated_at
		)
		SELECT
			a.room_id,
			a.current_playlist_item_id,
			a.is_playing,
			a.position_ms,
			a.updated_at,
			b.id,
			b.room_id,
			b.source_type,
			b.source_id,
			b.provider_url,
			b.playback_restriction,
			b.title,
			b.publisher,
			b.thumbnail_url,
			b.duration,
			b.added_by,
			COALESCE(
				(SELECT d.name FROM sessions d WHERE d.id = b.added_by),
				b.added_by_nickname
			) AS added_by_nickname,
			b.added_at,
			COALESCE(b.vote_count, 0) AS vote_count,
			(SELECT current_playlist_item_id FROM locked_playback_q) AS previous_playlist_item_id
		FROM updated_playback_q a
		LEFT JOIN next_playlist_item_q b ON b.id = a.current_playlist_item_id
	`)
	if err != nil {
		return fmt.Errorf("error preparing ProcessNextExpiredPlaybackV2Statement: %w", err)
	}

	c.ProcessNextExpiredPlaybackV2Statement = stmt

	return nil
}

func (c *Client) processNextExpiredPlayback(ctx context.Context) (*vibe.PlaybackAdvanceV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "processNextExpiredPlayback")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.ProcessNextExpiredPlaybackV2Statement.QueryRowContext(
		cctx,
		c.enabledProviders,
	)

	var row playbackAdvanceRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf("error no expired playback found"),
				},
			}
		}

		return nil, fmt.Errorf("error scanning expired playback: %w", err)
	}

	state, err := row.Playback.toPlaybackStateV2()
	if err != nil {
		return nil, fmt.Errorf("error converting expired playback state: %w", err)
	}

	return &vibe.PlaybackAdvanceV2{
		Playback:               state,
		PreviousPlaylistItemID: row.PreviousPlaylistItemID.String,
	}, nil
}

// ProcessNextExpiredPlaybackV2 checks for an expired playlistItem, skips it, and returns the new state.
func (c *Client) ProcessNextExpiredPlaybackV2(ctx context.Context) (*vibe.PlaybackAdvanceV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ProcessNextExpiredPlaybackV2")
	defer span.End()

	advance, err := c.processNextExpiredPlayback(ctx)
	if err != nil {
		return nil, fmt.Errorf("error processing next expired playback: %w", err)
	}

	return advance, nil
}

// prepareUpsertPlaybackStateV2Stmt prepares the UpsertPlaybackStateV2Statement.
func (c *Client) prepareUpsertPlaybackStateV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO playback_state (room_id, current_playlist_item_id, is_playing, position_ms, updated_at)
			SELECT a.id, $2, $3, $4, CURRENT_TIMESTAMP
			FROM rooms a
			WHERE a.id = $1
			FOR KEY SHARE OF a
			ON CONFLICT(room_id) DO UPDATE SET
			current_playlist_item_id = EXCLUDED.current_playlist_item_id,
			is_playing = EXCLUDED.is_playing,
			position_ms = EXCLUDED.position_ms,
			updated_at = EXCLUDED.updated_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpsertPlaybackStateV2Statement: %w", err)
	}

	c.UpsertPlaybackStateV2Statement = stmt

	return nil
}

// UpsertPlaybackStateV2 creates or updates the playback state for a room.
func (c *Client) UpsertPlaybackStateV2(ctx context.Context, state *vibe.PlaybackStateV2) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpsertPlaybackStateV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var currentPlaylistItemID string
	if state.CurrentPlaylistItem != nil {
		currentPlaylistItemID = state.CurrentPlaylistItem.ID
	}

	_, err := c.UpsertPlaybackStateV2Statement.ExecContext(cctx,
		state.RoomID,
		currentPlaylistItemID,
		state.IsPlaying,
		state.PositionMs,
	)
	if err != nil {
		return fmt.Errorf("error upserting playback state: %w", err)
	}

	return nil
}

func (c *Client) prepareSkipPlaylistItemStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH locked_playback_q AS (
			SELECT
				a.room_id,
				a.current_playlist_item_id,
				COALESCE(b.remove_on_play, FALSE) AS remove_on_play
			FROM playback_state a
			JOIN room_settings b ON b.room_id = a.room_id
			WHERE a.room_id = $1
			AND (
				$3 = ''
				OR EXISTS (
					SELECT 1
					FROM playlist_items c
					WHERE c.room_id = a.room_id
					AND c.id = a.current_playlist_item_id
					AND c.id = $3
					AND c.playback_restriction != ''
				)
			)
			FOR UPDATE OF a SKIP LOCKED
		),
		cleared_skip_votes_q AS (
			DELETE FROM skip_votes a
			USING locked_playback_q b
			WHERE a.room_id = b.room_id
			AND a.playlist_item_id = b.current_playlist_item_id
			RETURNING 1
		),
		cleared_playlist_item_votes_q AS (
			DELETE FROM playlist_item_votes a
			USING locked_playback_q b
			WHERE a.room_id = b.room_id
			AND a.playlist_item_id = b.current_playlist_item_id
			RETURNING 1
		),
		removed_playlist_item_q AS (
			DELETE FROM playlist_items a
			USING locked_playback_q b
			WHERE b.current_playlist_item_id IS NOT NULL
			AND b.remove_on_play
			AND a.room_id = b.room_id
			AND a.id = b.current_playlist_item_id
			RETURNING 1
		),
		requeued_playlist_item_q AS (
			UPDATE playlist_items a
			SET added_at = NOW()
			FROM locked_playback_q b
			WHERE b.current_playlist_item_id IS NOT NULL
			AND NOT b.remove_on_play
			AND a.room_id = b.room_id
			AND a.id = b.current_playlist_item_id
			RETURNING 1
		),
		next_playlist_item_q AS (
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
					(SELECT d.name FROM sessions d WHERE d.id = a.added_by),
					a.added_by_nickname
				) AS added_by_nickname,
				CASE
					WHEN a.id = c.current_playlist_item_id AND NOT c.remove_on_play THEN NOW()
					ELSE a.added_at
				END AS added_at,
				COUNT(b.user_id) AS vote_count
			FROM playlist_items a
			JOIN locked_playback_q c ON c.room_id = a.room_id
			LEFT JOIN playlist_item_votes b
			ON a.id = b.playlist_item_id
			AND a.room_id = b.room_id
			AND a.id IS DISTINCT FROM c.current_playlist_item_id
			WHERE NOT (c.remove_on_play AND a.id = c.current_playlist_item_id)
			AND ($3 = '' OR a.id != $3)
			AND a.source_type = ANY($2::text[])
			GROUP BY a.id, a.room_id, a.source_type, a.source_id, a.provider_url, a.playback_restriction, a.title, a.publisher, a.thumbnail_url, a.duration, a.added_by, a.added_by_nickname, a.added_at, c.current_playlist_item_id, c.remove_on_play
			ORDER BY vote_count DESC, MAX(b.created_at) ASC, added_at ASC
			LIMIT 1
		),
		updated_playback_q AS (
			UPDATE playback_state a
			SET current_playlist_item_id = b.id,
			is_playing = b.id IS NOT NULL,
			position_ms = 0,
			updated_at = NOW()
			FROM locked_playback_q c
			LEFT JOIN next_playlist_item_q b ON b.room_id = c.room_id
			WHERE a.room_id = c.room_id
			RETURNING a.room_id, a.current_playlist_item_id, a.is_playing, a.position_ms, a.updated_at
		)
		SELECT
			a.room_id,
			a.current_playlist_item_id,
			a.is_playing,
			a.position_ms,
			a.updated_at,
			b.id,
			b.room_id,
			b.source_type,
			b.source_id,
			b.provider_url,
			b.playback_restriction,
			b.title,
			b.publisher,
			b.thumbnail_url,
			b.duration,
			b.added_by,
			COALESCE(
				(SELECT d.name FROM sessions d WHERE d.id = b.added_by),
				b.added_by_nickname
			) AS added_by_nickname,
			b.added_at,
			COALESCE(b.vote_count, 0) AS vote_count,
			(SELECT current_playlist_item_id FROM locked_playback_q) AS previous_playlist_item_id
		FROM updated_playback_q a
		LEFT JOIN next_playlist_item_q b ON b.id = a.current_playlist_item_id
	`)
	if err != nil {
		return fmt.Errorf("error preparing SkipPlaylistItemStatement: %w", err)
	}

	c.SkipPlaylistItemStatement = stmt

	return nil
}

// skipPlaylistItem skips the current track to the next one in the queue (internal).
func (c *Client) skipPlaylistItem(ctx context.Context, roomID, restrictedPlaylistItemID string) (*vibe.PlaybackAdvanceV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "skipPlaylistItem")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.SkipPlaylistItemStatement.QueryRowContext(
		cctx,
		roomID,
		c.enabledProviders,
		restrictedPlaylistItemID,
	)

	var row playbackAdvanceRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if restrictedPlaylistItemID != "" {
				return &vibe.PlaybackAdvanceV2{Playback: &vibe.PlaybackStateV2{}}, nil
			}

			state, err := c.getPlaybackStateV2(ctx, roomID)
			if err != nil {
				return nil, fmt.Errorf("error getting playback state in skipPlaylistItem: %w", err)
			}
			return &vibe.PlaybackAdvanceV2{Playback: state}, nil
		}
		return nil, fmt.Errorf("error scanning playback state in skipPlaylistItem: %w", err)
	}

	state, err := row.Playback.toPlaybackStateV2()
	if err != nil {
		return nil, fmt.Errorf("error converting playback state in skipPlaylistItem: %w", err)
	}

	return &vibe.PlaybackAdvanceV2{
		Playback:               state,
		PreviousPlaylistItemID: row.PreviousPlaylistItemID.String,
	}, nil
}

func (c *Client) SkipRestrictedPlaylistItem(
	ctx context.Context,
	roomID string,
	playlistItemID string,
) (*vibe.PlaybackAdvanceV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SkipRestrictedPlaylistItem")
	defer span.End()

	advance, err := c.skipPlaylistItem(ctx, roomID, playlistItemID)
	if err != nil {
		return nil, fmt.Errorf("error skipping restricted playlistItem: %w", err)
	}

	return advance, nil
}

// UpdatePlaybackV2 updates the playback state based on the action (play/pause/seek).
func (c *Client) UpdatePlaybackV2(ctx context.Context, roomID string, userID string, action string, positionMs int) (*vibe.PlaybackStateV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdatePlaybackV2")
	defer span.End()

	err := c.checkHostPermissions(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking host permissions in update playback in %s: %w", roomID, err)
	}

	state, err := c.GetPlaybackStateV2(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error getting playback state in UpdatePlaybackV2: %w", err)
	}

	switch action {
	case vibe.RoomActionPlay:
		state.IsPlaying = true
		if state.CurrentPlaylistItem != nil {
			break
		}

		state, err = c.StartPlaybackIfIdleV2(ctx, roomID)
		if err != nil {
			return nil, fmt.Errorf("error starting playback in UpdatePlaybackV2: %w", err)
		}

		if state.CurrentPlaylistItem == nil {
			state.IsPlaying = false
		}

		return state, nil
	case vibe.RoomActionPause:
		if state.IsPlaying {
			elapsed := time.Since(state.UpdatedAt).Milliseconds()
			state.PositionMs = state.PositionMs + int(elapsed)
		}
		state.IsPlaying = false
	case vibe.RoomActionSeek:
		state.PositionMs = positionMs
	default:
		return nil, fmt.Errorf("error invalid action in UpdatePlaybackV2: %s", action)
	}

	state.UpdatedAt = time.Now()

	err = c.UpsertPlaybackStateV2(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("error upserting playback state in UpdatePlaybackV2: %w", err)
	}

	return state, nil
}

func (c *Client) prepareStartPlaybackIfIdleV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		WITH locked_playback_q AS (
			SELECT r.id AS room_id
			FROM rooms r
			LEFT JOIN playback_state a ON a.room_id = r.id
			WHERE r.id = $1
			AND (
				a.current_playlist_item_id IS NULL
				OR NOT EXISTS (
					SELECT 1
					FROM playlist_items b
					WHERE b.room_id = a.room_id
					AND b.id = a.current_playlist_item_id
					AND b.source_type = ANY($2::text[])
				)
			)
			FOR UPDATE OF r SKIP LOCKED
		),
		next_playlist_item_q AS (
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
					(SELECT d.name FROM sessions d WHERE d.id = a.added_by),
					a.added_by_nickname
				) AS added_by_nickname,
				a.added_at,
				COUNT(b.user_id) AS vote_count
			FROM playlist_items a
			JOIN locked_playback_q c ON c.room_id = a.room_id
			LEFT JOIN playlist_item_votes b
			ON a.id = b.playlist_item_id
			AND a.room_id = b.room_id
			WHERE a.source_type = ANY($2::text[])
			GROUP BY a.id, a.room_id, a.source_type, a.source_id, a.provider_url, a.playback_restriction, a.title, a.publisher, a.thumbnail_url, a.duration, a.added_by, a.added_by_nickname, a.added_at
			ORDER BY vote_count DESC, MAX(b.created_at) ASC, a.added_at ASC
			LIMIT 1
		),
		updated_playback_q AS (
			INSERT INTO playback_state (room_id, current_playlist_item_id, is_playing, position_ms, updated_at)
			SELECT room_id, id, TRUE, 0, NOW()
			FROM next_playlist_item_q
			ON CONFLICT (room_id) DO UPDATE
			SET current_playlist_item_id = EXCLUDED.current_playlist_item_id,
				is_playing = TRUE,
				position_ms = 0,
				updated_at = EXCLUDED.updated_at
			WHERE playback_state.current_playlist_item_id IS NULL
			OR NOT EXISTS (
				SELECT 1 FROM playlist_items s
				WHERE s.room_id = playback_state.room_id
				AND s.id = playback_state.current_playlist_item_id
				AND s.source_type = ANY($2::text[])
			)
			RETURNING room_id, current_playlist_item_id, is_playing, position_ms, updated_at
		)
		SELECT
			a.room_id,
			a.current_playlist_item_id,
			a.is_playing,
			a.position_ms,
			a.updated_at,
			b.id,
			b.room_id,
			b.source_type,
			b.source_id,
			b.provider_url,
			b.playback_restriction,
			b.title,
			b.publisher,
			b.thumbnail_url,
			b.duration,
			b.added_by,
			COALESCE(
				(SELECT d.name FROM sessions d WHERE d.id = b.added_by),
				b.added_by_nickname
			) AS added_by_nickname,
			b.added_at,
			COALESCE(b.vote_count, 0) AS vote_count
		FROM updated_playback_q a
		JOIN next_playlist_item_q b ON b.id = a.current_playlist_item_id
	`)
	if err != nil {
		return fmt.Errorf("error preparing StartPlaybackIfIdleV2Statement: %w", err)
	}

	c.StartPlaybackIfIdleV2Statement = stmt

	return nil
}

func (c *Client) startPlaybackIfIdleV2(ctx context.Context, roomID string) (*vibe.PlaybackStateV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "startPlaybackIfIdleV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.StartPlaybackIfIdleV2Statement.QueryRowContext(
		cctx,
		roomID,
		c.enabledProviders,
	)

	var row playbackPlaylistItemRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.PlaybackStateV2{}, nil
		}
		return nil, fmt.Errorf("error scanning playback state in startPlaybackIfIdleV2: %w", err)
	}

	state, err := row.toPlaybackStateV2()
	if err != nil {
		return nil, fmt.Errorf("error converting playback state in startPlaybackIfIdleV2: %w", err)
	}

	return state, nil
}

// StartPlaybackIfIdleV2 attempts to start playback if the room is currently idle.
// It returns the new state if it successfully started playback, or the current state if it didn't.
func (c *Client) StartPlaybackIfIdleV2(ctx context.Context, roomID string) (*vibe.PlaybackStateV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "StartPlaybackIfIdleV2")
	defer span.End()

	startedState, err := c.startPlaybackIfIdleV2(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error starting playback if idle in StartPlaybackIfIdleV2: %w", err)
	}

	if startedState.RoomID == "" || startedState.CurrentPlaylistItem == nil {
		state, err := c.getPlaybackStateV2(ctx, roomID)
		if err != nil {
			return nil, fmt.Errorf("error getting playback state in StartPlaybackIfIdleV2: %w", err)
		}
		return state, nil
	}

	return startedState, nil
}

func (c *Client) checkHostPermissions(ctx context.Context, roomID, userID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "checkHostPermissions")
	defer span.End()

	if userID == "" {
		return nil
	}

	// Register user as active participant
	err := c.UpdateParticipant(ctx, roomID, userID, true, false, "")
	if err != nil {
		return fmt.Errorf("error updating participant in check host permission in %s for %s: %w", roomID, userID, err)
	}

	room, err := c.GetRoomV2(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("error getting room in check host permission in %s for %s: %w", roomID, userID, err)
	}

	if room.Mode != vibe.RoomModeHost {
		return fmt.Errorf("error shared playback controls are unavailable in server mode")
	}

	if room.HostID == "" {
		err := c.SetRoomHost(ctx, roomID, userID)
		if err != nil {
			return fmt.Errorf("error setting room host in check host permission in %s for %s: %w", roomID, userID, err)
		}
		return nil
	}

	if room.HostID == userID {
		return nil
	}

	activeParticipants, err := c.GetActiveParticipants(ctx, roomID, 30*time.Second)
	if err != nil {
		return fmt.Errorf("error getting active participants in check host permission in %s for %s: %w", roomID, userID, err)
	}

	hostStillActive := false
	for _, p := range activeParticipants {
		if p.UserID == room.HostID && p.IsActiveListener {
			hostStillActive = true
			break
		}
	}

	if !hostStillActive {
		err := c.SetRoomHost(ctx, roomID, userID)
		if err != nil {
			return fmt.Errorf("error setting room host in check host permission in %s for %s: %w", roomID, userID, err)
		}
		return nil
	}

	return fmt.Errorf("error only the host can perform this action in %s for %s", roomID, userID)
}
