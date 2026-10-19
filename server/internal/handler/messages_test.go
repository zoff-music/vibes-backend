package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type messageRoomStub struct{}

func (messageRoomStub) GetRoomV2(_ context.Context, id string, _ string) (*vibe.RoomV2, error) {
	return &vibe.RoomV2{ID: id}, nil
}

type messageReplayStub struct {
	messages chan []byte
}

func (s messageReplayStub) PrepareReplay(_ context.Context, _ string, _ string) (*vibe.ReplaySubscription, error) {
	return &vibe.ReplaySubscription{AfterID: "0-0", RequiresSnapshot: true}, nil
}

func (s messageReplayStub) SubscribeFrom(_ context.Context, _ string, _ string) (*vibe.SubscriptionContainer, error) {
	return &vibe.SubscriptionContainer{Subscription: s}, nil
}

func (s messageReplayStub) Listen() chan []byte { return s.messages }

func (s messageReplayStub) Destroy() {}

func TestMessagesConnection(t *testing.T) {
	tests := []struct {
		name      string
		user      string
		status    int
		connected bool
	}{
		{name: "empty history still connects", user: "listener", status: 200, connected: true},
		{name: "unauthorized does not connect", status: 401},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := make(chan []byte)
			close(messages)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/rooms/test/messages", nil)
			request = mux.SetURLVars(request, map[string]string{"id": "test"})
			ctx := context.WithValue(request.Context(), helper.SessionKey, vibe.SessionPayload{UserID: tt.user})
			response := httptest.NewRecorder()
			Messages(messageRoomStub{}, messageReplayStub{messages: messages})(response, request.WithContext(ctx))
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d", response.Code, tt.status)
			}
			if strings.Contains(response.Body.String(), "event: connected\ndata: {\"time\":") != tt.connected {
				t.Fatalf("unexpected connection frame: %s", response.Body.String())
			}
		})
	}
}
