package handler

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/vibe"
)

func TestRoomEventsV2SendsCompactSnapshot(t *testing.T) {
	tests := []roomEventsTest{
		{
			name: "uses a dedicated queue snapshot event",
			storage: roomEventsStorageStub{
				playback: &vibe.PlaybackState{
					RoomID:      "electro",
					CurrentSong: &vibe.Song{ID: "song-1", Title: "First song"},
				},
				room:  &vibe.Room{ID: "electro", Name: "Electro"},
				songs: []vibe.Song{{ID: "song-1", Title: "First song"}},
			},
			expectedEvents: []string{
				"event: connected",
				"event: settings_update",
				"event: songs_snapshot",
				"event: playback_update",
			},
			unexpectedEvents: []string{"event: songs_update"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			request := httptest.NewRequest(http.MethodGet, "/rooms/electro/events", nil)
			request = request.WithContext(ctx)
			request = mux.SetURLVars(request, map[string]string{"id": "electro"})
			response := &snapshotResponseRecorder{
				ResponseRecorder: httptest.NewRecorder(),
				cancel:           cancel,
			}
			publisher := &roomEventsPublisherStub{
				subscription: &roomEventsSubscriptionStub{messages: make(chan []byte)},
			}

			RoomEventsV2(publisher, &tt.storage).ServeHTTP(response, request)

			body := response.Body.String()
			for _, expectedEvent := range tt.expectedEvents {
				if !strings.Contains(body, expectedEvent) {
					t.Fatalf("expected response to contain %q, got %q", expectedEvent, body)
				}
			}
			for _, unexpectedEvent := range tt.unexpectedEvents {
				if strings.Contains(body, unexpectedEvent) {
					t.Fatalf("expected response not to contain %q, got %q", unexpectedEvent, body)
				}
			}
		})
	}
}

func TestRoomEventsV2UsesCompactMutation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "replaces the legacy full queue payload"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compactPayload, err := json.Marshal(vibe.SongPositionUpdate{
				Song:     vibe.Song{ID: "song-2", Title: "Updated song"},
				Position: 0,
			})
			if err != nil {
				t.Fatalf("failed to marshal compact payload: %v", err)
			}
			eventData, err := json.Marshal(vibe.RoomEvent{
				ID:      "123-0",
				Type:    vibe.QueueReordered,
				Payload: []byte(`[{"id":"song-1","title":"Legacy full queue song"}]`),
				V2: &vibe.RoomEventV2Payload{
					Type:    vibe.SongUpdated,
					Payload: compactPayload,
				},
			})
			if err != nil {
				t.Fatalf("failed to marshal room event: %v", err)
			}

			messages := make(chan []byte, 1)
			messages <- eventData
			close(messages)
			publisher := &roomEventsPublisherStub{
				subscription: &roomEventsSubscriptionStub{messages: messages},
				replay:       &vibe.ReplaySubscription{AfterID: "122-0"},
			}
			storage := &roomEventsStorageStub{}
			request := httptest.NewRequest(http.MethodGet, "/rooms/electro/events", nil)
			request = mux.SetURLVars(request, map[string]string{"id": "electro"})
			response := httptest.NewRecorder()

			RoomEventsV2(publisher, storage).ServeHTTP(response, request)

			body := response.Body.String()
			if !strings.Contains(body, "event: song_updated") {
				t.Fatalf("expected compact song update, got %q", body)
			}
			if strings.Contains(body, "Legacy full queue song") {
				t.Fatalf("expected legacy queue payload to be filtered, got %q", body)
			}
		})
	}
}
