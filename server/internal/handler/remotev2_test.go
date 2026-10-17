package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type remoteUpdateFixture struct {
	remote        *vibe.RemoteControlV2
	err           error
	notifications int
}

func (f *remoteUpdateFixture) GetRoomV2(context.Context, string, string) (*vibe.RoomV2, error) {
	return &vibe.RoomV2{ID: "room"}, nil
}

func (f *remoteUpdateFixture) UpdateOwnedRemoteControlV2(context.Context, string, string, vibe.RemoteUpdateRequestV2) (*vibe.RemoteControlV2, error) {
	return f.remote, f.err
}

func (f *remoteUpdateFixture) UpdatePairedRemoteControlV2(context.Context, string, vibe.RemoteUpdateRequestV2) (*vibe.RemoteControlV2, error) {
	return f.remote, f.err
}

func (f *remoteUpdateFixture) NotifyRemoteUpdateV2(context.Context, string, vibe.RemoteEventV2) error {
	f.notifications++
	return nil
}

func TestRemoteUpdateNoOp(t *testing.T) {
	tests := []struct {
		name          string
		remote        *vibe.RemoteControlV2
		err           error
		status        int
		notifications int
	}{
		{name: "unchanged report", err: internalerror.ErrExpected{Err: internalerror.ErrNonRecoverable{Err: vibe.ErrRemoteUnchanged{}}}, status: http.StatusNoContent},
		{name: "changed report", remote: &vibe.RemoteControlV2{ID: "remote"}, status: http.StatusNoContent, notifications: 1},
		{name: "missing or unauthorized remote", remote: &vibe.RemoteControlV2{}, status: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := remoteUpdateFixture{remote: tt.remote, err: tt.err}
			body, err := json.Marshal(vibe.RemoteUpdateRequestV2{})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPatch, "/remotes/remote", bytes.NewReader(body))
			request = mux.SetURLVars(request, map[string]string{"id": "remote"})
			ctx := context.WithValue(request.Context(), helper.SessionKey, vibe.SessionPayload{UserID: "owner", AuthType: "cookie"})
			response := httptest.NewRecorder()

			UpdateRemoteControlV2(&fixture, &fixture)(response, request.WithContext(ctx))

			if response.Code != tt.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if fixture.notifications != tt.notifications {
				t.Fatalf("notifications=%d", fixture.notifications)
			}
		})
	}
}
