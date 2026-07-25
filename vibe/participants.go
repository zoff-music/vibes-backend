package vibe

import (
	"context"
	"time"
)

// Participant represents a user in a room
type Participant struct {
	RoomID           string    `json:"roomId"`
	UserID           string    `json:"userId"`
	IsActiveListener bool      `json:"isActiveListener"`
	IsCastReceiver   bool      `json:"isCastReceiver"`
	CastOwnerID      string    `json:"castOwnerId,omitempty"`
	LastSeenAt       time.Time `json:"lastSeenAt"`
}

// ListenerCounts holds active listener counts for a room.
type ListenerCounts struct {
	ActiveListeners     int
	ActiveCastReceivers int
}

// InactiveParticipantDeleter removes inactive room participants.
type InactiveParticipantDeleter interface {
	DeleteInactiveParticipants(ctx context.Context, olderThan time.Duration) (int, error)
}

// RoomEventParticipantStorage defines the exact storage used by the room event handler.
type RoomEventParticipantStorage interface {
	UpdateParticipant(ctx context.Context, roomID, userID string, isActiveListener bool, isCastReceiver bool, castOwnerID string) error
	GetActiveListenerCounts(ctx context.Context, roomID string, activeWithin time.Duration) (ListenerCounts, error)
	GetPlaybackState(ctx context.Context, roomID string) (*PlaybackState, error)
}
