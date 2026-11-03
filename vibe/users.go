package vibe

import (
	"time"
)

// RoomPassword contains only the room identity and its stored credential.
type RoomPassword struct {
	RoomID string
	Hash   string `json:"-"`
}

func (r *RoomPassword) IsEmpty() bool {
	return r.RoomID == ""
}

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
