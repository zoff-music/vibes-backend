package handler

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type chatSkipStub struct {
	messageStorageStub
	result vibe.SkipPlaylistItemResult
	fail   bool
}

func (s *chatSkipStub) GetRoomV2(context.Context, string, string) (*vibe.RoomV2, error) {
	return s.room.ToRoomV2(), nil
}

func (s *chatSkipStub) GetPlaybackStateV2(context.Context, string) (*vibe.PlaybackStateV2, error) {
	return &vibe.PlaybackStateV2{CurrentPlaylistItem: &vibe.PlaylistItem{ID: "old", Title: "Song title"}}, nil
}
func (s *chatSkipStub) GetPlaylistItems(context.Context, string) ([]vibe.PlaylistItem, error) {
	return []vibe.PlaylistItem{}, nil
}
func (s *chatSkipStub) SkipPlaylistItem(context.Context, string, string) (*vibe.SkipPlaylistItemResult, error) {
	if s.fail {
		return nil, fmt.Errorf("skip unavailable")
	}
	return &s.result, nil
}

type chatSkipEvents struct{ messageEventsStub }

func (s *chatSkipEvents) NotifyRoomUpdates(context.Context, string, []vibe.RoomEvent) error {
	return nil
}

func TestSkipChatActivity(t *testing.T) {
	tests := []struct {
		name   string
		result vibe.SkipPlaylistItemResult
		kind   string
		fail   bool
	}{
		{name: "completed", result: vibe.SkipPlaylistItemResult{Skipped: true, PreviousPlaylistItemID: "old"}, kind: "skipped"},
		{name: "vote", result: vibe.SkipPlaylistItemResult{Voted: true}, kind: "skipvoted"},
		{name: "threshold reached", result: vibe.SkipPlaylistItemResult{Skipped: true, Voted: true, PreviousPlaylistItemID: "old"}, kind: "skipped"},
		{name: "duplicate vote", result: vibe.SkipPlaylistItemResult{AlreadyVoted: true}},
		{name: "nothing playing"},
		{name: "failed", fail: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.result.Playback = &vibe.PlaybackStateV2{}
			db := &chatSkipStub{messageStorageStub: messageStorageStub{room: vibe.Room{ID: "electro", IsAdmin: true}}, result: tt.result, fail: tt.fail}
			events := &chatSkipEvents{}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/rooms/electro/skips", nil)
			request = mux.SetURLVars(request, map[string]string{"id": "electro"})
			request = request.WithContext(context.WithValue(request.Context(), helper.SessionKey, vibe.SessionPayload{UserID: "signed-session", AuthType: "cookie"}))
			response := httptest.NewRecorder()
			SkipSong(db, events).ServeHTTP(response, request)
			expectedStatus := http.StatusOK
			if tt.fail {
				expectedStatus = http.StatusInternalServerError
			}
			if response.Code != expectedStatus {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
			if tt.kind == "" {
				if events.event.Type != "" {
					t.Fatal("no-op or failed skip published activity")
				}
				return
			}
			var message vibe.RoomMessage
			err := json.Unmarshal(events.event.Payload, &message)
			if err != nil {
				t.Fatal(err)
			}
			if message.Kind != tt.kind || message.Text != "Song title" || message.Name != "Actual name" || !message.IsAdmin || message.CreatedAt == 0 {
				t.Fatalf("incorrect skip activity: %+v", message)
			}
			if db.usages != 0 {
				t.Fatal("skip counted as a sent chat message")
			}
		})
	}
}
