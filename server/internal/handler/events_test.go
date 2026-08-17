package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/vibe"
)

type roomEventsStorageStub struct {
	playback      *vibe.PlaybackState
	playbackError error
	room          *vibe.Room
	roomError     error
	songs         []vibe.Song
	songsError    error
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
	return s.playback, s.playbackError
}

func (s *roomEventsStorageStub) GetRoom(
	_ context.Context,
	_ string,
	_ string,
) (*vibe.Room, error) {
	return s.room, s.roomError
}

func (s *roomEventsStorageStub) GetSongs(
	_ context.Context,
	_ string,
) ([]vibe.Song, error) {
	return s.songs, s.songsError
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
	_ context.Context,
	_ string,
) (*vibe.SubscriptionContainer, error) {
	return &vibe.SubscriptionContainer{Subscription: s.subscription}, nil
}

func (s *roomEventsPublisherStub) PrepareReplay(
	_ context.Context,
	_ string,
	_ string,
) (*vibe.ReplaySubscription, error) {
	return &vibe.ReplaySubscription{RequiresSnapshot: true}, nil
}

func (s *roomEventsPublisherStub) SubscribeFrom(
	_ context.Context,
	_ string,
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
	name             string
	storage          roomEventsStorageStub
	expectedEvents   []string
	expectedPayloads []string
	unexpectedEvents []string
}

func TestRoomEventsSendsAuthoritativeSnapshot(t *testing.T) {
	tests := []roomEventsTest{
		{
			name: "sends room queue and playback after connecting",
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
				"event: songs_update",
				"event: playback_update",
			},
			expectedPayloads: []string{
				`"name":"Electro"`,
				`"title":"First song"`,
				`"currentSong":{"id":"song-1"`,
			},
		},
		{
			name: "does not send a partial snapshot when songs are unavailable",
			storage: roomEventsStorageStub{
				room:       &vibe.Room{ID: "electro", Name: "Electro"},
				songsError: fmt.Errorf("error songs unavailable"),
			},
			expectedEvents: []string{
				"event: connected",
			},
			unexpectedEvents: []string{
				"event: settings_update",
				"event: songs_update",
				"event: playback_update",
			},
		},
		{
			name: "does not send a partial snapshot when playback is unavailable",
			storage: roomEventsStorageStub{
				room:          &vibe.Room{ID: "electro", Name: "Electro"},
				songs:         []vibe.Song{{ID: "song-1", Title: "First song"}},
				playbackError: fmt.Errorf("error playback unavailable"),
			},
			expectedEvents: []string{
				"event: connected",
			},
			unexpectedEvents: []string{
				"event: settings_update",
				"event: songs_update",
				"event: playback_update",
			},
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
				subscription: &roomEventsSubscriptionStub{
					messages: make(chan []byte),
				},
			}

			RoomEvents(publisher, &tt.storage, &tt.storage).ServeHTTP(response, request)

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
			for _, expectedPayload := range tt.expectedPayloads {
				if !strings.Contains(body, expectedPayload) {
					t.Fatalf("expected response to contain %q, got %q", expectedPayload, body)
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
