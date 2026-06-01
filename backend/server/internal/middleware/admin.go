package middleware

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/server/internal/helper"
)

// AdminMiddleware enforces admin session access based on the global admin password.
type AdminMiddleware struct {
	AdminPassword   *string
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

		password := ""
		if m.AdminPassword != nil {
			password = *m.AdminPassword
		}

		if password == "" {
			log.Printf("AdminMiddleware: admin password not configured")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if session.UserID != payload.UserID {
			log.Printf("AdminMiddleware: session/user mismatch")
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		expectedHash := helper.HashAdminPassword(password)
		if payload.PasswordHash != expectedHash {
			log.Printf("AdminMiddleware: admin password hash mismatch")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
