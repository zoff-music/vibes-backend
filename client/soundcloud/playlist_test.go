package soundcloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
)

func TestResolveProviderPlaylist(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		urn     string
		invalid bool
		failed  bool
	}{
		{name: "track link", kind: "track", urn: "soundcloud:tracks:1", invalid: true, failed: true},
		{name: "user link", kind: "user", urn: "soundcloud:users:1", invalid: true, failed: true},
		{name: "missing playlist identifier", kind: "playlist", failed: true},
		{name: "malformed provider response", failed: true},
		{name: "playlist", kind: "playlist", urn: "soundcloud:playlists:1"},
		{name: "system playlist", kind: "system-playlist", urn: "soundcloud:system-playlists:1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				if r.URL.Path == "/resolve" {
					err := json.NewEncoder(w).Encode(soundCloudPlaylistResponse{Kind: tt.kind, URN: tt.urn, Title: "Playlist"})
					if err != nil {
						t.Errorf("error encoding resolved response: %v", err)
					}
					return
				}

				err := json.NewEncoder(w).Encode(soundCloudPlaylistTracksResponse{Collection: []trackResponse{}})
				if err != nil {
					t.Errorf("error encoding playlist response: %v", err)
				}
			}))
			defer upstream.Close()

			resolver := Client{
				Enabled:        true,
				Endpoint:       upstream.URL,
				HTTPClient:     client.HTTPClient{Client: upstream.Client()},
				accessToken:    "test-token",
				tokenExpiresAt: time.Now().Add(time.Hour),
			}

			playlist, err := resolver.ResolveProviderPlaylist(context.Background(), "https://soundcloud.com/example/sets/test")
			var invalid internalerror.ErrProviderPlaylistInvalid
			if errors.As(err, &invalid) != tt.invalid || (err != nil) != tt.failed {
				t.Fatalf("unexpected error classification: %v", err)
			}
			if !tt.failed && (playlist == nil || playlist.ID != tt.urn) {
				t.Fatalf("unexpected playlist: %v", playlist)
			}
		})
	}
}
