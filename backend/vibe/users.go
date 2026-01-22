package vibe

import (
	"time"
)

// User represents a user session in a room
type User struct {
	ID         string    `json:"id"`
	RoomID     string    `json:"-"`
	IsAdmin    bool      `json:"isAdmin"`
	JoinedAt   time.Time `json:"joinedAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

// IsEmpty returns true if the user is empty/not found
func (u *User) IsEmpty() bool {
	return u.ID == ""
}
