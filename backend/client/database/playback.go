package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zoff-music/vibes/internalerror"
	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"
)

// prepareGetPlaybackStateStmt prepares the GetPlaybackStateStatement.
func (c *Client) prepareGetPlaybackStateStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT room_id, current_song_id, is_playing, position_ms, updated_at
		FROM playback_state
		WHERE room_id = ?1
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
		SELECT ps.room_id
		FROM playback_state ps
		JOIN songs s ON ps.current_song_id = s.id
		JOIN rooms r ON ps.room_id = r.id
		WHERE ps.is_playing = 1
		  AND (r.mode = 'server' OR r.mode = 'host')
		  AND ((unixepoch('now') - unixepoch(ps.updated_at)) * 1000 + ps.position_ms) >= (s.duration * 1000 - 2000)
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

	var roomID string
	err := c.ProcessNextExpiredPlaybackStatement.QueryRowContext(cctx).Scan(&roomID)
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

	// Skip the track
	// skipTrack is internal and doesn't check permissions, which is what we want here
	newState, err := c.skipTrack(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error skipping track for room %s: %w", roomID, err)
	}

	return newState, nil
}

// GetPlaybackState fetches the playback state for a room.
func (c *Client) GetPlaybackState(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetPlaybackState")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetPlaybackStateStatement.QueryRowContext(cctx, roomID)

	var scanned playbackStateRow
	err := scanned.scan(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return default state if no state exists
			return &vibe.PlaybackState{
				RoomID:        roomID,
				CurrentSongID: nil,
				IsPlaying:     false,
				PositionMs:    0,
				UpdatedAt:     time.Now(),
				ServerTimeMs:  time.Now().UnixMilli(),
			}, nil
		}

		return nil, fmt.Errorf("error fetching playback state: %w", err)
	}

	state, err := scanned.toPlaybackState()
	if err != nil {
		return nil, fmt.Errorf("error converting playback state row: %w", err)
	}

	if state.CurrentSongID != nil {
		song, err := c.GetSong(ctx, state.RoomID, *state.CurrentSongID)
		if err != nil {
			// Log error but don't fail, just return state without song detail
			// or maybe we should fail?
			// Let's just return nil song if not found (consistency issue)
			// For now, logging would be good but we don't have logger here.
			// Let's return error because consistency is important.
			// Actually, if song is deleted but playback state references it, that's a DB consistency issue.
			// We'll ignore the error and return nil song to recover gracefully.
		} else if !song.IsEmpty() {
			state.CurrentSong = song
		}
	}

	return state, nil
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
	var currentSongID *string
	if r.CurrentSongID.Valid && r.CurrentSongID.String != "" {
		currentSongID = &r.CurrentSongID.String
	}

	if !r.UpdatedAt.Valid {
		return nil, fmt.Errorf("error missing playback_state updated_at")
	}

	positionMs := int64(0)
	if r.PositionMs.Valid {
		positionMs = r.PositionMs.Int64
	}

	return &vibe.PlaybackState{
		RoomID:        r.RoomID.String,
		CurrentSongID: currentSongID,
		CurrentSong:   nil,
		IsPlaying:     r.IsPlaying.Valid && r.IsPlaying.Int64 == 1,
		PositionMs:    positionMs,
		UpdatedAt:     r.UpdatedAt.Time,
		ServerTimeMs:  time.Now().UnixMilli(),
	}, nil
}

// prepareUpsertPlaybackStateStmt prepares the UpsertPlaybackStateStatement.
func (c *Client) prepareUpsertPlaybackStateStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO playback_state (room_id, current_song_id, is_playing, position_ms, updated_at)
		VALUES (?1, ?2, ?3, ?4, ?5)
		ON CONFLICT(room_id) DO UPDATE SET
			current_song_id = excluded.current_song_id,
			is_playing = excluded.is_playing,
			position_ms = excluded.position_ms,
			updated_at = excluded.updated_at
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

	isPlaying := 0
	if state.IsPlaying {
		isPlaying = 1
	}

	_, err := c.UpsertPlaybackStateStatement.ExecContext(cctx,
		state.RoomID,
		state.CurrentSongID,
		isPlaying,
		state.PositionMs,
		state.UpdatedAt,
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

	state, err := c.GetPlaybackState(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("skip track: get playback state: %w", err)
	}

	currentPosition := -1
	if state.CurrentSong != nil {
		currentPosition = state.CurrentSong.Position
	} else if state.CurrentSongID != nil {
		// If we only have ID, we might need to fetch it to know position,
		// but typically state has CurrentSong populated if ID is present.
	}

	nextSong, err := c.GetNextSong(ctx, roomID, currentPosition)
	if err != nil {
		return nil, fmt.Errorf("skip track: get next song: %w", err)
	}

	if nextSong.IsEmpty() {
		state.CurrentSongID = nil
		state.CurrentSong = nil
		state.IsPlaying = false
	} else {
		state.CurrentSongID = &nextSong.ID
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

// SkipTrack skips the current track to the next one in the queue.
func (c *Client) SkipTrack(ctx context.Context, roomID string, userID string) (*vibe.PlaybackState, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SkipTrack")
	defer span.Finish()

	err := c.checkHostPermissions(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}

	return c.skipTrack(ctx, roomID)
}

// UpdatePlayback updates the playback state based on the action (play/pause/seek).
func (c *Client) UpdatePlayback(ctx context.Context, roomID string, userID string, action vibe.RoomAction, positionMs int64) (*vibe.PlaybackState, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UpdatePlayback")
	defer span.Finish()

	err := c.checkHostPermissions(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}

	state, err := c.GetPlaybackState(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("update playback: get playback state: %w", err)
	}

	switch action {
	case vibe.RoomActionPlay:
		if state.CurrentSongID == nil {
			// If no song is selected, try to play the first one
			firstSong, err := c.GetNextSong(ctx, roomID, 0)
			if err == nil && !firstSong.IsEmpty() {
				state.CurrentSongID = &firstSong.ID
				state.CurrentSong = firstSong
				state.PositionMs = 0
			}
		}
		state.IsPlaying = true
	case vibe.RoomActionPause:
		// Calculate projected position before pausing
		if state.IsPlaying {
			elapsed := time.Since(state.UpdatedAt).Milliseconds()
			state.PositionMs = state.PositionMs + elapsed
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

func (c *Client) checkHostPermissions(ctx context.Context, roomID, userID string) error {
	if userID == "" {
		return nil
	}

	// Register user as active participant
	_ = c.UpdateParticipant(ctx, roomID, userID, true)

	room, err := c.GetRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to fetch room: %w", err)
	}

	if room.Mode != vibe.RoomModeHost {
		return nil
	}

	if room.HostID == "" {
		// Claim host
		err := c.SetRoomHost(ctx, roomID, userID)
		if err != nil {
			return fmt.Errorf("failed to set room host: %w", err)
		}
		return nil
	}

	if room.HostID == userID {
		return nil
	}

	// Check if current host is still active
	activeParticipants, err := c.GetActiveParticipants(ctx, roomID, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to get active participants: %w", err)
	}

	hostStillActive := false
	for _, p := range activeParticipants {
		if p.UserID == room.HostID {
			hostStillActive = true
			break
		}
	}

	if !hostStillActive {
		// Previous host inactive, claim host
		err := c.SetRoomHost(ctx, roomID, userID)
		if err != nil {
			return fmt.Errorf("failed to set room host: %w", err)
		}
		return nil
	}

	return fmt.Errorf("only the host can perform this action")
}
