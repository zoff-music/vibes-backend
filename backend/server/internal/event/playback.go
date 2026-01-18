package event

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/zoff-music/vibes/vibe"
)

// PlaybackMonitor handles playback monitoring for server-mode rooms
type PlaybackMonitor struct {
	db  vibe.ExpiredPlaybackProcessor
	ips vibe.RoomEventNotifier
}

// Handle checks for rooms that need to auto-advance
func (h *PlaybackMonitor) Handle(ctx context.Context, data []byte) error {
	state, err := h.db.ProcessNextExpiredPlayback(ctx)
	if err != nil {
		return fmt.Errorf("error processing next expired playback: %w", err)
	}

	err = h.ips.NotifyRoomUpdates(ctx, state.RoomID, []*vibe.RoomEvent{
		{
			Type:    vibe.EventTypePlaybackUpdate,
			Payload: state,
		},
	})
	if err != nil {
		log.Errorf("error notifying room %s update: %v", state.RoomID, err)
	}

	return nil
}

// HostMonitor handles host health checks
type HostMonitor struct {
	db  vibe.AbandonnedHostProcessor
	ips vibe.RoomEventNotifier
}

// Handle checks for rooms that need a new host
func (h *HostMonitor) Handle(ctx context.Context, data []byte) error {
	info, err := h.db.ProcessNextAbandonedHost(ctx)
	if err != nil {
		return fmt.Errorf("error processing next abandoned host: %w", err)
	}

	err = h.ips.NotifyRoom(ctx, info.RoomID, &vibe.RoomEvent{
		Type:    vibe.EventTypeNewHost,
		Payload: map[string]string{"userId": info.NewHostID, "message": "You are now the host"},
	})
	if err != nil {
		log.Errorf("error notifying room %s host update: %v", info.RoomID, err)
	}

	return nil
}
