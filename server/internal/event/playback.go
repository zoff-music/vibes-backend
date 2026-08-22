package event

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/zoff-music/vibes-backend/vibe"
)

// ReviewRoomPlayback handles playback monitoring for server-mode rooms.
type ReviewRoomPlayback struct {
	DB     vibe.ExpiredPlaybackSongFetcher
	Events vibe.RoomBatchEventNotifier
}

// Handle checks for rooms that need to auto-advance.
func (h *ReviewRoomPlayback) Handle(ctx context.Context, _ []byte) error {
	state, err := h.DB.ProcessNextExpiredPlayback(ctx)
	if err != nil {
		return fmt.Errorf("error processing next expired playback: %w", err)
	}

	statePayload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("error marshaling playback state payload: %w", err)
	}

	songs, err := h.DB.GetSongs(ctx, state.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching songs for room %s: %w", state.RoomID, err)
	}

	songsPayload, err := json.Marshal(songs)
	if err != nil {
		return fmt.Errorf("error marshaling songs payload: %w", err)
	}

	err = h.Events.NotifyRoomUpdates(ctx, state.RoomID, []vibe.RoomEvent{
		{
			Type:    vibe.PlaybackUpdate,
			Payload: statePayload,
		},
		{
			Type:    vibe.QueueReordered,
			Payload: songsPayload,
		},
	})
	if err != nil {
		return fmt.Errorf("error notifying room %s update: %w", state.RoomID, err)
	}

	return nil
}

// ReviewHostHealth handles host health checks.
type ReviewHostHealth struct {
	DB     vibe.AbandonedHostProcessor
	Events vibe.RoomEventNotifier
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

	err = h.Events.NotifyRoomUpdate(ctx, info.RoomID, vibe.RoomEvent{
		Type:    vibe.NewHost,
		Payload: payloadBytes,
	})
	if err != nil {
		return fmt.Errorf("error notifying room %s host update: %w", info.RoomID, err)
	}

	return nil
}
