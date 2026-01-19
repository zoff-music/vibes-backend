package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/server/internal/helper"
)

type mockExternalAuthUpserter struct{}

func (m *mockExternalAuthUpserter) UpsertExternalAuth(ctx context.Context, userID, provider, code, state string, expiresAt time.Time) error {
	return nil
}

func TestOAuthCallback_StateValidation(t *testing.T) {
	state := "test-state-value"

	// 1. Set the cookie using the helper (as the authorize endpoint would)
	wCookie := httptest.NewRecorder()
	helper.SetOAuthStateCookie(wCookie, state)

	// Extract the cookie
	resp := wCookie.Result()
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("Helper failed to set cookie")
	}
	authCookie := cookies[0]

	// 2. Prepare the callback request
	req := httptest.NewRequest("GET", "/api/v1/callbacks/spotify?state="+state+"&code=somecode", nil)
	req.AddCookie(authCookie)
	req = mux.SetURLVars(req, map[string]string{"provider": "spotify"})

	w := httptest.NewRecorder()

	// 3. Call the handler
	// We expect 401 Unauthorized because we haven't authenticated the user session,
	// but getting 401 means we PASSED the state validation (which returns 400).
	OAuthCallback(&mockExternalAuthUpserter{}, "http://frontend")(w, req)

	if w.Code == http.StatusBadRequest {
		t.Errorf("Expected to pass state validation, but got 400 Bad Request. Body: %s", w.Body.String())
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized (due to missing session), got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestOAuthCallback_InvalidState(t *testing.T) {
	// Prepare request with MISMATCHING state
	req := httptest.NewRequest("GET", "/api/v1/callbacks/spotify?state=wrong-state&code=somecode", nil)
	// We manually set the cookie to the "correct" name but verify checking logic
	// Actually, let's use the helper to set a cookie with a different value

	state := "original-state"
	wCookie := httptest.NewRecorder()
	helper.SetOAuthStateCookie(wCookie, state)
	authCookie := wCookie.Result().Cookies()[0]

	req.AddCookie(authCookie)
	req = mux.SetURLVars(req, map[string]string{"provider": "spotify"})

	w := httptest.NewRecorder()

	OAuthCallback(&mockExternalAuthUpserter{}, "http://frontend")(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for invalid state, got %d", w.Code)
	}
}
