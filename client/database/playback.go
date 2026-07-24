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

// prepareGetPlaybackStateStmt prepares the GetPlaybackStateStatement.
func (c *Client) prepareGetPlaybackStateStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT room_id, current_song_id, is_playing, position_ms, updated_at
		FROM playback_state
		WHERE room_id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetPlaybackStateStatement: %w", err)
	}

	c.GetPlaybackStateStatement = stmt

	return nil
}

// getPlaybackState fetches the playback state for a room (internal).
func (c *Client) getPlaybackState(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getPlaybackState")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetPlaybackStateStatement.QueryRowContext(cctx, roomID)

	var row playbackStateRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.PlaybackState{
				RoomID:       roomID,
				CurrentSong:  nil,
				IsPlaying:    false,
				PositionMs:   0,
				UpdatedAt:    time.Now(),
				ServerTimeMs: int(time.Now().UnixMilli()),
			}, nil
		}

		return nil, fmt.Errorf("error fetching playback state: %w", err)
	}

	state, err := row.toPlaybackState()
	if err != nil {
		return nil, fmt.Errorf("error converting playback state row: %w", err)
	}

	if !row.CurrentSongID.Valid || row.CurrentSongID.String == "" {
		return state, nil
	}

	song, err := c.GetSong(ctx, state.RoomID, row.CurrentSongID.String)
	if err != nil {
		return nil, fmt.Errorf("error get current song %s: %w", row.CurrentSongID.String, err)
	}

	if song.IsEmpty() {
		return state, nil
	}

	state.CurrentSong = song
	if !state.IsPlaying {
		return state, nil
	}

	elapsed := time.Since(state.UpdatedAt).Milliseconds()
	currentPosition := state.PositionMs + int(elapsed)

	if state.CurrentSong.Duration > 0 {
		duration := state.CurrentSong.Duration * 1000
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

// GetPlaybackState fetches the playback state for a room.
// It will automatically attempt to start playback if the room is idle.
func (c *Client) GetPlaybackState(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetPlaybackState")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	state, err := c.getPlaybackState(cctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error get playback state: %w", err)
	}

	// Lazy-skip check on the PUBLIC API only
	if state.IsPlaying && state.CurrentSong != nil {
		elapsed := time.Since(state.UpdatedAt).Milliseconds()
		currentPosition := state.PositionMs + int(elapsed)

		if state.CurrentSong.Duration > 0 && currentPosition >= state.CurrentSong.Duration*1000 {
			newState, err := c.skipTrack(ctx, roomID)
			if err != nil {
				return nil, fmt.Errorf("error lazy skipping playback: %w", err)
			}
			return newState, nil
		}
	}

	if state.CurrentSong != nil {
		return state, nil
	}

	newState, err := c.StartPlaybackIfIdle(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error auto-start playback: %w", err)
	}

	return newState, nil
}

type playbackStateRow struct {
	RoomID        sql.NullString
	CurrentSongID sql.NullString
	IsPlaying     sql.NullBool
	PositionMs    sql.NullInt64
	UpdatedAt     sql.NullTime
}

func (r *playbackStateRow) scan(row *sql.Row) error {
	return row.Scan(
		&r.RoomID,
		&r.CurrentSongID,
		&r.IsPlaying,
		&r.PositionMs,
		&r.UpdatedAt,
	)
}

func (r *playbackStateRow) toPlaybackState() (*vibe.PlaybackState, error) {
	return &vibe.PlaybackState{
		RoomID:       r.RoomID.String,
		CurrentSong:  nil,
		IsPlaying:    r.IsPlaying.Bool,
		PositionMs:   int(r.PositionMs.Int64),
		UpdatedAt:    r.UpdatedAt.Time,
		ServerTimeMs: int(time.Now().UnixMilli()),
	}, nil
}

type playbackSongRow struct {
	PlaybackRoomID        sql.NullString
	PlaybackCurrentSongID sql.NullString
	PlaybackIsPlaying     sql.NullBool
	PlaybackPositionMs    sql.NullInt64
	PlaybackUpdatedAt     sql.NullTime
	Song                  songRow
}

func (r *playbackSongRow) scan(row *sql.Row) error {
	return row.Scan(
		&r.PlaybackRoomID,
		&r.PlaybackCurrentSongID,
		&r.PlaybackIsPlaying,
		&r.PlaybackPositionMs,
		&r.PlaybackUpdatedAt,
		&r.Song.ID,
		&r.Song.RoomID,
		&r.Song.SourceType,
		&r.Song.SourceID,
		&r.Song.Title,
		&r.Song.Artist,
		&r.Song.ThumbnailURL,
		&r.Song.Duration,
		&r.Song.AddedBy,
		&r.Song.AddedByNickname,
		&r.Song.AddedAt,
		&r.Song.VoteCount,
	)
}

func (r *playbackSongRow) toPlaybackState() (*vibe.PlaybackState, error) {
	state := &vibe.PlaybackState{
		RoomID:       r.PlaybackRoomID.String,
		CurrentSong:  nil,
		IsPlaying:    r.PlaybackIsPlaying.Bool,
		PositionMs:   int(r.PlaybackPositionMs.Int64),
		UpdatedAt:    r.PlaybackUpdatedAt.Time,
		ServerTimeMs: int(time.Now().UnixMilli()),
	}

	if !r.Song.ID.Valid || r.Song.ID.String == "" {
		return state, nil
	}

	song := r.Song.toSong()
	state.CurrentSong = &song

	return state, nil
}

// prepareProcessNextExpiredPlaybackStmt prepares the ProcessNextExpiredPlaybackStatement.
func (c *Client) prepareProcessNextExpiredPlaybackStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH locked_playback_q AS (
			SELECT
				a.room_id,
				a.current_song_id,
				COALESCE(d.remove_on_play, FALSE) AS remove_on_play
			FROM playback_state a
			JOIN songs b ON a.current_song_id = b.id
			JOIN rooms c ON a.room_id = c.id
			JOIN room_settings d ON d.room_id = c.id
			WHERE a.is_playing
			AND (c.mode = 'server' OR c.mode = 'host')
			AND ((EXTRACT(EPOCH FROM (NOW() - a.updated_at)) * 1000) + a.position_ms) >= (b.duration * 1000 - 500)
			LIMIT 1
			FOR UPDATE OF a SKIP LOCKED
		),
		cleared_skip_votes_q AS (
			DELETE FROM skip_votes a
			USING locked_playback_q b
			WHERE a.room_id = b.room_id
			AND a.song_id = b.current_song_id
			RETURNING 1
		),
		cleared_song_votes_q AS (
			DELETE FROM song_votes a
			USING locked_playback_q b
			WHERE a.room_id = b.room_id
			AND a.song_id = b.current_song_id
			RETURNING 1
		),
		removed_song_q AS (
			DELETE FROM songs a
			USING locked_playback_q b
			WHERE b.remove_on_play
			AND a.room_id = b.room_id
			AND a.id = b.current_song_id
			RETURNING 1
		),
		requeued_song_q AS (
			UPDATE songs a
			SET added_at = NOW()
			FROM locked_playback_q b
			WHERE NOT b.remove_on_play
			AND a.room_id = b.room_id
			AND a.id = b.current_song_id
			RETURNING 1
		),
		next_song_q AS (
			SELECT
				a.id,
				a.room_id,
				a.source_type,
				a.source_id,
				a.title,
				a.artist,
				a.thumbnail_url,
				a.duration,
				a.added_by,
				a.added_by_nickname,
				CASE
					WHEN a.id = c.current_song_id AND NOT c.remove_on_play THEN NOW()
					ELSE a.added_at
				END AS added_at,
				COUNT(b.user_id) AS vote_count
			FROM songs a
			JOIN locked_playback_q c ON c.room_id = a.room_id
			LEFT JOIN song_votes b
			ON a.id = b.song_id
			AND a.room_id = b.room_id
			WHERE NOT (c.remove_on_play AND a.id = c.current_song_id)
			GROUP BY a.id, a.room_id, a.source_type, a.source_id, a.title, a.artist, a.thumbnail_url, a.duration, a.added_by, a.added_by_nickname, a.added_at, c.current_song_id, c.remove_on_play
			ORDER BY vote_count DESC, MAX(b.created_at) ASC, added_at ASC
			LIMIT 1
		),
		updated_playback_q AS (
			UPDATE playback_state a
			SET current_song_id = b.id,
			is_playing = b.id IS NOT NULL,
			position_ms = 0,
			updated_at = NOW()
			FROM locked_playback_q c
			LEFT JOIN next_song_q b ON b.room_id = c.room_id
			WHERE a.room_id = c.room_id
			RETURNING a.room_id, a.current_song_id, a.is_playing, a.position_ms, a.updated_at
		)
		SELECT
			a.room_id,
			a.current_song_id,
			a.is_playing,
			a.position_ms,
			a.updated_at,
			b.id,
			b.room_id,
			b.source_type,
			b.source_id,
			b.title,
			b.artist,
			b.thumbnail_url,
			b.duration,
			b.added_by,
			b.added_by_nickname,
			b.added_at,
			COALESCE(b.vote_count, 0) AS vote_count
		FROM updated_playback_q a
		LEFT JOIN next_song_q b ON b.id = a.current_song_id
	`)
	if err != nil {
		return fmt.Errorf("error preparing ProcessNextExpiredPlaybackStatement: %w", err)
	}

	c.ProcessNextExpiredPlaybackStatement = stmt

	return nil
}

func (c *Client) processNextExpiredPlayback(ctx context.Context) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "processNextExpiredPlayback")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.ProcessNextExpiredPlaybackStatement.QueryRowContext(cctx)

	var row playbackSongRow
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

	state, err := row.toPlaybackState()
	if err != nil {
		return nil, fmt.Errorf("error converting expired playback state: %w", err)
	}

	return state, nil
}

// ProcessNextExpiredPlayback checks for an expired song, skips it, and returns the new state.
func (c *Client) ProcessNextExpiredPlayback(ctx context.Context) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ProcessNextExpiredPlayback")
	defer span.End()

	state, err := c.processNextExpiredPlayback(ctx)
	if err != nil {
		return nil, fmt.Errorf("error processing next expired playback: %w", err)
	}

	return state, nil
}

// prepareUpsertPlaybackStateStmt prepares the UpsertPlaybackStateStatement.
func (c *Client) prepareUpsertPlaybackStateStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO playback_state (room_id, current_song_id, is_playing, position_ms, updated_at)
			SELECT a.id, $2, $3, $4, CURRENT_TIMESTAMP
			FROM rooms a
			WHERE a.id = $1
			FOR KEY SHARE OF a
			ON CONFLICT(room_id) DO UPDATE SET
			current_song_id = EXCLUDED.current_song_id,
			is_playing = EXCLUDED.is_playing,
			position_ms = EXCLUDED.position_ms,
			updated_at = EXCLUDED.updated_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpsertPlaybackStateStatement: %w", err)
	}

	c.UpsertPlaybackStateStatement = stmt

	return nil
}

// UpsertPlaybackState creates or updates the playback state for a room.
func (c *Client) UpsertPlaybackState(ctx context.Context, state *vibe.PlaybackState) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpsertPlaybackState")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var currentSongID string
	if state.CurrentSong != nil {
		currentSongID = state.CurrentSong.ID
	}

	_, err := c.UpsertPlaybackStateStatement.ExecContext(cctx,
		state.RoomID,
		currentSongID,
		state.IsPlaying,
		state.PositionMs,
	)
	if err != nil {
		return fmt.Errorf("error upserting playback state: %w", err)
	}

	return nil
}

func (c *Client) prepareSkipTrackStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH locked_playback_q AS (
			SELECT
				a.room_id,
				a.current_song_id,
				COALESCE(b.remove_on_play, FALSE) AS remove_on_play
			FROM playback_state a
			JOIN room_settings b ON b.room_id = a.room_id
			WHERE a.room_id = $1
			FOR UPDATE OF a SKIP LOCKED
		),
		cleared_skip_votes_q AS (
			DELETE FROM skip_votes a
			USING locked_playback_q b
			WHERE a.room_id = b.room_id
			AND a.song_id = b.current_song_id
			RETURNING 1
		),
		cleared_song_votes_q AS (
			DELETE FROM song_votes a
			USING locked_playback_q b
			WHERE a.room_id = b.room_id
			AND a.song_id = b.current_song_id
			RETURNING 1
		),
		removed_song_q AS (
			DELETE FROM songs a
			USING locked_playback_q b
			WHERE b.current_song_id IS NOT NULL
			AND b.remove_on_play
			AND a.room_id = b.room_id
			AND a.id = b.current_song_id
			RETURNING 1
		),
		requeued_song_q AS (
			UPDATE songs a
			SET added_at = NOW()
			FROM locked_playback_q b
			WHERE b.current_song_id IS NOT NULL
			AND NOT b.remove_on_play
			AND a.room_id = b.room_id
			AND a.id = b.current_song_id
			RETURNING 1
		),
		next_song_q AS (
			SELECT
				a.id,
				a.room_id,
				a.source_type,
				a.source_id,
				a.title,
				a.artist,
				a.thumbnail_url,
				a.duration,
				a.added_by,
				a.added_by_nickname,
				CASE
					WHEN a.id = c.current_song_id AND NOT c.remove_on_play THEN NOW()
					ELSE a.added_at
				END AS added_at,
				COUNT(b.user_id) AS vote_count
			FROM songs a
			JOIN locked_playback_q c ON c.room_id = a.room_id
			LEFT JOIN song_votes b
			ON a.id = b.song_id
			AND a.room_id = b.room_id
			WHERE NOT (c.remove_on_play AND a.id = c.current_song_id)
			GROUP BY a.id, a.room_id, a.source_type, a.source_id, a.title, a.artist, a.thumbnail_url, a.duration, a.added_by, a.added_by_nickname, a.added_at, c.current_song_id, c.remove_on_play
			ORDER BY vote_count DESC, MAX(b.created_at) ASC, added_at ASC
			LIMIT 1
		),
		updated_playback_q AS (
			UPDATE playback_state a
			SET current_song_id = b.id,
			is_playing = b.id IS NOT NULL,
			position_ms = 0,
			updated_at = NOW()
			FROM locked_playback_q c
			LEFT JOIN next_song_q b ON b.room_id = c.room_id
			WHERE a.room_id = c.room_id
			RETURNING a.room_id, a.current_song_id, a.is_playing, a.position_ms, a.updated_at
		)
		SELECT
			a.room_id,
			a.current_song_id,
			a.is_playing,
			a.position_ms,
			a.updated_at,
			b.id,
			b.room_id,
			b.source_type,
			b.source_id,
			b.title,
			b.artist,
			b.thumbnail_url,
			b.duration,
			b.added_by,
			b.added_by_nickname,
			b.added_at,
			COALESCE(b.vote_count, 0) AS vote_count
		FROM updated_playback_q a
		LEFT JOIN next_song_q b ON b.id = a.current_song_id
	`)
	if err != nil {
		return fmt.Errorf("error preparing SkipTrackStatement: %w", err)
	}

	c.SkipTrackStatement = stmt

	return nil
}

// skipTrack skips the current track to the next one in the queue (internal).
func (c *Client) skipTrack(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "skipTrack")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.SkipTrackStatement.QueryRowContext(cctx, roomID)

	var row playbackSongRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			state, err := c.getPlaybackState(ctx, roomID)
			if err != nil {
				return nil, fmt.Errorf("error getting playback state in skipTrack: %w", err)
			}
			return state, nil
		}
		return nil, fmt.Errorf("error scanning playback state in skipTrack: %w", err)
	}

	state, err := row.toPlaybackState()
	if err != nil {
		return nil, fmt.Errorf("error converting playback state in skipTrack: %w", err)
	}

	return state, nil
}

// UpdatePlayback updates the playback state based on the action (play/pause/seek).
func (c *Client) UpdatePlayback(ctx context.Context, roomID string, userID string, action vibe.RoomAction, positionMs int) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdatePlayback")
	defer span.End()

	err := c.checkHostPermissions(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking host permissions in update playback in %s: %w", roomID, err)
	}

	state, err := c.GetPlaybackState(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error getting playback state in UpdatePlayback: %w", err)
	}

	switch action {
	case vibe.RoomActionPlay:
		state.IsPlaying = true
		if state.CurrentSong != nil {
			break
		}

		state, err = c.StartPlaybackIfIdle(ctx, roomID)
		if err != nil {
			return nil, fmt.Errorf("error starting playback in UpdatePlayback: %w", err)
		}

		if state.CurrentSong == nil {
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
		return nil, fmt.Errorf("error invalid action in UpdatePlayback: %s", action)
	}

	state.UpdatedAt = time.Now()

	err = c.UpsertPlaybackState(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("error upserting playback state in UpdatePlayback: %w", err)
	}

	return state, nil
}

func (c *Client) prepareStartPlaybackIfIdleStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH locked_playback_q AS (
			SELECT a.room_id
			FROM playback_state a
			WHERE a.room_id = $1
			AND a.current_song_id IS NULL
			FOR UPDATE OF a SKIP LOCKED
		),
		next_song_q AS (
			SELECT
				a.id,
				a.room_id,
				a.source_type,
				a.source_id,
				a.title,
				a.artist,
				a.thumbnail_url,
				a.duration,
				a.added_by,
				a.added_by_nickname,
				a.added_at,
				COUNT(b.user_id) AS vote_count
			FROM songs a
			JOIN locked_playback_q c ON c.room_id = a.room_id
			LEFT JOIN song_votes b
			ON a.id = b.song_id
			AND a.room_id = b.room_id
			GROUP BY a.id, a.room_id, a.source_type, a.source_id, a.title, a.artist, a.thumbnail_url, a.duration, a.added_by, a.added_by_nickname, a.added_at
			ORDER BY vote_count DESC, MAX(b.created_at) ASC, a.added_at ASC
			LIMIT 1
		),
		updated_playback_q AS (
			UPDATE playback_state a
			SET current_song_id = b.id,
			is_playing = TRUE,
			position_ms = 0,
			updated_at = NOW()
			FROM locked_playback_q c
			JOIN next_song_q b ON b.room_id = c.room_id
			WHERE a.room_id = c.room_id
			RETURNING a.room_id, a.current_song_id, a.is_playing, a.position_ms, a.updated_at
		)
		SELECT
			a.room_id,
			a.current_song_id,
			a.is_playing,
			a.position_ms,
			a.updated_at,
			b.id,
			b.room_id,
			b.source_type,
			b.source_id,
			b.title,
			b.artist,
			b.thumbnail_url,
			b.duration,
			b.added_by,
			b.added_by_nickname,
			b.added_at,
			COALESCE(b.vote_count, 0) AS vote_count
		FROM updated_playback_q a
		JOIN next_song_q b ON b.id = a.current_song_id
	`)
	if err != nil {
		return fmt.Errorf("error preparing StartPlaybackIfIdleStatement: %w", err)
	}

	c.StartPlaybackIfIdleStatement = stmt

	return nil
}

func (c *Client) startPlaybackIfIdle(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "startPlaybackIfIdle")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.StartPlaybackIfIdleStatement.QueryRowContext(cctx, roomID)

	var row playbackSongRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.PlaybackState{}, nil
		}
		return nil, fmt.Errorf("error scanning playback state in startPlaybackIfIdle: %w", err)
	}

	state, err := row.toPlaybackState()
	if err != nil {
		return nil, fmt.Errorf("error converting playback state in startPlaybackIfIdle: %w", err)
	}

	return state, nil
}

// StartPlaybackIfIdle attempts to start playback if the room is currently idle.
// It returns the new state if it successfully started playback, or the current state if it didn't.
func (c *Client) StartPlaybackIfIdle(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "StartPlaybackIfIdle")
	defer span.End()

	startedState, err := c.startPlaybackIfIdle(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error starting playback if idle in StartPlaybackIfIdle: %w", err)
	}

	if startedState.RoomID == "" || startedState.CurrentSong == nil {
		state, err := c.getPlaybackState(ctx, roomID)
		if err != nil {
			return nil, fmt.Errorf("error getting playback state in StartPlaybackIfIdle: %w", err)
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

	room, err := c.GetRoom(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("error getting room in check host permission in %s for %s: %w", roomID, userID, err)
	}

	if room.Mode != vibe.RoomModeHost {
		return nil
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
