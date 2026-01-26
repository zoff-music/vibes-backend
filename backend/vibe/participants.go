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

// ParticipantStorage defines methods for managing room participants
type ParticipantStorage interface {
	// UpdateParticipant updates the last seen time for a participant
	UpdateParticipant(ctx context.Context, roomID, userID string, isActiveListener bool, isCastReceiver bool, castOwnerID string) error
	// GetActiveParticipants returns a list of participants active in the room within the duration
	GetActiveParticipants(ctx context.Context, roomID string, activeWithin time.Duration) ([]Participant, error)
	// GetActiveListenerCounts returns listener and cast receiver counts within the duration
	GetActiveListenerCounts(ctx context.Context, roomID string, activeWithin time.Duration) (ListenerCounts, error)
	// SetRoomHost updates the host for a room
	SetRoomHost(ctx context.Context, roomID, userID string) error
	// RemoveParticipant removes a participant from a room
	RemoveParticipant(ctx context.Context, roomID, userID string) error
	// DeleteInactiveParticipants removes participants who haven't been seen within the duration
	DeleteInactiveParticipants(ctx context.Context, olderThan time.Duration) (int, error)
}

// RoomEventParticipantStorage defines the requirements for the SSE event handler
type RoomEventParticipantStorage interface {
	PlaybackFetcher
	ParticipantStorage
}
