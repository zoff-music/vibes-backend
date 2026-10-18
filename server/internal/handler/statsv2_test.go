package handler

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http/httptest"
	"testing"

	"github.com/zoff-music/vibes-backend/vibe"
)

type statsCancellationFixture struct {
	cacheErr error
	fetchErr error
	writeErr error
	fetches  int
}

func (f *statsCancellationFixture) GetCachedStatsV2(_ context.Context, _ vibe.RoomType) (*vibe.CachedStatsV2, error) {
	if f.cacheErr != nil {
		return nil, fmt.Errorf("error reading fixture cache: %w", f.cacheErr)
	}
	return &vibe.CachedStatsV2{}, nil
}

func (f *statsCancellationFixture) GetStatsV2(_ context.Context, _ vibe.RoomType) (*vibe.StatsV2, error) {
	f.fetches++
	if f.fetchErr != nil {
		return nil, fmt.Errorf("error reading fixture stats: %w", f.fetchErr)
	}
	return &vibe.StatsV2{}, nil
}

func (f *statsCancellationFixture) CacheStatsV2(_ context.Context, _ vibe.RoomType, _ vibe.StatsV2) error {
	if f.writeErr != nil {
		return fmt.Errorf("error writing fixture cache: %w", f.writeErr)
	}
	return nil
}

func TestStatsCancellation(t *testing.T) {
	tests := []struct {
		name    string
		fixture statsCancellationFixture
		status  int
		fetches int
		logged  bool
	}{
		{name: "cache cancellation stops database fallback", fixture: statsCancellationFixture{cacheErr: context.Canceled}, status: 499},
		{name: "database cancellation", fixture: statsCancellationFixture{fetchErr: context.Canceled}, status: 499, fetches: 1},
		{name: "optional cache write cancellation", fixture: statsCancellationFixture{writeErr: context.Canceled}, status: 200, fetches: 1},
		{name: "cache timeout still falls back", fixture: statsCancellationFixture{cacheErr: context.DeadlineExceeded}, status: 200, fetches: 1, logged: true},
		{name: "database timeout remains visible", fixture: statsCancellationFixture{fetchErr: context.DeadlineExceeded}, status: 500, fetches: 1, logged: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&output)
			t.Cleanup(func() { log.SetOutput(previous) })
			response := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/api/v2/stats", nil)

			GetStatsV2(&tt.fixture, &tt.fixture).ServeHTTP(response, request)

			if response.Code != tt.status || tt.fixture.fetches != tt.fetches {
				t.Fatalf("status/fetches = %d/%d, want %d/%d", response.Code, tt.fixture.fetches, tt.status, tt.fetches)
			}
			if (output.Len() > 0) != tt.logged {
				t.Fatalf("logged = %v, want %v", output.Len() > 0, tt.logged)
			}
		})
	}
}
