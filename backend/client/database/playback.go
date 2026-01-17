package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"
)

// prepareGetPlaybackStateStmt prepares the GetPlaybackStateStatement.
func (c *Client) prepareGetPlaybackStateStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT room_id, current_song_id, is_playing, position_ms, updated_at
		FROM playback_state
		WHERE room_id = ?
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetPlaybackStateStatement: %w", err)
	}

	c.GetPlaybackStateStatement = stmt

	return nil
}

// GetPlaybackState fetches the playback state for a room.
func (c *Client) GetPlaybackState(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetPlaybackState")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var row playbackStateRow

	err := c.GetPlaybackStateStatement.QueryRowContext(cctx, roomID).Scan(
		&row.RoomID,
		&row.CurrentSongID,
		&row.IsPlaying,
		&row.PositionMs,
		&row.UpdatedAt,
	)
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

	state := row.toPlaybackState()

	return &state, nil
}

// prepareUpsertPlaybackStateStmt prepares the UpsertPlaybackStateStatement.
func (c *Client) prepareUpsertPlaybackStateStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO playback_state (room_id, current_song_id, is_playing, position_ms, updated_at)
		VALUES (?, ?, ?, ?, ?)
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

// --- Internal types and helpers ---

type playbackStateRow struct {
	RoomID        string
	CurrentSongID sql.NullString
	IsPlaying     int
	PositionMs    int64
	UpdatedAt     time.Time
}

func (r *playbackStateRow) toPlaybackState() vibe.PlaybackState {
	var currentSongID *string
	if r.CurrentSongID.Valid {
		currentSongID = &r.CurrentSongID.String
	}

	return vibe.PlaybackState{
		RoomID:        r.RoomID,
		CurrentSongID: currentSongID,
		IsPlaying:     r.IsPlaying == 1,
		PositionMs:    r.PositionMs,
		UpdatedAt:     r.UpdatedAt,
		ServerTimeMs:  time.Now().UnixMilli(),
	}
}
