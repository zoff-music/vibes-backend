package handler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

type metadataFixture struct {
	vibe.PlaylistItemMetadataRefreshStorage
	vibe.RoomEventBatchV3Notifier
	item        vibe.ProviderItem
	providerErr error
	removed     bool
	interval    time.Duration
	deferred    time.Duration
	events      []vibe.RoomEventV3
}

func (f *metadataFixture) ClaimPlaylistItemMetadataRefresh(context.Context, string, time.Duration) (*vibe.PlaylistItemMetadataRefresh, error) {
	return &vibe.PlaylistItemMetadataRefresh{PlaylistItemID: "item", RoomID: "room", SourceID: "video"}, nil
}
func (f *metadataFixture) GetProviderItem(context.Context, string) (*vibe.ProviderItem, error) {
	if f.providerErr != nil {
		return nil, fmt.Errorf("error fetching fixture provider: %w", f.providerErr)
	}
	return &f.item, nil
}
func (f *metadataFixture) RemovePlaylistItem(_ context.Context, roomID, itemID string) error {
	if roomID != "room" || itemID != "item" {
		return fmt.Errorf("error removing unexpected fixture item")
	}
	f.removed = true
	return nil
}
func (f *metadataFixture) RefreshPlaylistItemMetadata(_ context.Context, _ vibe.PlaylistItemMetadataRefresh, _ vibe.ProviderItem, interval time.Duration) error {
	f.interval = interval
	return nil
}
func (f *metadataFixture) DeferPlaylistItemMetadataRefresh(_ context.Context, _ string, interval time.Duration) error {
	f.deferred = interval
	return nil
}
func (f *metadataFixture) GetPlaybackStateV2(context.Context, string) (*vibe.PlaybackStateV2, error) {
	return &vibe.PlaybackStateV2{}, nil
}
func (f *metadataFixture) GetPlaylistItemPosition(context.Context, string, string) (*vibe.PlaylistItemPositionUpdate, error) {
	return &vibe.PlaylistItemPositionUpdate{PlaylistItem: vibe.PlaylistItem{ID: "item"}}, nil
}
func (f *metadataFixture) NotifyRoomUpdatesV3(_ context.Context, _ string, events []vibe.RoomEventV3) error {
	f.events = events
	return nil
}
func TestMetadataRefresh(t *testing.T) {
	tests := []struct {
		name        string
		restriction string
		providerErr error
		removed     bool
		interval    time.Duration
		deferred    time.Duration
		wantErr     bool
	}{
		{name: "embedding disabled removes item", restriction: vibe.PlaybackRestrictionEmbedding, removed: true},
		{name: "available refreshes in five days", interval: 5 * 24 * time.Hour},
		{name: "age restriction is not embedding disabled", restriction: vibe.PlaybackRestrictionAge, interval: 5 * 24 * time.Hour},
		{name: "missing removes item", providerErr: internalerror.ErrProviderItemNotFound{Err: fmt.Errorf("error missing")}, removed: true},
		{name: "transient error retains item", providerErr: fmt.Errorf("error provider unavailable"), wantErr: true},
		{name: "quota defers without removing", providerErr: internalerror.ErrProviderQuotaExceeded{Err: fmt.Errorf("error quota"), Provider: "youtube"}, deferred: 24 * time.Hour, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &metadataFixture{item: vibe.ProviderItem{PlaybackRestriction: tt.restriction}, providerErr: tt.providerErr}
			h := MetaRefresh{DB: f, Provider: f, Events: f, ProviderName: "youtube"}
			err := h.Handle(context.Background(), nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v", err)
			}
			if f.removed != tt.removed || f.interval != tt.interval || f.deferred != tt.deferred {
				t.Fatalf("removed=%v interval=%v deferred=%v", f.removed, f.interval, f.deferred)
			}
			if tt.removed && (len(f.events) != 2 || f.events[0].Type != vibe.PlaylistItemRemoved || f.events[1].Type != vibe.PlaybackUpdate) {
				t.Fatalf("missing removal/playback events: %v", f.events)
			}
		})
	}
}
