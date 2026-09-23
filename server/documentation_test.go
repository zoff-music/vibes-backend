package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/config"
)

type documentationRoutingTest struct {
	name    string
	enabled bool
	path    string
	status  int
}

func TestDocumentationRouting(t *testing.T) {
	tests := []documentationRoutingTest{
		{name: "public spec with admin enabled", enabled: true, path: "/api/swagger/doc.json", status: 200},
		{name: "public spec with admin disabled", path: "/api/swagger/doc.json", status: 200},
		{name: "admin spec requires authentication", enabled: true, path: "/api/swagger/admin.json", status: 401},
		{name: "admin spec disabled", path: "/api/swagger/admin.json", status: 404},
		{name: "nested spec blocked", enabled: true, path: "/api/swagger/nested/doc.json", status: 404},
		{name: "asset query blocked", enabled: true, path: "/api/swagger/swagger-ui.css?x=/doc.json", status: 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{CookieSecret: "documentation-test-secret"}
			if tt.enabled {
				cfg.AdminPasswordPepper = "documentation-test-pepper"
			}
			server := Server{Config: cfg, Router: mux.NewRouter()}
			server.setupRoutes()
			response := httptest.NewRecorder()
			server.Router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if response.Code != tt.status {
				t.Fatalf("status %d, want %d", response.Code, tt.status)
			}
			if strings.Contains(response.Body.String(), "/api/v1/admin/") {
				t.Fatal("anonymous request exposed admin spec")
			}
		})
	}
}
