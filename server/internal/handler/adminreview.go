package handler

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/zoff-music/vibes-backend/vibe"
)

// ReviewAdminRooms handles scheduled admin room updates.
type ReviewAdminRooms struct {
	DB     vibe.AdminRoomV2Lister
	Events vibe.AdminEventV2Notifier
}

// Handle fetches admin rooms and broadcasts the update.
func (h *ReviewAdminRooms) Handle(ctx context.Context, _ []byte) error {
	rooms, err := h.DB.ListAdminRoomsV2(ctx)
	if err != nil {
		return fmt.Errorf("error listing admin rooms: %w", err)
	}

	payload, err := json.Marshal(rooms)
	if err != nil {
		return fmt.Errorf("error marshaling admin rooms: %w", err)
	}

	err = h.Events.NotifyAdminUpdateV2(ctx, vibe.AdminEventV2{
		Type:    vibe.AdminRoomsUpdate,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("error notifying admin rooms update: %w", err)
	}

	return nil
}
