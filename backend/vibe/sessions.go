package vibe

import "context"

// CreateSessionRequest is the request payload for creating a session.
type CreateSessionRequest struct {
	Nickname string `json:"nickname,omitempty"`
	Password string `json:"password,omitempty"`
}

// SessionResponse is returned when creating a session
type SessionResponse struct {
	UserID   string  `json:"userId"`
	Nickname *string `json:"nickname,omitempty"`
	IsAdmin  bool    `json:"isAdmin"`
	Room     *Room   `json:"room"`
}

// SessionCreator creates sessions for rooms.
type SessionCreator interface {
	GetRoom(ctx context.Context, id string, userID string) (*Room, error)
	CreateUser(ctx context.Context, user *User) (*User, error)
}

// SessionCreatorGetter creates sessions for rooms and can get rooms and users.
type SessionCreatorGetter interface {
	GetRoom(ctx context.Context, id string, userID string) (*Room, error)
	UpdateRoom(ctx context.Context, room *Room) (*Room, error)
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetUser(ctx context.Context, roomID, userID string) (*User, error)
	AuthenticateAdmin(ctx context.Context, roomID, userID, password string) (bool, bool, error)
}
