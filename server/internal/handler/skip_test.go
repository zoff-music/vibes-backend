package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type skipMetadataFixture struct {
	vibe.RoomV2Skipper
	vibe.RoomEventBatchV3Notifier
	result    vibe.SkipPlaylistItemResult
	skipErr   error
	scheduled string
	skipCalls int
}

func (f *skipMetadataFixture) GetPlaybackStateV2(context.Context, string) (*vibe.PlaybackStateV2, error) {
	return &vibe.PlaybackStateV2{}, nil
}
func (f *skipMetadataFixture) SkipPlaylistItem(context.Context, string, string) (*vibe.SkipPlaylistItemResult, error) {
	f.skipCalls++
	if f.skipErr != nil {
		return nil, fmt.Errorf("error skipping fixture: %w", f.skipErr)
	}
	return &f.result, nil
}
func (f *skipMetadataFixture) SchedulePlaylistItemMetadataRefresh(_ context.Context, roomID, itemID string) error {
	f.scheduled = roomID + "/" + itemID
	return nil
}
func (f *skipMetadataFixture) GetPlaylistItemPosition(context.Context, string, string) (*vibe.PlaylistItemPositionUpdate, error) {
	return &vibe.PlaylistItemPositionUpdate{}, nil
}
func (f *skipMetadataFixture) GetRoomV2(context.Context, string, string) (*vibe.RoomV2, error) {
	return &vibe.RoomV2{}, nil
}
func (f *skipMetadataFixture) NotifyRoomUpdatesV3(context.Context, string, []vibe.RoomEventV3) error {
	return nil
}
func TestSkipMetadataReport(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		itemID        string
		emptyBody     bool
		skipped       bool
		denied        bool
		wantScheduled string
		status        int
	}{
		{name: "legacy empty body", emptyBody: true, status: 200},
		{name: "legacy empty object", status: 200},
		{name: "youtube skip", code: "101", itemID: "failed", skipped: true, wantScheduled: "room/failed", status: 200},
		{name: "soundcloud vote", code: "soundcloud_error", itemID: "failed", wantScheduled: "room/failed", status: 200},
		{name: "stale report never flags next item", code: "150", itemID: "old", skipped: true, status: 200},
		{name: "missing item rejected", code: "101", status: 400},
		{name: "oversized code rejected", code: strings.Repeat("x", 129), itemID: "failed", status: 400},
		{name: "denied skip never schedules", code: "101", itemID: "failed", denied: true, status: 403},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &skipMetadataFixture{result: vibe.SkipPlaylistItemResult{Skipped: tt.skipped, PreviousPlaylistItemID: "failed", Playback: &vibe.PlaybackStateV2{CurrentPlaylistItem: &vibe.PlaylistItem{ID: "failed"}}}}
			if tt.denied {
				f.skipErr = internalerror.ErrSkipDisabled{Err: fmt.Errorf("error denied")}
			}
			body, err := json.Marshal(vibe.SkipPlaylistItemRequest{ErrorCode: tt.code, PlaylistItemID: tt.itemID})
			if err != nil {
				t.Fatal(err)
			}
			if tt.emptyBody {
				body = nil
			}
			req := httptest.NewRequest("POST", "/rooms/room/skips", bytes.NewReader(body))
			req = mux.SetURLVars(req, map[string]string{"id": "room"})
			req = req.WithContext(context.WithValue(req.Context(), helper.SessionKey, vibe.SessionPayload{UserID: "user"}))
			response := httptest.NewRecorder()
			SkipPlaylistItem(f, f)(response, req)
			if response.Code != tt.status || f.scheduled != tt.wantScheduled {
				t.Fatalf("status=%d scheduled=%q body=%s", response.Code, f.scheduled, response.Body.String())
			}
			if tt.status == 400 && f.skipCalls != 0 {
				t.Fatal("invalid request executed a skip")
			}
		})
	}
}
