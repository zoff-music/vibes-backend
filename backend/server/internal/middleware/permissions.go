package middleware

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/server/internal/helper"
	"github.com/zoff-music/vibes/vibe"
)

// PermissionMiddleware handles authentication for protected routes
type PermissionMiddleware struct {
	DB              vibe.PermissionProvider
	ProtectedRoutes map[string]bool
}

// Middleware is the actual middleware function
func (m *PermissionMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeName := mux.CurrentRoute(r).GetName()
		if !m.ProtectedRoutes[routeName] {
			next.ServeHTTP(w, r)
			return
		}

		vars := mux.Vars(r)
		roomID := vars["id"]

		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)

		// 1. Check if we have a session. If not, we can't be admin.
		// NOTE: If the room has NO password, we might still want to allow access.
		// But first we need to fetch the room to know if it has a password.

		room, err := m.DB.GetRoom(ctx, roomID)
		if err != nil {
			log.Printf("PermissionMiddleware: failed to fetch room %s: %v", roomID, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if room.IsEmpty() {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}

		// If room has no password, everyone is allowed to edit settings (as per requirements)
		if !room.HasPassword {
			next.ServeHTTP(w, r)
			return
		}

		// Room has password, so we MUST have a valid user session
		if !ok || session.UserID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Fetch the user
		user, err := m.DB.GetUser(ctx, roomID, session.UserID)
		if err != nil {
			log.Printf("PermissionMiddleware: failed to fetch user %s in room %s: %v", session.UserID, roomID, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if user.IsEmpty() {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}

		// Check if user belongs to this room or is not admin
		if user.RoomID != roomID || !user.IsAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		// Allowed
		next.ServeHTTP(w, r)
	})
}
