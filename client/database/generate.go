package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareCreateGeneratedRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH created_room_q AS (
			INSERT INTO rooms (id, name, mode, host_id, created_at)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		),
		created_settings_q AS (
			INSERT INTO room_settings (
				room_id,
				skip_allowed,
				democratic_skip,
				skip_vote_threshold,
				max_continuous_adds,
				remove_on_play,
				loop_queue,
				allow_duplicates,
				enabled_sources,
				only_admin_add_songs
			)
			SELECT id, $6, $7, $8, $9, $10, $11, $12, $13, $14
			FROM created_room_q
		)
		SELECT id FROM created_room_q
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateGeneratedRoomStatement: %w", err)
	}

	c.CreateGeneratedRoomStatement = stmt
	return nil
}

func (c *Client) CreateGeneratedRoom(
	ctx context.Context,
	room vibe.Room,
	playlist vibe.GeneratedPlaylist,
) (*vibe.GeneratedRoom, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateGeneratedRoom")
	defer span.End()

	if len(playlist) == 0 {
		return nil, fmt.Errorf("error generated playlist has no youtube songs")
	}

	createdRoom, err := c.createGeneratedRoom(ctx, room)
	if err != nil {
		return nil, fmt.Errorf("error creating generated room: %w", err)
	}

	err = c.addGeneratedSongs(ctx, *createdRoom, playlist)
	if err != nil {
		return nil, fmt.Errorf("error adding generated songs: %w", err)
	}

	return &vibe.GeneratedRoom{
		Room:   *createdRoom,
		Tracks: playlist,
	}, nil
}

func (c *Client) createGeneratedRoom(
	ctx context.Context,
	room vibe.Room,
) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "createGeneratedRoom")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.CreateGeneratedRoomStatement.QueryRowContext(
		cctx,
		room.ID,
		room.Name,
		room.Mode,
		room.HostID,
		room.CreatedAt,
		room.Settings.SkipAllowed,
		room.Settings.DemocraticSkip,
		room.Settings.SkipVoteThreshold,
		room.Settings.MaxContinuousAdds,
		room.Settings.RemoveOnPlay,
		room.Settings.LoopQueue,
		room.Settings.AllowDuplicates,
		strings.Join(room.Settings.EnabledSources, ","),
		room.Settings.OnlyAdminAddSongs,
	)

	var scanned createRoomRow
	err := scanned.scan(row)
	if err != nil {
		return nil, fmt.Errorf("error creating generated room row: %w", err)
	}

	return &room, nil
}

func (c *Client) addGeneratedSongs(
	ctx context.Context,
	room vibe.Room,
	playlist vibe.GeneratedPlaylist,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "addGeneratedSongs")
	defer span.End()

	playbackCreated := false
	for _, track := range playlist {
		song := &vibe.Song{
			ID:           uuid.New().String(),
			RoomID:       room.ID,
			SourceType:   vibe.SourceTypeYouTube,
			SourceID:     track.YouTubeID,
			Title:        track.Title,
			Artist:       track.Artist,
			ThumbnailURL: track.ThumbnailURL,
			Duration:     track.Duration,
			AddedBy:      room.HostID,
			AddedAt:      time.Now(),
		}

		result, err := c.AddSong(ctx, song)
		if err != nil {
			return fmt.Errorf("error adding generated song: %w", err)
		}
		if playbackCreated || result.Outcome != vibe.AddSongOutcomeAdded {
			continue
		}

		state := &vibe.PlaybackState{
			RoomID:       room.ID,
			CurrentSong:  &result.Song,
			IsPlaying:    true,
			PositionMs:   0,
			UpdatedAt:    time.Now(),
			ServerTimeMs: int(time.Now().UnixMilli()),
		}
		err = c.UpsertPlaybackState(ctx, state)
		if err != nil {
			return fmt.Errorf("error starting generated room playback: %w", err)
		}
		playbackCreated = true
	}

	return nil
}
