package vibe

import (
	"context"
	"time"
)

// User represents a user session in a room
type User struct {
	ID         string    `json:"id"`
	RoomID     string    `json:"-"`
	Nickname   *string   `json:"nickname,omitempty"`
	IsAdmin    bool      `json:"isAdmin"`
	JoinedAt   time.Time `json:"joinedAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

// IsEmpty returns true if the user is empty/not found
func (u *User) IsEmpty() bool {
	return u.ID == ""
}

// UserFetcher fetches user data
type UserFetcher interface {
	GetUser(ctx context.Context, roomID, userID string) (*User, error)
	GetUsersInRoom(ctx context.Context, roomID string) ([]User, error)
	CountUsersInRoom(ctx context.Context, roomID string) (int, error)
}

// UserManager manages users in rooms
type UserManager interface {
	CreateUser(ctx context.Context, user *User) (*User, error)
	UpdateUserLastSeen(ctx context.Context, roomID, userID string) error
	RemoveUser(ctx context.Context, roomID, userID string) error
	CleanupInactiveUsers(ctx context.Context, roomID string, threshold time.Duration) error
}
