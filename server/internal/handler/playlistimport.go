package handler

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/vibe"
)

type ImportPlaylistSong struct {
	DB     vibe.PlaylistImportProcessor
	Events vibe.RoomBatchEventNotifier
}

func (h *ImportPlaylistSong) Handle(ctx context.Context, _ []byte) error {
	playlistImport, err := h.DB.ProcessNextPlaylistImport(
		ctx,
		playlistImportRetryInterval,
	)
	if err != nil {
		return fmt.Errorf("error processing next playlist import in Handle: %w", err)
	}
	if playlistImport.Exhausted {
		err = h.DB.DeletePlaylistImport(ctx, playlistImport.ID)
		if err != nil {
			return fmt.Errorf("error deleting exhausted playlist import in Handle: %w", err)
		}

		return nil
	}

	result, err := h.DB.AddPlaylistSong(ctx, &playlistImport.Song)
	if err != nil {
		return fmt.Errorf("error adding playlist import item in Handle: %w", err)
	}

	var events []vibe.RoomEvent
	if result.Outcome == vibe.AddSongOutcomeAdded {
		songPayload, err := json.Marshal(result.Song)
		if err != nil {
			return fmt.Errorf("error marshaling playlist import song in Handle: %w", err)
		}
		events = append(events, vibe.RoomEvent{
			Type:    vibe.SongAdded,
			Payload: songPayload,
		})
	}
	if playlistImport.NextPosition == 0 && result.Outcome == vibe.AddSongOutcomeAdded {
		playbackState, err := h.DB.StartPlaybackIfIdle(ctx, playlistImport.RoomID)
		if err != nil {
			return fmt.Errorf("error starting playlist import playback in Handle: %w", err)
		}
		playbackPayload, err := json.Marshal(playbackState)
		if err != nil {
			return fmt.Errorf("error marshaling playlist import playback in Handle: %w", err)
		}
		events = append(events, vibe.RoomEvent{
			Type:    vibe.PlaybackUpdate,
			Payload: playbackPayload,
		})
	}

	err = h.DB.CompletePlaylistImportItem(
		ctx,
		playlistImport.ID,
		playlistImport.NextPosition,
	)
	if err != nil {
		return fmt.Errorf("error completing playlist import item in Handle: %w", err)
	}

	if len(events) > 0 {
		err = h.Events.NotifyRoomUpdates(ctx, playlistImport.RoomID, events)
		if err != nil {
			return fmt.Errorf("error notifying playlist import item in Handle: %w", err)
		}
	}

	return nil
}

const playlistImportRetryInterval = 5 * time.Minute
