package middleware

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

func TestMissingAdminCredentials(t *testing.T) {
	tests := []struct {
		name   string
		route  string
		user   string
		logged bool
	}{
		{name: "session probe without user", route: "AdminSession"},
		{name: "session probe without cookie", route: "AdminSession", user: "listener"},
		{name: "protected route without user", route: "AdminUsers", logged: true},
		{name: "protected route without cookie", route: "AdminUsers", user: "listener", logged: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&output)
			t.Cleanup(func() { log.SetOutput(previous) })
			middleware := AdminMiddleware{ProtectedRoutes: map[string]bool{tt.route: true}}
			router := mux.NewRouter()
			router.HandleFunc("/test", func(_ http.ResponseWriter, _ *http.Request) { t.Fatal("unauthorized request reached handler") }).Name(tt.route)
			router.Use(middleware.Middleware)
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := context.WithValue(request.Context(), helper.SessionKey, vibe.SessionPayload{UserID: tt.user})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request.WithContext(ctx))
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d", response.Code)
			}
			if (output.Len() > 0) != tt.logged {
				t.Fatalf("unexpected log: %s", output.String())
			}
			if tt.logged && !strings.Contains(output.String(), "route=\""+tt.route+"\"") {
				t.Fatal("missing route name")
			}
		})
	}
}
