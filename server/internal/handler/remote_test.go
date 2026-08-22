package handler

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/vibe"
)

type remotePairingStub struct {
	event    vibe.RemoteEvent
	remote   *vibe.RemoteControl
	remoteID string
}

func (s *remotePairingStub) PairRemoteControl(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
) (*vibe.RemoteControl, error) {
	return s.remote, nil
}

func (s *remotePairingStub) NotifyRemoteUpdate(
	_ context.Context,
	remoteID string,
	event vibe.RemoteEvent,
) error {
	s.remoteID = remoteID
	s.event = event
	return nil
}

func TestPairRemoteControlNotifiesMachine(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "publishes paired remote state"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remoteID := "4f31cb5c-40f3-48b5-878a-9a61cdca9a53"
			observedAt := time.Now()
			stub := &remotePairingStub{
				remote: &vibe.RemoteControl{
					ID:                 remoteID,
					CurrentRoomID:      "room-1",
					CurrentSongID:      "song-1",
					PlaybackPositionMs: 1200,
					PlaybackIsPlaying:  true,
					PlaybackObservedAt: observedAt,
					Paired:             true,
				},
			}
			request := httptest.NewRequest(
				http.MethodPost,
				"/remotes/"+remoteID+"/sessions",
				strings.NewReader(`{"pairingCode":"ABCD1234"}`),
			)
			request = mux.SetURLVars(request, map[string]string{"id": remoteID})
			response := httptest.NewRecorder()

			PairRemoteControl(stub, stub, "secret").ServeHTTP(response, request)

			if response.Code != http.StatusCreated {
				t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
			}
			if stub.remoteID != remoteID {
				t.Fatalf("expected remote ID %q, got %q", remoteID, stub.remoteID)
			}
			if stub.event.Type != vibe.RemoteStateUpdate {
				t.Fatalf("expected event type %q, got %q", vibe.RemoteStateUpdate, stub.event.Type)
			}
			if stub.event.Origin != vibe.RemoteOriginController {
				t.Fatalf("expected event origin %q, got %q", vibe.RemoteOriginController, stub.event.Origin)
			}
			if !stub.event.Online || !stub.event.Paired {
				t.Fatal("expected paired remote to be online and paired")
			}

			var session vibe.RemoteSession
			err := json.UnmarshalRead(response.Body, &session)
			if err != nil {
				t.Fatalf("expected remote session response: %v", err)
			}
			if session.ID != remoteID || !session.Online || !session.Paired {
				t.Fatal("expected response to contain active paired remote")
			}
		})
	}
}
