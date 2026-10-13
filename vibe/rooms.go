package vibe

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
)

const RoomNameMaxLength = 100

// RoomNameReservation is a temporarily reserved room name.
type RoomNameReservation struct {
	Name      string    `json:"name"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// RoomNameReservationRequest requests a custom or generated room name.
type RoomNameReservationRequest struct {
	Name string `json:"name,omitempty" maxLength:"100"`
}

func (r RoomNameReservationRequest) Validate() bool {
	if r.Name == "" {
		return true
	}

	length := utf8.RuneCountInString(strings.TrimSpace(r.Name))

	return length > 0 && length <= RoomNameMaxLength
}

// RoomHostInfo holds info about a host update
type RoomHostInfo struct {
	RoomID    string
	NewHostID string
}

type NewHostUpdate struct {
	UserID  string `json:"userId"`
	Message string `json:"message"`
}

// PublicRoomSearch selects a zero-based, inclusive range of public rooms.
type PublicRoomSearch struct {
	Query    string
	RoomType RoomType
	Live     bool
	From     int
	To       int
}

// RoomNameReserver reserves custom or generated room names.
type RoomNameReserver interface {
	ReserveRoomName(
		ctx context.Context,
		name string,
		ownerID string,
	) (*RoomNameReservation, error)
	RoomNameSuggester
}

// RoomNameSuggester reserves a generated room name.
type RoomNameSuggester interface {
	ReserveSuggestedRoomName(
		ctx context.Context,
		ownerID string,
	) (*RoomNameReservation, error)
}

// RoomNameReservationDeleter deletes expired room name reservations.
type RoomNameReservationDeleter interface {
	DeleteExpiredRoomNameReservations(ctx context.Context) (int, error)
}

// RoomExistenceChecker checks whether a room ID is already in use.
type RoomExistenceChecker interface {
	RoomExists(ctx context.Context, roomID string) (bool, error)
}

// RoomModeServer is the mode where the server controls playback
const RoomModeServer = "server"

// RoomModeHost is the mode where a host controls playback
const RoomModeHost = "host"
