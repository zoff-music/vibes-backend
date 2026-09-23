package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/server/internal/middleware"
	_ "github.com/zoff-music/vibes-backend/swaggerdocs"
	"github.com/zoff-music/vibes-backend/vibe"
)

type documentationRouteTest struct {
	name    string
	path    string
	status  int
	content string
}

func TestDocumentationRoutes(t *testing.T) {
	tests := []documentationRouteTest{
		{name: "branded page", path: "/api/swagger/index.html", status: 200, content: "https://zoff.me/logo.png"},
		{name: "directory page", path: "/api/swagger/", status: 200, content: "Great integrations."},
		{name: "public spec", path: "/api/swagger/doc.json", status: 200, content: "/api/v1/rooms"},
		{name: "public spec query", path: "/api/swagger/doc.json?cache=1", status: 200, content: "/api/v1/rooms"},
		{name: "nested spec denied", path: "/api/swagger/nested/doc.json", status: 404},
		{name: "static json denied", path: "/api/swagger/swagger.json", status: 404},
		{name: "static yaml denied", path: "/api/swagger/swagger.yaml", status: 404},
		{name: "unprotected admin denied", path: "/api/swagger/admin.json", status: 404},
		{name: "asset query cannot select full spec", path: "/api/swagger/swagger-ui.css?x=/doc.json", status: 200, content: ".swagger-ui"},
		{name: "theme", path: "/api/swagger/theme.css", status: 200, content: "#120b1e"},
	}
	handler := Documentation()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if response.Code != tt.status {
				t.Fatalf("status %d, want %d", response.Code, tt.status)
			}
			if !strings.Contains(response.Body.String(), tt.content) {
				t.Fatalf("missing %q", tt.content)
			}
			for _, private := range []string{"/api/v1/admin/", "vibe.AdminUser", "vibe.AdminLoginRequest"} {
				if strings.Contains(response.Body.String(), private) {
					t.Fatalf("public response exposes %s", private)
				}
			}
			if response.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatal("response must not be cached")
			}
		})
	}
}

type documentationAdminStub struct {
	missing bool
	failed  bool
	version int64
}

func (s *documentationAdminStub) GetAdminUser(_ context.Context, id string) (*vibe.AdminUser, error) {
	if s.failed {
		return nil, fmt.Errorf("error database unavailable")
	}
	if s.missing {
		return &vibe.AdminUser{}, nil
	}
	return &vibe.AdminUser{ID: id, SessionVersion: s.version}, nil
}

type documentationAdminTest struct {
	name      string
	noAdmin   bool
	noSession bool
	tampered  bool
	mismatch  bool
	age       time.Duration
	missing   bool
	failed    bool
	version   int64
	status    int
}

func TestAdminDocumentationSession(t *testing.T) {
	tests := []documentationAdminTest{
		{name: "valid admin", version: 1, status: 200},
		{name: "public session", noAdmin: true, status: 401},
		{name: "admin without matching session cookie", noSession: true, status: 403},
		{name: "tampered admin cookie", tampered: true, status: 401},
		{name: "different user", mismatch: true, status: 403},
		{name: "expired cookie", age: 25 * time.Hour, status: 401},
		{name: "future cookie", age: -5 * time.Minute, status: 401},
		{name: "deleted admin", missing: true, status: 401},
		{name: "revoked session", version: 2, status: 401},
		{name: "database unavailable", failed: true, status: 401},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := "documentation-test-secret"
			router := mux.NewRouter()
			router.HandleFunc("/api/swagger/admin.json", AdminDocumentation()).Name("AdminDocumentation")
			sessionMiddleware := middleware.SessionMiddleware{Secret: secret, CookieMaxAge: time.Hour}
			adminMiddleware := middleware.AdminMiddleware{DB: &documentationAdminStub{missing: tt.missing, failed: tt.failed, version: tt.version}, CookieSecret: secret, ProtectedRoutes: map[string]bool{"AdminDocumentation": true}}
			router.Use(sessionMiddleware.Middleware, adminMiddleware.Middleware)
			request := httptest.NewRequest(http.MethodGet, "/api/swagger/admin.json", nil)
			if !tt.noSession {
				raw, err := json.Marshal(vibe.SessionPayload{UserID: "user", AuthType: "cookie"})
				if err != nil {
					t.Fatal(err)
				}
				encoded := base64.StdEncoding.EncodeToString(raw)
				mac := hmac.New(sha256.New, []byte(secret))
				mac.Write([]byte(encoded))
				request.AddCookie(&http.Cookie{Name: "session", Value: encoded + "." + base64.StdEncoding.EncodeToString(mac.Sum(nil))})
			}
			if !tt.noAdmin {
				userID := "user"
				if tt.mismatch {
					userID = "other-user"
				}
				cookie, err := helper.SignAdminAuthPayload(vibe.AdminAuthPayload{AdminID: "admin", UserID: userID, SessionVersion: 1, IssuedAt: time.Now().Add(-tt.age).Unix()}, secret)
				if err != nil {
					t.Fatal(err)
				}
				if tt.tampered {
					cookie += "tampered"
				}
				request.AddCookie(&http.Cookie{Name: helper.AdminAuthCookieName, Value: cookie})
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("status %d, want %d", response.Code, tt.status)
			}
			exposed := strings.Contains(response.Body.String(), "/api/v1/admin/")
			if exposed != (tt.status == 200) {
				t.Fatalf("incorrect admin visibility: %t", exposed)
			}
			if response.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatal("admin response must not be cached")
			}
			if !strings.Contains(strings.Join(response.Header().Values("Vary"), ","), "Cookie") {
				t.Fatal("missing cookie cache variation")
			}
		})
	}
}
