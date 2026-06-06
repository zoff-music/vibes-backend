package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/opentracing"
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

// prepareProcessNextExpiredPlaybackStmt prepares the ProcessNextExpiredPlaybackStatement.
func (c *Client) prepareProcessNextExpiredPlaybackStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT a.room_id
		FROM playback_state a
		JOIN songs b 
		ON a.current_song_id = b.id
		JOIN rooms c 
		ON a.room_id = c.id
		WHERE a.is_playing = 1
		AND (c.mode = 'server' OR c.mode = 'host')
		AND ((EXTRACT(EPOCH FROM (NOW() - a.updated_at)) * 1000) + a.position_ms) >= (b.duration * 1000 - 500)
		LIMIT 1
	`)
	if err != nil {
		return fmt.Errorf("error preparing ProcessNextExpiredPlaybackStatement: %w", err)
	}

	c.ProcessNextExpiredPlaybackStatement = stmt

	return nil
}

// ProcessNextExpiredPlayback checks for an expired song, skips it, and returns the new state.
func (c *Client) ProcessNextExpiredPlayback(ctx context.Context) (*vibe.PlaybackState, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ProcessNextExpiredPlayback")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.ProcessNextExpiredPlaybackStatement.QueryRowContext(cctx)

	var row sql.NullString
	err := r.Scan(&row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf("no expired playback found"),
				},
			}
		}

		return nil, fmt.Errorf("error finding expired playback: %w", err)
	}

	roomID := row.String

	// Skip the track
	// skipTrack is internal and doesn't check permissions, which is what we want here
	newState, err := c.skipTrack(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error skipping track for room %s: %w", roomID, err)
	}

	return newState, nil
}

// getPlaybackState fetches the playback state for a room (internal).
func (c *Client) getPlaybackState(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "getPlaybackState")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetPlaybackStateStatement.QueryRowContext(cctx, roomID)

	var row playbackStateRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return default state if no state exists
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetPlaybackState")
	defer span.Finish()

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
				return nil, fmt.Errorf("lazy skip: %w", err)
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
	IsPlaying     sql.NullInt64
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
		IsPlaying:    r.IsPlaying.Valid && r.IsPlaying.Int64 == 1,
		PositionMs:   int(r.PositionMs.Int64),
		UpdatedAt:    r.UpdatedAt.Time,
		ServerTimeMs: int(time.Now().UnixMilli()),
	}, nil
}

// prepareUpsertPlaybackStateStmt prepares the UpsertPlaybackStateStatement.
func (c *Client) prepareUpsertPlaybackStateStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO playback_state (room_id, current_song_id, is_playing, position_ms, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "UpsertPlaybackState")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var currentSongID string
	if state.CurrentSong != nil {
		currentSongID = state.CurrentSong.ID
	}

	_, err := c.UpsertPlaybackStateStatement.ExecContext(cctx,
		state.RoomID,
		currentSongID,
		boolToInt(state.IsPlaying),
		state.PositionMs,
	)
	if err != nil {
		return fmt.Errorf("error upserting playback state: %w", err)
	}

	return nil
}

// skipTrack skips the current track to the next one in the queue (internal).
func (c *Client) skipTrack(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "skipTrack")
	defer span.Finish()

	// Use internal getPlaybackState to avoid recursion / lazy-skip trigger
	state, err := c.getPlaybackState(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("skip track: get playback state: %w", err)
	}

	room, err := c.GetRoom(ctx, roomID, "")
	if err != nil {
		return nil, fmt.Errorf("skip track: get room: %w", err)
	}

	var currentSongID string
	if state.CurrentSong != nil {
		currentSongID = state.CurrentSong.ID
	}

	// Clear votes for the current song when it ends/is skipped
	if currentSongID != "" {
		log.Printf("[DEBUG-VOTES] Clearing votes for current song %s in room %s during skip", currentSongID, roomID)

		err = c.clearSkipVotes(ctx, roomID, currentSongID)
		if err != nil {
			return nil, fmt.Errorf("skip track: clear skip votes: %w", err)
		}

		err = c.clearVotesSong(ctx, roomID, currentSongID)
		if err != nil {
			return nil, fmt.Errorf("skip track: clear song votes: %w", err)
		}

		if room.Settings.RemoveOnPlay {
			err = c.RemoveSong(ctx, roomID, currentSongID)
			if err != nil {
				return nil, fmt.Errorf("skip track: remove song: %w", err)
			}
		}

		if !room.Settings.RemoveOnPlay {
			// If song stays in queue, update its added_at timestamp to treat it as "new"
			// This prevents it from being immediately replayed and moves it to the end of the queue.
			err = c.updateSongAddedAt(ctx, roomID, currentSongID)
			if err != nil {
				return nil, fmt.Errorf("skip track: update song added_at: %w", err)
			}
		}
	}

	nextSong, err := c.GetNextSong(ctx, roomID, 0)
	if err != nil {
		return nil, fmt.Errorf("skip track: get next song: %w", err)
	}

	log.Printf("[DEBUG-SKIP] next-song %+v", nextSong)

	// 3. Handle LoopQueue if no next song found
	if nextSong.IsEmpty() && room.Settings.LoopQueue {
		// We can just get the next song directly since logic is same for both branches due to prev updateSongAddedAt
		// If RemoveOnPlay was true, current was gone, so we get next (first)
		// If RemoveOnPlay was false, current was moved to end, so we get next (first)

		nextSong, err = c.GetNextSong(ctx, roomID, 0)
		if err != nil {
			return nil, fmt.Errorf("skip track: get first song for loop: %w", err)
		}
	}

	state.CurrentSong = nil
	state.IsPlaying = false
	if !nextSong.IsEmpty() {
		state.CurrentSong = nextSong
		state.IsPlaying = true
	}

	state.PositionMs = 0
	state.UpdatedAt = time.Now()

	err = c.UpsertPlaybackState(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("skip track: upsert playback state: %w", err)
	}

	return state, nil
}

// UpdatePlayback updates the playback state based on the action (play/pause/seek).
func (c *Client) UpdatePlayback(ctx context.Context, roomID string, userID string, action vibe.RoomAction, positionMs int) (*vibe.PlaybackState, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UpdatePlayback")
	defer span.Finish()

	err := c.checkHostPermissions(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking host permissions in update playback in %s: %w", roomID, err)
	}

	state, err := c.GetPlaybackState(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("update playback: get playback state: %w", err)
	}

	switch action {
	case vibe.RoomActionPlay:
		state.IsPlaying = true
		if state.CurrentSong != nil {
			break
		}

		// If no song is selected, try to play the first one
		firstSong, err := c.GetNextSong(ctx, roomID, 0)
		if err != nil {
			return nil, fmt.Errorf("update playback: get next song: %w", err)
		}

		if firstSong.IsEmpty() {
			break
		}

		state.CurrentSong = firstSong
		state.PositionMs = 0
	case vibe.RoomActionPause:
		if state.IsPlaying {
			elapsed := time.Since(state.UpdatedAt).Milliseconds()
			state.PositionMs = state.PositionMs + int(elapsed)
		}
		state.IsPlaying = false
	case vibe.RoomActionSeek:
		state.PositionMs = positionMs
	default:
		return nil, fmt.Errorf("update playback: invalid action: %s", action)
	}

	state.UpdatedAt = time.Now()

	err = c.UpsertPlaybackState(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("update playback: upsert playback state: %w", err)
	}

	return state, nil
}

func (c *Client) prepareStartPlaybackIfIdleStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE playback_state
		SET current_song_id = $1,
		is_playing = 1,
		updated_at = CURRENT_TIMESTAMP
		WHERE room_id = $2
		AND current_song_id IS NULL
	`)
	if err != nil {
		return fmt.Errorf("error preparing StartPlaybackIfIdleStatement: %w", err)
	}

	c.StartPlaybackIfIdleStatement = stmt

	return nil
}

// StartPlaybackIfIdle attempts to start playback if the room is currently idle.
// It returns the new state if it successfully started playback, or the current state if it didn't.
func (c *Client) StartPlaybackIfIdle(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "StartPlaybackIfIdle")
	defer span.Finish()

	// 1. Check if we have songs in the queue
	firstSong, err := c.GetNextSong(ctx, roomID, 0)
	if err != nil {
		return nil, fmt.Errorf("start playback if idle: get first song: %w", err)
	}

	if firstSong.IsEmpty() {
		// No songs, just return current state
		return c.getPlaybackState(ctx, roomID)
	}

	// 2. Atomic update: only update if current_song_id IS NULL
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := c.StartPlaybackIfIdleStatement.ExecContext(cctx, firstSong.ID, roomID)
	if err != nil {
		return nil, fmt.Errorf("start playback if idle: exec: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("start playback if idle: rows affected: %w", err)
	}

	if rows == 0 {
		// Someone else started it or it wasn't idle, return current state
		return c.getPlaybackState(ctx, roomID)
	}

	now := time.Now().UTC()
	// 3. Successfully started, return the new state
	return &vibe.PlaybackState{
		RoomID:       roomID,
		CurrentSong:  firstSong,
		IsPlaying:    true,
		PositionMs:   0,
		UpdatedAt:    now,
		ServerTimeMs: int(now.UnixMilli()),
	}, nil
}

func (c *Client) checkHostPermissions(ctx context.Context, roomID, userID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "checkHostPermissions")
	defer span.Finish()

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
		// Claim host
		err := c.SetRoomHost(ctx, roomID, userID)
		if err != nil {
			return fmt.Errorf("error setting room host in check host permission in %s for %s: %w", roomID, userID, err)
		}
		return nil
	}

	if room.HostID == userID {
		return nil
	}

	// Check if current host is still active
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
		// Previous host inactive, claim host
		err := c.SetRoomHost(ctx, roomID, userID)
		if err != nil {
			return fmt.Errorf("error setting room host in check host permission in %s for %s: %w", roomID, userID, err)
		}
		return nil
	}

	return fmt.Errorf("error only the host can perform this action in %s for %s", roomID, userID)
}
