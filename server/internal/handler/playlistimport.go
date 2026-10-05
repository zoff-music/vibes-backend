package handler

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/vibe"
)

type ImportPlaylistItem struct {
	DB     vibe.PlaylistImportProcessor
	Events vibe.RoomBatchEventV3Notifier
}

func (h *ImportPlaylistItem) Handle(ctx context.Context, _ []byte) error {
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

	result, err := h.DB.AddImportedPlaylistItem(ctx, &playlistImport.PlaylistItem)
	if err != nil {
		return fmt.Errorf("error adding playlist import item in Handle: %w", err)
	}

	var events []vibe.RoomEventV3

	if result.Outcome == vibe.AddPlaylistItemOutcomeAdded {
		playlistItemPayload, err := json.Marshal(result.PlaylistItem)
		if err != nil {
			return fmt.Errorf("error marshaling playlist import item in Handle: %w", err)
		}

		events = append(events, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemAdded,
			Payload: playlistItemPayload,
		})
	}

	if result.Outcome == vibe.AddPlaylistItemOutcomeAdded || playlistImport.Attempts > 1 {
		playbackState, err := h.DB.StartPlaylistPlayback(ctx, playlistImport.RoomID)
		if err != nil {
			return fmt.Errorf("error starting playlist import playback in Handle: %w", err)
		}

		if playbackState.CurrentPlaylistItem != nil {
			playbackPayload, err := json.Marshal(playbackState)
			if err != nil {
				return fmt.Errorf("error marshaling playlist import playback in Handle: %w", err)
			}

			events = append(events, vibe.RoomEventV3{
				Type:    vibe.PlaybackUpdate,
				Payload: playbackPayload,
			})
		}
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
		err = h.Events.NotifyRoomUpdatesV3(ctx, playlistImport.RoomID, events)
		if err != nil {
			return fmt.Errorf("error notifying playlist import item in Handle: %w", err)
		}
	}

	return nil
}

const playlistImportRetryInterval = 5 * time.Minute
