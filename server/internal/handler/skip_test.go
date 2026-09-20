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
	result vibe.SkipSongResult
	fail   bool
}

func (s *chatSkipStub) GetPlaybackState(context.Context, string) (*vibe.PlaybackState, error) {
	return &vibe.PlaybackState{CurrentSong: &vibe.Song{ID: "old", Title: "Song title"}}, nil
}
func (s *chatSkipStub) GetSongs(context.Context, string) ([]vibe.Song, error) {
	return []vibe.Song{}, nil
}
func (s *chatSkipStub) SkipSong(context.Context, string, string) (*vibe.SkipSongResult, error) {
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
		result vibe.SkipSongResult
		kind   string
		fail   bool
	}{
		{name: "completed", result: vibe.SkipSongResult{Skipped: true, PreviousSongID: "old"}, kind: "skipped"},
		{name: "vote", result: vibe.SkipSongResult{Voted: true}, kind: "skipvoted"},
		{name: "threshold reached", result: vibe.SkipSongResult{Skipped: true, Voted: true, PreviousSongID: "old"}, kind: "skipped"},
		{name: "duplicate vote", result: vibe.SkipSongResult{AlreadyVoted: true}},
		{name: "nothing playing"},
		{name: "failed", fail: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.result.Playback = &vibe.PlaybackState{}
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
