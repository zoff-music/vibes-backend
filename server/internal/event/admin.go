package event

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/zoff-music/vibes-backend/vibe"
)

// ReviewAdminRooms handles scheduled admin room updates.
type ReviewAdminRooms struct {
	DB     vibe.AdminRoomLister
	Events vibe.AdminEventNotifier
}

// Handle fetches admin rooms and broadcasts the update.
func (h *ReviewAdminRooms) Handle(ctx context.Context, _ []byte) error {
	rooms, err := h.DB.ListAdminRooms(ctx)
	if err != nil {
		return fmt.Errorf("error listing admin rooms: %w", err)
	}

	payload, err := json.Marshal(rooms)
	if err != nil {
		return fmt.Errorf("error marshaling admin rooms: %w", err)
	}

	err = h.Events.NotifyAdminUpdate(ctx, vibe.AdminEvent{
		Type:    vibe.AdminRoomsUpdate,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("error notifying admin rooms update: %w", err)
	}

	return nil
}
