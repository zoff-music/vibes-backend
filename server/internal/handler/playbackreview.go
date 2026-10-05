package handler

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/zoff-music/vibes-backend/vibe"
)

// ReviewRoomPlayback handles playback monitoring for server-mode rooms.
type ReviewRoomPlayback struct {
	DB     vibe.ExpiredPlaybackPlaylistItemFetcher
	Events vibe.RoomBatchEventV3Notifier
}

// Handle checks for rooms that need to auto-advance.
func (h *ReviewRoomPlayback) Handle(ctx context.Context, _ []byte) error {
	advance, err := h.DB.ProcessNextExpiredPlaybackV2(ctx)
	if err != nil {
		return fmt.Errorf("error processing next expired playback: %w", err)
	}

	statePayload, err := json.Marshal(advance.Playback)
	if err != nil {
		return fmt.Errorf("error marshaling playback state payload: %w", err)
	}

	playlistItems, err := h.DB.GetPlaylistItems(ctx, advance.Playback.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching playlistItems for room %s: %w", advance.Playback.RoomID, err)
	}

	playlistItemsPayload, err := json.Marshal(playlistItems)
	if err != nil {
		return fmt.Errorf("error marshaling playlistItems payload: %w", err)
	}
	var compactEvent *vibe.RoomEventV3Payload
	for position, playlistItem := range playlistItems {
		if playlistItem.ID != advance.PreviousPlaylistItemID {
			continue
		}

		compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemPositionUpdate{
			PlaylistItem: playlistItem,
			Position:     position,
		})
		if marshalErr != nil {
			return fmt.Errorf("error marshaling compact playback advance event: %w", marshalErr)
		}
		compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemUpdated, Payload: compactPayload}
		break
	}
	if compactEvent == nil {
		compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemIDUpdate{ID: advance.PreviousPlaylistItemID})
		if marshalErr != nil {
			return fmt.Errorf("error marshaling compact playback advance removal fallback: %w", marshalErr)
		}
		compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemRemoved, Payload: compactPayload}
	}

	err = h.Events.NotifyRoomUpdatesV3(ctx, advance.Playback.RoomID, []vibe.RoomEventV3{
		{
			Type:    vibe.PlaybackUpdate,
			Payload: statePayload,
		},
		{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: playlistItemsPayload,
			Compact: compactEvent,
		},
	})
	if err != nil {
		return fmt.Errorf("error notifying room %s update: %w", advance.Playback.RoomID, err)
	}

	return nil
}

// ReviewHostHealth handles host health checks.
type ReviewHostHealth struct {
	DB     vibe.AbandonedHostProcessor
	Events vibe.RoomEventV3Notifier
}

// Handle checks for rooms that need a new host.
func (h *ReviewHostHealth) Handle(ctx context.Context, _ []byte) error {
	info, err := h.DB.ProcessNextAbandonedHost(ctx)
	if err != nil {
		return fmt.Errorf("error processing next abandoned host: %w", err)
	}

	payload := vibe.NewHostUpdate{
		UserID:  info.NewHostID,
		Message: "You are now the host",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling new host payload: %w", err)
	}

	err = h.Events.NotifyRoomUpdateV3(ctx, info.RoomID, vibe.RoomEventV3{
		Type:    vibe.NewHost,
		Payload: payloadBytes,
	})
	if err != nil {
		return fmt.Errorf("error notifying room %s host update: %w", info.RoomID, err)
	}

	return nil
}
