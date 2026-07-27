package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// AdminMiddleware enforces access to authenticated admin users.
type AdminMiddleware struct {
	DB              vibe.AdminUserFetcher
	CookieSecret    string
	ProtectedRoutes map[string]bool
}

// Middleware is the actual middleware function
func (m *AdminMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeName := mux.CurrentRoute(r).GetName()
		if !m.ProtectedRoutes[routeName] {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			log.Printf("AdminMiddleware: missing user session")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		cookie, err := r.Cookie(helper.AdminAuthCookieName)
		if err != nil {
			log.Printf("AdminMiddleware: missing admin cookie")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		payload, err := helper.ParseAdminAuthPayload(cookie.Value, m.CookieSecret)
		if err != nil {
			log.Printf("AdminMiddleware: invalid admin session: %v", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if session.UserID != payload.UserID {
			log.Printf("AdminMiddleware: session/user mismatch")
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		issuedAt := time.Unix(payload.IssuedAt, 0)
		if issuedAt.After(time.Now().Add(adminSessionClockSkew)) ||
			time.Since(issuedAt) > adminSessionDuration {
			log.Printf("AdminMiddleware: expired admin session")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		admin, err := m.DB.GetAdminUser(ctx, payload.AdminID)
		if err != nil {
			log.Printf("AdminMiddleware: error getting admin user: %v", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if admin.IsEmpty() || admin.SessionVersion != payload.SessionVersion {
			log.Printf("AdminMiddleware: invalid admin user session")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx = context.WithValue(ctx, helper.AdminUserKey, admin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

const adminSessionDuration = 24 * time.Hour

const adminSessionClockSkew = time.Minute
