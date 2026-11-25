package youtube

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

func TestPlaylistAuthorization(t *testing.T) {
	tests := []struct {
		name, token, reason string
		status              int
		wantAccessError     bool
	}{
		{"public", "", "", 200, false},
		{"private", "", "playlistItemsNotAccessible", 403, true},
		{"hidden or missing", "", "playlistNotFound", 404, true},
		{"quota", "", "quotaExceeded", 403, false},
		{"authenticated", "test-token", "", 200, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wantHeader := ""
				if tt.token != "" {
					wantHeader = "Bearer " + tt.token
				}
				if r.Header.Get("Authorization") != wantHeader {
					t.Error("unexpected authorization header")
				}
				w.WriteHeader(tt.status)
				if tt.reason != "" {
					_ = json.NewEncoder(w).Encode(searchErrorResponse{Error: searchError{Errors: []searchErrorReason{{Reason: tt.reason}}}})
					return
				}
				_ = json.NewEncoder(w).Encode(youtubePlaylistItemsResponse{Items: []youtubePlaylistItem{}})
			}))
			t.Cleanup(upstream.Close)
			c := Client{apiKey: "test-key", Endpoint: upstream.URL, HTTPClient: client.HTTPClient{Client: upstream.Client()}}
			_, err := c.GetProviderPlaylist(t.Context(), "list", vibe.RoomTypeMusic, tt.token)
			var accessError internalerror.ErrPlaylistAccess
			if errors.As(err, &accessError) != tt.wantAccessError || (err == nil) != (tt.status == 200) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
