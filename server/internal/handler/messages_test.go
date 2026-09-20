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
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type messageStorageStub struct {
	room   vibe.Room
	usages int
}

func (s *messageStorageStub) CreateMessageUsage(_ context.Context, roomID string, _ time.Time) error {
	s.usages++
	return nil
}

func (s *messageStorageStub) GetRoom(_ context.Context, _, _ string) (*vibe.Room, error) {
	return &s.room, nil
}
func (s *messageStorageStub) GetOrCreateSessionProfile(_ context.Context, _ string) (*vibe.SessionProfile, error) {
	return &vibe.SessionProfile{Name: "Actual name"}, nil
}

type messageEventsStub struct {
	event  vibe.RoomEvent
	roomID string
	fail   bool
}

func (s *messageEventsStub) NotifyRoomUpdate(_ context.Context, roomID string, event vibe.RoomEvent) error {
	if s.fail {
		return fmt.Errorf("error stream unavailable")
	}
	s.roomID = roomID
	s.event = event
	return nil
}

func TestCreateMessages(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		session bool
		cast    bool
		admin   bool
		missing bool
		fail    bool
		status  int
	}{
		{name: "listener", text: " hello ", session: true, status: 201},
		{name: "admin", text: "hi", session: true, admin: true, status: 201},
		{name: "unicode", text: "hello 🎶", session: true, status: 201},
		{name: "empty", text: "  ", session: true, status: 400},
		{name: "too long", text: strings.Repeat("x", 501), session: true, status: 400},
		{name: "no session", text: "hi", status: 401},
		{name: "cast receiver", text: "hi", session: true, cast: true, status: 401},
		{name: "no room", text: "hi", session: true, missing: true, status: 404},
		{name: "retention unavailable", text: "hi", session: true, fail: true, status: 503},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &messageStorageStub{room: vibe.Room{ID: "electro", IsAdmin: tt.admin}}
			if tt.missing {
				db.room.ID = ""
			}
			events := &messageEventsStub{fail: tt.fail}
			payload, err := json.Marshal(vibe.CreateMessageRequest{Text: tt.text})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/rooms/electro/messages", bytes.NewReader(payload))
			request = mux.SetURLVars(request, map[string]string{"id": "electro"})
			if tt.session {
				session := vibe.SessionPayload{UserID: "signed-session", AuthType: "cookie"}
				if tt.cast {
					session.AuthType = "cast"
				}
				request = request.WithContext(context.WithValue(request.Context(), helper.SessionKey, session))
			}
			response := httptest.NewRecorder()
			CreateMessages(db, events).ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("status %d, want %d: %s", response.Code, tt.status, response.Body.String())
			}
			expectedUsages := 0
			if tt.status == 201 {
				expectedUsages = 1
			}
			if db.usages != expectedUsages {
				t.Fatalf("recorded %d usages, want %d", db.usages, expectedUsages)
			}
			if tt.status != 201 {
				if events.event.Type != "" {
					t.Fatal("rejected message was published")
				}
				return
			}
			var message vibe.RoomMessage
			err = json.Unmarshal(events.event.Payload, &message)
			if err != nil {
				t.Fatal(err)
			}
			if message.Name != "Actual name" || message.UserID != "signed-session" || message.IsAdmin != tt.admin || message.Kind != "chat" || message.ID == "" || message.Text != strings.TrimSpace(tt.text) {
				t.Fatalf("wrong attributed message: %+v", message)
			}
			if events.roomID != "electro" || events.event.UserID != "" {
				t.Fatal("message must reach every chat subscriber, including sender")
			}
		})
	}
}

type messageSubscriptionStub struct{ messages chan []byte }

func (s *messageSubscriptionStub) Listen() chan []byte { return s.messages }
func (s *messageSubscriptionStub) Destroy()            {}

type messageReplayStub struct {
	topic        string
	cursor       string
	after        string
	reset        bool
	subscription *messageSubscriptionStub
}

func (s *messageReplayStub) PrepareReplay(_ context.Context, topic, cursor string) (*vibe.ReplaySubscription, error) {
	s.topic = topic
	s.cursor = cursor
	return &vibe.ReplaySubscription{AfterID: "123-0", RequiresSnapshot: s.reset}, nil
}
func (s *messageReplayStub) SubscribeFrom(_ context.Context, _ string, after string) (*vibe.SubscriptionContainer, error) {
	s.after = after
	return &vibe.SubscriptionContainer{Subscription: s.subscription}, nil
}
func TestMessagesReplay(t *testing.T) {
	tests := []struct {
		name   string
		header string
		query  string
		reset  bool
		after  string
		cursor string
	}{
		{name: "new subscription gets history", reset: true, after: "0-0"},
		{name: "expired cursor gets retained history", query: "12-0", reset: true, after: "0-0", cursor: "12-0"},
		{name: "reconnect resumes", query: "123-0", after: "123-0", cursor: "123-0"},
		{name: "header wins", header: "123-0", query: "12-0", after: "123-0", cursor: "123-0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := json.Marshal(vibe.RoomMessage{ID: "entry", Name: "Mia", Kind: "voted", Text: "Song title"})
			if err != nil {
				t.Fatal(err)
			}
			wire, err := json.Marshal(vibe.RoomEvent{ID: "124-0", Type: vibe.MessageEvent, Payload: payload})
			if err != nil {
				t.Fatal(err)
			}
			subscription := &messageSubscriptionStub{messages: make(chan []byte, 1)}
			subscription.messages <- wire
			close(subscription.messages)
			replay := &messageReplayStub{reset: tt.reset, subscription: subscription}
			request := httptest.NewRequest(http.MethodGet, "/?lastEventId="+tt.query, nil)
			request.Header.Set("Last-Event-ID", tt.header)
			request = mux.SetURLVars(request, map[string]string{"id": "electro"})
			request = request.WithContext(context.WithValue(request.Context(), helper.SessionKey, vibe.SessionPayload{UserID: "listener"}))
			response := httptest.NewRecorder()
			Messages(&messageStorageStub{room: vibe.Room{ID: "electro"}}, replay).ServeHTTP(response, request)
			if replay.topic != "chat:electro" || replay.after != tt.after || replay.cursor != tt.cursor {
				t.Fatalf("wrong replay: %+v", replay)
			}
			if !strings.Contains(response.Body.String(), "event: message\n") || !strings.Contains(response.Body.String(), "id: 124-0\n") || !strings.Contains(response.Body.String(), "event: event_cursor\n") {
				t.Fatalf("missing replay framing: %s", response.Body.String())
			}
		})
	}
}
