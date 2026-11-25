package soundcloud

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
)

func TestPlaylistAccess(t *testing.T) {
	tests := []struct {
		name, token, wantToken string
		status                 int
		wantAccessError        bool
	}{
		{"public", "", "app-token", 200, false},
		{"private", "", "app-token", 404, true},
		{"forbidden", "", "app-token", 403, true},
		{"quota", "", "app-token", 429, false},
		{"authorized", "user-token", "user-token", 200, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "OAuth "+tt.wantToken {
					t.Error("incorrect provider credentials")
				}
				if r.URL.Path == "/resolve" {
					w.WriteHeader(tt.status)
					if tt.status == 200 {
						_ = json.NewEncoder(w).Encode(soundCloudPlaylistResponse{URN: "soundcloud:playlists:1", Kind: "playlist"})
					}
					return
				}
				_ = json.NewEncoder(w).Encode(soundCloudPlaylistTracksResponse{Collection: []trackResponse{}})
			}))
			t.Cleanup(upstream.Close)
			c := Client{Enabled: true, Endpoint: upstream.URL, HTTPClient: client.HTTPClient{Client: upstream.Client()}, accessToken: "app-token", tokenExpiresAt: time.Now().Add(time.Hour)}
			_, err := c.ResolveProviderPlaylist(t.Context(), "https://soundcloud.com/user/sets/list", tt.token)
			var accessError internalerror.ErrPlaylistAccess
			if errors.As(err, &accessError) != tt.wantAccessError || (err == nil) != (tt.status == 200) {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.accessToken != "app-token" {
				t.Fatal("import changed the shared provider token")
			}
		})
	}
}
