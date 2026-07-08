package helper

import (
	"context"
)

type SessionPayload struct {
	UserID string `json:"user_id"`
	// AuthType indicates how this session was authenticated.
	// Values: "cookie" | "cast"
	AuthType string `json:"auth_type"`
	// CastRoomID is set only for AuthType=="cast" and is used to prevent a cast
	// token for room A from being used against room B endpoints.
	CastRoomID string `json:"cast_room_id,omitempty"`
}

// GetSessionFromContext extracts the session payload from the context
func GetSessionFromContext(ctx context.Context) (SessionPayload, bool) {
	session, ok := ctx.Value(SessionKey).(SessionPayload)
	return session, ok
}

const SessionKey = "session"
