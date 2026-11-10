package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/vibe"
)

type existingRoomFixture struct {
	room        vibe.RoomV2
	requestedID string
}

func (f *existingRoomFixture) GetRoomV2(_ context.Context, roomID string, _ string) (*vibe.RoomV2, error) {
	f.requestedID = roomID
	return &f.room, nil
}

func TestGetRoomPreservesExistingID(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		displayName string
	}{
		{name: "legacy accented name", id: "dmen-8", displayName: "dümen-8"},
		{name: "encoded unicode name", id: "u6771-u4eac", displayName: "東京"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := existingRoomFixture{room: vibe.RoomV2{ID: tt.id, Name: tt.displayName}}
			router := mux.NewRouter()
			router.HandleFunc("/api/v2/rooms/{id}", GetRoomV2(&fixture))
			request := httptest.NewRequest(http.MethodGet, "/api/v2/rooms/"+tt.id, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			if fixture.requestedID != tt.id {
				t.Fatalf("lookup ID = %q, want stored ID %q", fixture.requestedID, tt.id)
			}
			var room vibe.RoomV2
			err := json.Unmarshal(response.Body.Bytes(), &room)
			if err != nil {
				t.Fatalf("error decoding room response: %v", err)
			}
			if room.ID != tt.id || room.Name != tt.displayName {
				t.Fatalf("room ID/name = %q/%q, want %q/%q", room.ID, room.Name, tt.id, tt.displayName)
			}
		})
	}
}
