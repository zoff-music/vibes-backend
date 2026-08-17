package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type playbackStorageStub struct {
	playback   *vibe.PlaybackState
	room       *vibe.Room
	updateCall bool
}

func (s *playbackStorageStub) GetPlaybackState(
	_ context.Context,
	_ string,
) (*vibe.PlaybackState, error) {
	state := *s.playback
	return &state, nil
}

func (s *playbackStorageStub) GetRoom(
	_ context.Context,
	_ string,
	_ string,
) (*vibe.Room, error) {
	return s.room, nil
}

func (s *playbackStorageStub) UpdatePlayback(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ int,
) (*vibe.PlaybackState, error) {
	s.updateCall = true
	return s.playback, nil
}

type playbackNotifierStub struct {
	remoteEvent  vibe.RemoteEvent
	remoteID     string
	roomNotified bool
}

func (s *playbackNotifierStub) NotifyRemoteUpdate(
	_ context.Context,
	remoteID string,
	event vibe.RemoteEvent,
) error {
	s.remoteID = remoteID
	s.remoteEvent = event
	return nil
}

func (s *playbackNotifierStub) NotifyRoomUpdate(
	_ context.Context,
	_ string,
	_ vibe.RoomEvent,
) error {
	s.roomNotified = true
	return nil
}

func TestUpdatePlaybackStateTargetsRemoteMachine(t *testing.T) {
	tests := []struct {
		name              string
		action            string
		initialIsPlaying  bool
		expectedIsPlaying bool
	}{
		{
			name:              "pauses only the paired machine",
			action:            vibe.RoomActionPause,
			initialIsPlaying:  true,
			expectedIsPlaying: false,
		},
		{
			name:              "plays only the paired machine",
			action:            vibe.RoomActionPlay,
			initialIsPlaying:  false,
			expectedIsPlaying: true,
		},
		{
			name:              "seeks without changing the paired machine play state",
			action:            vibe.RoomActionSeek,
			initialIsPlaying:  true,
			expectedIsPlaying: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &playbackStorageStub{
				playback: &vibe.PlaybackState{
					RoomID:      "electro",
					CurrentSong: &vibe.Song{ID: "song-1"},
					IsPlaying:   tt.initialIsPlaying,
					PositionMs:  1200,
				},
				room: &vibe.Room{ID: "electro", Mode: vibe.RoomModeServer},
			}
			notifier := &playbackNotifierStub{}
			request := httptest.NewRequest(
				http.MethodPut,
				"/rooms/electro/states",
				strings.NewReader(`{"action":"`+tt.action+`","positionMs":2400}`),
			)
			request = mux.SetURLVars(request, map[string]string{"id": "electro"})
			ctx := context.WithValue(request.Context(), helper.SessionKey, helper.SessionPayload{
				AuthType: "remote",
				RemoteID: "remote-1",
				UserID:   "owner-1",
			})
			response := httptest.NewRecorder()

			UpdatePlaybackState(storage, notifier, notifier).ServeHTTP(
				response,
				request.WithContext(ctx),
			)

			if response.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
			}
			if storage.updateCall {
				t.Fatal("expected authoritative room playback to remain unchanged")
			}
			if notifier.roomNotified {
				t.Fatal("expected no room-wide playback notification")
			}
			if notifier.remoteID != "remote-1" {
				t.Fatalf("expected remote ID %q, got %q", "remote-1", notifier.remoteID)
			}
			if notifier.remoteEvent.PlaybackIsPlaying != tt.expectedIsPlaying {
				t.Fatalf("expected playing state %t, got %t", tt.expectedIsPlaying, notifier.remoteEvent.PlaybackIsPlaying)
			}
			if notifier.remoteEvent.PlaybackPositionMs != 2400 {
				t.Fatalf("expected position %d, got %d", 2400, notifier.remoteEvent.PlaybackPositionMs)
			}
			if notifier.remoteEvent.CurrentSongID != "song-1" {
				t.Fatalf("expected song ID %q, got %q", "song-1", notifier.remoteEvent.CurrentSongID)
			}

			var state vibe.PlaybackState
			err := json.NewDecoder(response.Body).Decode(&state)
			if err != nil {
				t.Fatalf("expected playback response: %v", err)
			}
			if state.IsPlaying != tt.expectedIsPlaying {
				t.Fatalf("expected response playing state %t, got %t", tt.expectedIsPlaying, state.IsPlaying)
			}
		})
	}
}
