package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type roomSessionDeleterStub struct {
	clearedRoomID string
	clearedUserID string
	room          *vibe.Room
}

func (s *roomSessionDeleterStub) ClearRoomAdmin(
	_ context.Context,
	roomID string,
	userID string,
) error {
	s.clearedRoomID = roomID
	s.clearedUserID = userID
	return nil
}

func (s *roomSessionDeleterStub) GetRoom(
	_ context.Context,
	_ string,
	_ string,
) (*vibe.Room, error) {
	return s.room, nil
}

type deleteSessionTest struct {
	name           string
	session        helper.SessionPayload
	expectedStatus int
	expectedClear  bool
}

func TestDeleteRoomAdminSession(t *testing.T) {
	tests := []deleteSessionTest{
		{
			name: "clears room admin access",
			session: helper.SessionPayload{
				UserID: "user-1",
			},
			expectedStatus: http.StatusOK,
			expectedClear:  true,
		},
		{
			name:           "requires a session",
			expectedStatus: http.StatusUnauthorized,
			expectedClear:  false,
		},
		{
			name: "rejects a remote session",
			session: helper.SessionPayload{
				UserID:   "user-1",
				AuthType: "remote",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedClear:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &roomSessionDeleterStub{
				room: &vibe.Room{ID: "2000", IsAdmin: false},
			}
			request := httptest.NewRequest(http.MethodDelete, "/rooms/2000/sessions", nil)
			request = mux.SetURLVars(request, map[string]string{"id": "2000"})
			if tt.session.UserID != "" {
				ctx := context.WithValue(request.Context(), helper.SessionKey, tt.session)
				request = request.WithContext(ctx)
			}
			response := httptest.NewRecorder()

			DeleteRoomAdminSession(stub).ServeHTTP(response, request)

			if response.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, response.Code)
			}
			if !tt.expectedClear {
				if stub.clearedRoomID != "" || stub.clearedUserID != "" {
					t.Fatal("expected room admin access not to be cleared")
				}
				return
			}
			if stub.clearedRoomID != "2000" || stub.clearedUserID != "user-1" {
				t.Fatalf(
					"expected room 2000 and user-1 to be cleared, got %s and %s",
					stub.clearedRoomID,
					stub.clearedUserID,
				)
			}

			var session vibe.SessionResponse
			err := json.NewDecoder(response.Body).Decode(&session)
			if err != nil {
				t.Fatalf("expected session response: %v", err)
			}
			if session.IsAdmin || session.Room == nil || session.Room.IsAdmin {
				t.Fatal("expected a guest room session response")
			}
		})
	}
}
