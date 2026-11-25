package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type playlistOAuthFixture struct {
	vibe.CodeValidatorUpserter // Any call to durable token storage fails the test.
	state                      string
	verifier                   string
	sourceHash                 string
	storedToken                string
	deadline                   time.Time
}

func (f *playlistOAuthFixture) SavePendingOAuthState(_ context.Context, _, state, verifier string) error {
	f.state = state
	f.verifier = verifier
	return nil
}

func (f *playlistOAuthFixture) GetOAuthURL(state, _ string) string {
	return "https://provider.example/authorize?state=" + url.QueryEscape(state)
}

func (f *playlistOAuthFixture) ConsumePendingOAuthState(_ context.Context, user, state string) (*vibe.PendingOAuthState, error) {
	if state != f.state || user != "initiator" {
		return &vibe.PendingOAuthState{}, nil
	}
	f.state = ""
	return &vibe.PendingOAuthState{UserID: user, CodeVerifier: f.verifier}, nil
}

func (f *playlistOAuthFixture) ExchangeCode(_ context.Context, _, _ string) (*vibe.TokenResponse, error) {
	return &vibe.TokenResponse{AccessToken: "provider-access", RefreshToken: "provider-refresh", ExpiresIn: 3600}, nil
}

func (f *playlistOAuthFixture) SavePlaylistAuthorization(_ context.Context, _, sourceHash, accessToken string, deadline time.Time) error {
	f.sourceHash = sourceHash
	f.storedToken = accessToken
	f.deadline = deadline
	return nil
}

func TestPlaylistOAuthCallback(t *testing.T) {
	tests := []struct {
		name, provider, source, callbackUser string
		status                               int
	}{
		{"youtube", "youtube", "PL123", "initiator", http.StatusOK},
		{"soundcloud", "soundcloud", "https://soundcloud.com/artist/sets/music", "initiator", http.StatusOK},
		{"different browser", "youtube", "PL123", "other", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &playlistOAuthFixture{}
			ctx := context.WithValue(t.Context(), helper.SessionKey, vibe.SessionPayload{UserID: "initiator", AuthType: "cookie"})
			start := httptest.NewRequestWithContext(ctx, http.MethodGet, "/?playlist="+url.QueryEscape(tt.source), nil)
			redirect := httptest.NewRecorder()
			Authorize(f, f, tt.provider)(redirect, start)
			if redirect.Code != http.StatusTemporaryRedirect || f.state == "" {
				t.Fatal("authorization did not start")
			}
			state := f.state
			callbackCtx := context.WithValue(t.Context(), helper.SessionKey, vibe.SessionPayload{UserID: tt.callbackUser, AuthType: "cookie"})
			callback := httptest.NewRequestWithContext(callbackCtx, http.MethodGet, "/?code=provider-code&state="+url.QueryEscape(state), nil)
			result := httptest.NewRecorder()
			OAuthCallback(f, f, tt.provider)(result, callback)
			if result.Code != tt.status || strings.Contains(result.Body.String(), "provider-access") || strings.Contains(result.Body.String(), "provider-refresh") {
				t.Fatal("unexpected callback response")
			}
			if tt.status != http.StatusOK {
				if f.storedToken != "" {
					t.Fatal("another session stored a credential")
				}
				return
			}
			if f.sourceHash != playlistHash(tt.provider+"."+tt.source) || f.storedToken != "provider-access" || time.Until(f.deadline) > 5*time.Minute || result.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("invalid import authorization")
			}
			replay := httptest.NewRecorder()
			OAuthCallback(f, f, tt.provider)(replay, callback)
			if replay.Code != http.StatusBadRequest {
				t.Fatal("callback accepted a replay")
			}
		})
	}
}
