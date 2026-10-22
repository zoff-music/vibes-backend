package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

type playlistResolverStub struct {
	err error
}

type providerCacheStub struct {
	err error
}

func (s providerCacheStub) CacheProviderItems(_ context.Context, _ string, _ []vibe.ProviderItem) error {
	return s.err
}

func (s playlistResolverStub) ResolveProviderPlaylist(_ context.Context, _ string) (*vibe.ProviderPlaylist, error) {
	if s.err != nil {
		return nil, fmt.Errorf("error resolving test playlist: %w", s.err)
	}

	return &vibe.ProviderPlaylist{ID: "playlist", Items: []vibe.ProviderItem{}}, nil
}

func TestResolveSoundCloudPlaylistV2(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		status   int
		body     string
		cacheErr error
	}{
		{name: "valid playlist", status: http.StatusOK, body: "playlist"},
		{name: "cache failure", cacheErr: fmt.Errorf("error cache unavailable"), status: http.StatusInternalServerError, body: "something went wrong"},
		{name: "not a playlist", err: internalerror.ErrProviderPlaylistInvalid{Err: fmt.Errorf("error wrong resource kind")}, status: http.StatusBadRequest, body: "This SoundCloud link does not point to a playlist"},
		{name: "provider failure", err: fmt.Errorf("error upstream failure with private details"), status: http.StatusInternalServerError, body: "something went wrong"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v2/soundcloud/playlists?url=https://soundcloud.com/example/sets/test", nil)
			response := httptest.NewRecorder()
			ResolveSoundCloudPlaylistV2(playlistResolverStub{err: tt.err}, providerCacheStub{err: tt.cacheErr})(response, request)

			if response.Code != tt.status || !strings.Contains(response.Body.String(), tt.body) {
				t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
			}
			if strings.Contains(response.Body.String(), "private details") {
				t.Fatal("upstream details leaked")
			}
		})
	}
}
