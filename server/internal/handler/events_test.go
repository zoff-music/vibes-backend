package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/vibe"
)

type roomEventsStorageStub struct {
	playback *vibe.PlaybackState
	room     *vibe.Room
	songs    []vibe.Song
}

func (s *roomEventsStorageStub) GetActiveListenerCounts(
	_ context.Context,
	_ string,
	_ time.Duration,
) (vibe.ListenerCounts, error) {
	return vibe.ListenerCounts{}, nil
}

func (s *roomEventsStorageStub) GetPlaybackState(
	_ context.Context,
	_ string,
) (*vibe.PlaybackState, error) {
	return s.playback, nil
}

func (s *roomEventsStorageStub) GetRoom(
	_ context.Context,
	_ string,
	_ string,
) (*vibe.Room, error) {
	return s.room, nil
}

func (s *roomEventsStorageStub) GetSongs(
	_ context.Context,
	_ string,
) ([]vibe.Song, error) {
	return s.songs, nil
}

func (s *roomEventsStorageStub) UpdateParticipant(
	_ context.Context,
	_ string,
	_ string,
	_ bool,
	_ bool,
	_ string,
) error {
	return nil
}

type roomEventsSubscriptionStub struct {
	messages chan []byte
}

func (s *roomEventsSubscriptionStub) Destroy() {}

func (s *roomEventsSubscriptionStub) Listen() chan []byte {
	return s.messages
}

type roomEventsPublisherStub struct {
	subscription *roomEventsSubscriptionStub
}

func (s *roomEventsPublisherStub) NotifyRoomUpdate(
	_ context.Context,
	_ string,
	_ vibe.RoomEvent,
) error {
	return nil
}

func (s *roomEventsPublisherStub) Subscribe(
	_ string,
) (*vibe.SubscriptionContainer, error) {
	return &vibe.SubscriptionContainer{Subscription: s.subscription}, nil
}

type snapshotResponseRecorder struct {
	*httptest.ResponseRecorder
	cancel     context.CancelFunc
	flushCount int
}

func (r *snapshotResponseRecorder) Flush() {
	r.flushCount++
	r.ResponseRecorder.Flush()
	if r.flushCount == 4 {
		r.cancel()
	}
}

type roomEventsTest struct {
	name           string
	expectedEvents []string
}

func TestRoomEventsSendsAuthoritativeSnapshot(t *testing.T) {
	tests := []roomEventsTest{
		{
			name: "sends room queue and playback after connecting",
			expectedEvents: []string{
				"event: connected",
				"event: settings_update",
				"event: songs_update",
				"event: playback_update",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			request := httptest.NewRequest(http.MethodGet, "/rooms/electro/events", nil)
			request = request.WithContext(ctx)
			request = mux.SetURLVars(request, map[string]string{"id": "electro"})
			response := &snapshotResponseRecorder{
				ResponseRecorder: httptest.NewRecorder(),
				cancel:           cancel,
			}
			storage := &roomEventsStorageStub{
				playback: &vibe.PlaybackState{RoomID: "electro"},
				room:     &vibe.Room{ID: "electro", Name: "Electro"},
				songs:    []vibe.Song{{ID: "song-1", Title: "First song"}},
			}
			publisher := &roomEventsPublisherStub{
				subscription: &roomEventsSubscriptionStub{
					messages: make(chan []byte),
				},
			}

			RoomEvents(publisher, storage).ServeHTTP(response, request)

			body := response.Body.String()
			previousIndex := -1
			for _, expectedEvent := range tt.expectedEvents {
				index := strings.Index(body, expectedEvent)
				if index == -1 {
					t.Fatalf("expected response to contain %q, got %q", expectedEvent, body)
				}
				if index <= previousIndex {
					t.Fatalf("expected %q after previous event in %q", expectedEvent, body)
				}
				previousIndex = index
			}
		})
	}
}
