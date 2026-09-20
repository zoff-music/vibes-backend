package helper

import (
	"context"

	"github.com/zoff-music/vibes-backend/vibe"
)

// GetSessionFromContext extracts the session payload from the context
func GetSessionFromContext(ctx context.Context) (vibe.SessionPayload, bool) {
	session, ok := ctx.Value(SessionKey).(vibe.SessionPayload)
	return session, ok
}

func GetAdminUserFromContext(ctx context.Context) (*vibe.AdminUser, bool) {
	user, ok := ctx.Value(AdminUserKey).(*vibe.AdminUser)
	return user, ok
}

const SessionKey = "session"

const AdminUserKey = "admin_user"
