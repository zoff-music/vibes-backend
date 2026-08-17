package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/vibe"
)

type adminRoomDeleteStub struct {
	deleted      bool
	notification vibe.AdminEvent
	rooms        []vibe.AdminRoomSummary
}

func (s *adminRoomDeleteStub) DeleteAdminRoom(
	_ context.Context,
	_ string,
) (bool, error) {
	return s.deleted, nil
}

func (s *adminRoomDeleteStub) ListAdminRooms(
	_ context.Context,
) ([]vibe.AdminRoomSummary, error) {
	return s.rooms, nil
}

func (s *adminRoomDeleteStub) NotifyAdminUpdate(
	_ context.Context,
	event vibe.AdminEvent,
) error {
	s.notification = event
	return nil
}

type adminDeleteRoomTest struct {
	name           string
	deleted        bool
	expectEmpty    bool
	expectedStatus int
	expectEvent    bool
}

func TestAdminDeleteRoomReturnsNoContent(t *testing.T) {
	tests := []adminDeleteRoomTest{
		{
			name:           "deletes room and publishes refreshed rooms",
			deleted:        true,
			expectEmpty:    true,
			expectedStatus: http.StatusNoContent,
			expectEvent:    true,
		},
		{
			name:           "returns not found without publishing rooms",
			deleted:        false,
			expectedStatus: http.StatusNotFound,
			expectEvent:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &adminRoomDeleteStub{
				deleted: tt.deleted,
				rooms: []vibe.AdminRoomSummary{
					{ID: "remaining-room", Name: "Remaining room"},
				},
			}
			request := httptest.NewRequest(http.MethodDelete, "/admin/rooms/deleted-room", nil)
			request = mux.SetURLVars(request, map[string]string{"id": "deleted-room"})
			response := httptest.NewRecorder()

			AdminDeleteRoom(stub, stub).ServeHTTP(response, request)

			if response.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, response.Code)
			}
			if tt.expectEmpty && response.Body.Len() != 0 {
				t.Fatalf("expected an empty response body, got %q", response.Body.String())
			}
			if !tt.expectEvent {
				if stub.notification.Type != "" {
					t.Fatalf("expected no notification, got %q", stub.notification.Type)
				}
				return
			}

			if stub.notification.Type != vibe.AdminRoomsUpdate {
				t.Fatalf(
					"expected event type %q, got %q",
					vibe.AdminRoomsUpdate,
					stub.notification.Type,
				)
			}
			var rooms []vibe.AdminRoomSummary
			err := json.Unmarshal(stub.notification.Payload, &rooms)
			if err != nil {
				t.Fatalf("expected a room-list payload: %v", err)
			}
			if len(rooms) != 1 || rooms[0].ID != "remaining-room" {
				t.Fatalf("expected refreshed rooms in notification, got %#v", rooms)
			}
		})
	}
}
