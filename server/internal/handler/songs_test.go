package handler

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type addSongStorageStub struct {
	room  vibe.Room
	fail  bool
	added bool
}

func (s *addSongStorageStub) GetRoom(_ context.Context, _ string, _ string) (*vibe.Room, error) {
	if s.fail {
		return nil, fmt.Errorf("error private database detail")
	}
	return &s.room, nil
}

func (s *addSongStorageStub) AddSong(_ context.Context, song *vibe.Song) (*vibe.AddSongResult, error) {
	s.added = true
	return &vibe.AddSongResult{Song: *song, Outcome: vibe.AddSongOutcomeAdded}, nil
}

func (s *addSongStorageStub) GetSongs(_ context.Context, _ string) ([]vibe.Song, error) {
	return []vibe.Song{{ID: "first"}, {ID: "second"}}, nil
}

func (s *addSongStorageStub) UpsertPlaybackState(_ context.Context, _ *vibe.PlaybackState) error {
	return nil
}

type addSongEventsStub struct{}

func (s *addSongEventsStub) GetCachedMusicTrack(_ context.Context, _ string, _ string) (*vibe.MusicTrack, error) {
	return &vibe.MusicTrack{}, nil
}

func (s *addSongEventsStub) NotifyRoomUpdate(_ context.Context, _ string, _ vibe.RoomEvent) error {
	return nil
}

func TestAddSongRejectionMessages(t *testing.T) {
	tests := []struct {
		name        string
		noSession   bool
		missingRoom bool
		adminOnly   bool
		admin       bool
		disabled    bool
		invalidLink bool
		invalidBody bool
		live        bool
		fail        bool
		status      int
		code        string
		message     string
	}{
		{name: "missing session", noSession: true, status: 401, code: "song_session_required", message: "Rejoin the room"},
		{name: "room removed", missingRoom: true, status: 404, code: "song_room_not_found", message: "room no longer exists"},
		{name: "non-admin in admin-only room", adminOnly: true, status: 403, code: "song_room_admin_required", message: "Log in as a room admin"},
		{name: "provider disabled", disabled: true, status: 400, code: "song_provider_disabled", message: "provider is not enabled"},
		{name: "invalid provider link", invalidLink: true, status: 400, code: "song_provider_url_invalid", message: "song link is not valid"},
		{name: "invalid request", invalidBody: true, status: 400, code: "song_request_invalid", message: "song details could not be read"},
		{name: "live video", live: true, status: 400, code: "youtube_live_video_not_supported", message: "live"},
		{name: "private server error stays private", fail: true, status: 500},
		{name: "ordinary listener can still add", status: 201},
		{name: "admin can still add in restricted room", adminOnly: true, admin: true, status: 201},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room := vibe.Room{ID: "room", IsAdmin: tt.admin, Settings: vibe.DefaultRoomSettings()}
			room.Settings.OnlyAdminAddSongs = tt.adminOnly
			if tt.missingRoom {
				room.ID = ""
			}
			if tt.disabled {
				room.Settings.EnabledSources = []string{"soundcloud"}
			}
			payload := vibe.AddSongRequest{SourceType: "youtube", SourceID: "test", Title: "Test", Duration: 6375}
			if tt.live {
				payload.Duration = 0
			}
			if tt.invalidLink {
				payload.SourceType = "soundcloud"
				payload.ProviderURL = "https://example.com/track"
			}
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			if tt.invalidBody {
				body = nil
			}
			request := httptest.NewRequest(http.MethodPost, "/rooms/room/songs", bytes.NewReader(body))
			request = mux.SetURLVars(request, map[string]string{"id": "room"})
			if !tt.noSession {
				ctx := context.WithValue(request.Context(), helper.SessionKey, helper.SessionPayload{UserID: "listener"})
				request = request.WithContext(ctx)
			}
			db := &addSongStorageStub{room: room, fail: tt.fail}
			response := httptest.NewRecorder()
			AddSong(db, &addSongEventsStub{}).ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("expected %d, got %d: %s", tt.status, response.Code, response.Body.String())
			}
			if db.added != (tt.status == http.StatusCreated) {
				t.Fatal("unexpected song insertion")
			}
			if tt.code != "" {
				var result client.ErrorCodeResponseBody
				err = json.Unmarshal(response.Body.Bytes(), &result)
				if err != nil {
					t.Fatal(err)
				}
				if result.Error != tt.code || !result.Propagate || !strings.Contains(strings.ToLower(result.Message), strings.ToLower(tt.message)) {
					t.Fatalf("unexpected public error: %+v", result)
				}
				if response.Header().Get("X-preserve-error") != "1" {
					t.Fatal("missing preserved error header")
				}
			}
			if strings.Contains(response.Body.String(), "private database detail") {
				t.Fatal("internal error details exposed")
			}
		})
	}
}
