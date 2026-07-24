package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

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

	createdRoom, err := c.CreateRoom(ctx, &room)
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
