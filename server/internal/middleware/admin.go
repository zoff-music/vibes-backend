package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// AdminMiddleware enforces admin access on protected routes and verifies
// optional moderator identity on room activity routes without granting access.
type AdminMiddleware struct {
	DB              vibe.AdminUserFetcher
	CookieSecret    string
	ProtectedRoutes map[string]bool
	OptionalRoutes  map[string]bool
}

// Middleware is the actual middleware function
func (m *AdminMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeName := mux.CurrentRoute(r).GetName()
		required := m.ProtectedRoutes[routeName]
		if !required && !m.OptionalRoutes[routeName] {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Add("Vary", "Cookie")

		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			if !required {
				next.ServeHTTP(w, r)
				return
			}

			if routeName != "AdminSession" {
				log.Printf("AdminMiddleware: unauthorized route=%s reason=missing_user_session", strconv.Quote(routeName))
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		cookie, err := r.Cookie(helper.AdminAuthCookieName)
		if err != nil {
			if !required {
				next.ServeHTTP(w, r)
				return
			}

			if routeName != "AdminSession" {
				log.Printf("AdminMiddleware: unauthorized route=%s reason=missing_admin_cookie", strconv.Quote(routeName))
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		payload, err := helper.ParseAdminAuthPayload(cookie.Value, m.CookieSecret)
		if err != nil {
			if !required {
				next.ServeHTTP(w, r)
				return
			}

			log.Printf("AdminMiddleware: invalid admin session: %v", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if session.UserID != payload.UserID {
			if !required {
				next.ServeHTTP(w, r)
				return
			}

			log.Printf("AdminMiddleware: session/user mismatch")
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		issuedAt := time.Unix(payload.IssuedAt, 0)
		if issuedAt.After(time.Now().Add(adminSessionClockSkew)) ||
			time.Since(issuedAt) > adminSessionDuration {
			if !required {
				next.ServeHTTP(w, r)
				return
			}

			log.Printf("AdminMiddleware: expired admin session")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		admin, err := m.DB.GetAdminUser(ctx, payload.AdminID)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				w.WriteHeader(statusClientClosedRequest)
				return
			}
			log.Printf("AdminMiddleware: error getting admin user: %v", err)

			if !required {
				next.ServeHTTP(w, r)
				return
			}

			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if admin.IsEmpty() || admin.SessionVersion != payload.SessionVersion {
			if !required {
				next.ServeHTTP(w, r)
				return
			}

			log.Printf("AdminMiddleware: invalid admin user session")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		cctx, cancel := context.WithDeadline(ctx, issuedAt.Add(adminSessionDuration))
		defer cancel()

		ctx = context.WithValue(cctx, helper.AdminUserKey, admin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

const adminSessionDuration = 24 * time.Hour

const adminSessionClockSkew = time.Minute
