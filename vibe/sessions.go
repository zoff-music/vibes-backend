package vibe

import (
	"context"
	"strings"
	"unicode/utf8"
)

type SessionProfile struct {
	Name string `json:"name"`
}

type UpdateSessionProfileRequest struct {
	Name string `json:"name"`
}

func (r UpdateSessionProfileRequest) Validate() bool {
	name := strings.TrimSpace(r.Name)
	length := utf8.RuneCountInString(name)
	return length >= minimumSessionNameLength && length <= maximumSessionNameLength
}

type SessionProfileFetcherCreator interface {
	GetOrCreateSessionProfile(ctx context.Context, id string) (*SessionProfile, error)
}

type SessionProfileUpdater interface {
	UpdateSessionProfile(ctx context.Context, id string, name string) (*SessionProfile, error)
}

// CreateSessionRequest is the request payload for creating a session.
type CreateSessionRequest struct {
	Nickname string `json:"nickname,omitempty"`
	Password string `json:"password,omitempty"`
}

// SessionResponse is returned when creating a session
type SessionResponse struct {
	UserID    string  `json:"userId"`
	SessionID string  `json:"sessionId"`
	Nickname  *string `json:"nickname,omitempty"`
	IsAdmin   bool    `json:"isAdmin"`
	Room      *Room   `json:"room"`
}

// AdminAuthResult represents the result of an admin authentication attempt
type AdminAuthResult struct {
	IsAdmin          bool
	IsFirstTimeSetup bool
}

// AdminSessionCreator authenticates an admin and fetches its room.
type AdminSessionCreator interface {
	GetRoom(ctx context.Context, id string, userID string) (*Room, error)
	AuthenticateAdmin(ctx context.Context, roomID, userID, password string) (*AdminAuthResult, error)
}

// RoomAdminSessionDeleter removes room-scoped admin access and fetches the room.
type RoomAdminSessionDeleter interface {
	ClearRoomAdmin(ctx context.Context, roomID string, userID string) error
	GetRoom(ctx context.Context, id string, userID string) (*Room, error)
}

const minimumSessionNameLength = 1

const maximumSessionNameLength = 30
