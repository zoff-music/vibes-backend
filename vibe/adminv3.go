package vibe

import (
	"context"
	"time"
)

type AdminRoomChange struct {
	ID        string    `json:"id"`
	RecheckAt time.Time `json:"recheckAt,omitempty"`
}

type AdminRoomChangeNotifier interface {
	NotifyAdminRoomChange(ctx context.Context, roomID string) error
}

type AdminRoomDepartureNotifier interface {
	NotifyAdminRoomDeparture(ctx context.Context, roomID string) error
}

type AdminRoomSummaryFetcher interface {
	GetAdminRoomSummary(ctx context.Context, roomID string) (*AdminRoomSummaryV2, error)
}

type AdminRoomSummaryListerUserFetcher interface {
	AdminRoomV2ListerUserFetcher
	AdminRoomSummaryFetcher
}

type AdminEventChangeNotifier interface {
	AdminEventV2Notifier
	AdminRoomChangeNotifier
}

func (r *AdminRoomSummaryV2) IsEmpty() bool {
	return r.ID == ""
}

const AdminRoomChangesTopic = "admin:rooms"

const AdminRoomUpdate = "admin_room_update"

const AdminRoomRemoved = "admin_room_removed"
