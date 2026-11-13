package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/zoff-music/vibes-backend/vibe"
)

type roomsFixture struct {
	ids   []string
	fail  bool
	rooms []vibe.PublicRoomV3
}

func (f *roomsFixture) GetRoomsV3(_ context.Context, search vibe.RoomsSearchV3) (*vibe.PublicRoomResultV3, error) {
	f.ids = search.IDs
	if f.fail {
		return nil, fmt.Errorf("error fetching fixture rooms")
	}
	return &vibe.PublicRoomResultV3{Rooms: f.rooms, From: search.From, To: search.To}, nil
}

func TestGetRoomsV3(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		ids    []string
		status int
		fail   bool
	}{
		{name: "ordered and deduplicated", query: "?ids=private,public,private", ids: []string{"private", "public"}, status: 200},
		{name: "trim whitespace", query: "?ids=%20private%20,public", ids: []string{"private", "public"}, status: 200},
		{name: "public listing without IDs", ids: []string{}, status: 200},
		{name: "empty IDs", query: "?ids=", status: 400},
		{name: "empty member", query: "?ids=a,,b", status: 400},
		{name: "duplicate parameter", query: "?ids=a&ids=b", status: 400},
		{name: "too many", query: "?ids=" + strings.Repeat("a,", 20) + "b", status: 400},
		{name: "long ID", query: "?ids=" + strings.Repeat("a", 800), status: 400},
		{name: "invalid UTF8", query: "?ids=%FF", status: 400},
		{name: "null character", query: "?ids=a%00", status: 400},
		{name: "database failure", query: "?ids=private", ids: []string{"private"}, status: 500, fail: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := roomsFixture{fail: tt.fail}
			response := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/api/v3/rooms?from=0&to=2&"+strings.TrimPrefix(tt.query, "?"), nil)
			GetRoomsV3(&fixture).ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("status=%d, want %d: %s", response.Code, tt.status, response.Body.String())
			}
			if !reflect.DeepEqual(fixture.ids, tt.ids) {
				t.Fatalf("ids=%v, want %v", fixture.ids, tt.ids)
			}
			if tt.status == 200 {
				var result vibe.PublicRoomResultV3
				err := json.Unmarshal(response.Body.Bytes(), &result)
				if err != nil {
					t.Fatal(err)
				}
				if result.Rooms == nil {
					t.Fatal("rooms must be an empty array")
				}
				if response.Header().Get("Cache-Control") != "private, no-store" {
					t.Fatal("private response must not be cached")
				}
			}
		})
	}
}

func TestGetRoomsV3Pagination(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		status int
	}{
		{name: "missing range", query: "?ids=private", status: 400},
		{name: "missing end", query: "?from=0", status: 400},
		{name: "missing start", query: "?to=2", status: 400},
		{name: "negative start", query: "?from=-1&to=2", status: 400},
		{name: "reversed", query: "?from=3&to=2", status: 400},
		{name: "oversized", query: "?from=0&to=100", status: 400},
		{name: "invalid", query: "?from=no&to=2", status: 400},
		{name: "duplicate start", query: "?from=0&from=1&to=2", status: 400},
		{name: "inclusive range", query: "?from=4&to=5", status: 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := roomsFixture{}
			response := httptest.NewRecorder()
			GetRoomsV3(&fixture).ServeHTTP(response, httptest.NewRequest("GET", "/api/v3/rooms"+tt.query, nil))
			if response.Code != tt.status {
				t.Fatalf("status=%d, want %d", response.Code, tt.status)
			}
			if tt.status != 200 && fixture.ids != nil {
				t.Fatal("invalid pagination accessed database")
			}
			if tt.status == 200 {
				var result vibe.PublicRoomResultV3
				err := json.Unmarshal(response.Body.Bytes(), &result)
				if err != nil {
					t.Fatal(err)
				}
				if result.From != 4 || result.To != 5 {
					t.Fatalf("range=%d,%d", result.From, result.To)
				}
			}
		})
	}
}
