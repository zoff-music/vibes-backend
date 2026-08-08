package helper

import (
	"context"

	"github.com/zoff-music/vibes-backend/vibe"
)

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

// GetSessionFromContext extracts the session payload from the context
func GetSessionFromContext(ctx context.Context) (SessionPayload, bool) {
	session, ok := ctx.Value(SessionKey).(SessionPayload)
	return session, ok
}

func GetAdminUserFromContext(ctx context.Context) (*vibe.AdminUser, bool) {
	user, ok := ctx.Value(AdminUserKey).(*vibe.AdminUser)
	return user, ok
}

const SessionKey = "session"

const AdminUserKey = "admin_user"
