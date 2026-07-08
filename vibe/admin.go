package vibe

import "context"

type AdminRoomSummary struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	UserCount     int      `json:"userCount"`
	SongCount     int      `json:"songCount"`
	ActiveSources []string `json:"activeSources"`
}

type AdminLoginRequest struct {
	Password string `json:"password"`
}

type AdminSessionResponse struct {
	Authorized bool `json:"authorized"`
}

type AdminEvent struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

type AdminRoomLister interface {
	ListAdminRooms(ctx context.Context) ([]AdminRoomSummary, error)
}

type AdminEventNotifier interface {
	NotifyAdminUpdate(ctx context.Context, event AdminEvent) error
}

type AdminSubscriberPublisher interface {
	Subscriber
	AdminEventNotifier
}

const AdminRoomsUpdate = "admin_rooms_update"
