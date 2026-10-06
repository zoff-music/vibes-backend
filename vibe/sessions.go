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
	Name string `json:"name" minLength:"1" maxLength:"30"`
}

func (r UpdateSessionProfileRequest) Validate() bool {
	name := strings.TrimSpace(r.Name)
	length := utf8.RuneCountInString(name)
	return length >= minimumSessionNameLength && length <= SessionNameMaxLength
}

type SessionProfileFetcherCreator interface {
	GetOrCreateSessionProfile(ctx context.Context, id string) (*SessionProfile, error)
}

type SessionProfileUpdater interface {
	UpdateSessionProfile(ctx context.Context, id string, name string) (*SessionProfile, error)
}

type SessionProfileRoomUpdater interface {
	SessionProfileUpdater
	SessionProfileFetcherCreator
	GetSessionRooms(ctx context.Context, userID string) ([]SessionRoom, error)
}

type SessionRoom struct {
	ID      string
	IsAdmin bool
	IsHost  bool
}

// CreateSessionRequest is the request payload for creating a session.
type CreateSessionRequest struct {
	Nickname string `json:"nickname,omitempty"`
	Password string `json:"password,omitempty"`
}

// SessionResponse is returned when creating a session
//
// Deprecated: Use SessionResponseV2 for new clients. Retained for v1 room authentication.
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

const minimumSessionNameLength = 1

const SessionNameMaxLength = 30

type SessionPayload struct {
	UserID string `json:"user_id"`
	IsNew  bool   `json:"-"`
	// AuthType indicates how this session was authenticated.
	// Values: "cookie" | "cast" | "remote"
	AuthType string `json:"auth_type"`
	// CastRoomID is set only for AuthType=="cast" and is used to prevent a cast
	// token for room A from being used against room B endpoints.
	CastRoomID   string `json:"cast_room_id,omitempty"`
	RemoteID     string `json:"remote_id,omitempty"`
	RemoteRoomID string `json:"remote_room_id,omitempty"`
	EventOrigin  string `json:"-"`
}
