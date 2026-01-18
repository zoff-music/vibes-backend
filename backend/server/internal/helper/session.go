package helper

import (
	"context"
)

const SessionKey = "session"

type SessionPayload struct {
	UserID string `json:"user_id"`
}

// GetSessionFromContext extracts the session payload from the context
func GetSessionFromContext(ctx context.Context) (SessionPayload, bool) {
	session, ok := ctx.Value(SessionKey).(SessionPayload)
	return session, ok
}
