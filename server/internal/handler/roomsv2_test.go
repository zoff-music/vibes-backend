package handler

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zoff-music/vibes-backend/vibe"
)

type publicRoomsSearchStub struct {
	search vibe.PublicRoomSearch
	called bool
	fail   bool
}

func (s *publicRoomsSearchStub) SearchPublicRooms(
	_ context.Context,
	search vibe.PublicRoomSearch,
) (*vibe.PublicRoomResult, error) {
	s.search = search
	s.called = true
	if s.fail {
		return nil, fmt.Errorf("error unavailable database")
	}
	return &vibe.PublicRoomResult{
		Rooms: []vibe.PublicRoom{{ID: "electro", Name: "electro", SongCount: 12}},
		From:  search.From,
		To:    search.From,
		Total: 23,
		Count: 1,
	}, nil
}

func TestGetPublicRoomsV2(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		status int
		search vibe.PublicRoomSearch
		fail   bool
	}{
		{name: "defaults to ten public rooms", status: 200, search: vibe.PublicRoomSearch{To: 9}},
		{name: "live filter and trimmed search", query: "?live=true&q=%20Electro%20&from=10&to=19", status: 200, search: vibe.PublicRoomSearch{Query: "Electro", Live: true, From: 10, To: 19}},
		{name: "all rooms and one row", query: "?live=false&from=2&to=2", status: 200, search: vibe.PublicRoomSearch{From: 2, To: 2}},
		{name: "default page after an offset", query: "?from=10", status: 200, search: vibe.PublicRoomSearch{From: 10, To: 19}},
		{name: "maximum page", query: "?from=20&to=119", status: 200, search: vibe.PublicRoomSearch{From: 20, To: 119}},
		{name: "negative offset", query: "?from=-1", status: 400},
		{name: "invalid offset", query: "?from=1.5", status: 400},
		{name: "overflowing offset", query: "?from=9223372036854775807", status: 400},
		{name: "invalid end", query: "?to=no", status: 400},
		{name: "reversed range", query: "?from=10&to=9", status: 400},
		{name: "oversized page", query: "?to=100", status: 400},
		{name: "invalid filter", query: "?live=yes", status: 400},
		{name: "oversized search", query: "?q=" + strings.Repeat("a", 101), status: 400},
		{name: "invalid UTF-8 search", query: "?q=%FF", status: 400},
		{name: "null search character", query: "?q=%00", status: 400},
		{name: "database error", status: 500, fail: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &publicRoomsSearchStub{fail: tt.fail}
			request := httptest.NewRequest(http.MethodGet, "/api/v2/rooms/public"+tt.query, nil)
			response := httptest.NewRecorder()
			GetPublicRoomsV2(stub).ServeHTTP(response, request)

			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d; %s", response.Code, tt.status, response.Body.String())
			}
			if tt.status == http.StatusBadRequest {
				if stub.called {
					t.Fatal("invalid search reached the database")
				}
				return
			}
			if tt.fail {
				return
			}
			if stub.search != tt.search {
				t.Fatalf("search = %#v, want %#v", stub.search, tt.search)
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Fatal("missing JSON content type")
			}
			var result vibe.PublicRoomResult
			err := json.Unmarshal(response.Body.Bytes(), &result)
			if err != nil {
				t.Fatalf("invalid room result: %v", err)
			}
			if result.Total != 23 || result.Count != 1 || result.From != tt.search.From || result.Rooms[0].ListenerCount != 0 {
				t.Fatalf("unexpected result: %#v", result)
			}
		})
	}
}
