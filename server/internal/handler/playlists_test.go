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
	cache         map[string]vibe.ProviderMetadata
	providerCalls int
	source        string
	force         bool

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
	return &vibe.PlaylistItemMetadataRefresh{Provider: "youtube", SourceID: f.source, Force: f.force}, nil
}
func (f *metadataFixture) GetProviderItem(context.Context, string) (*vibe.ProviderItem, error) {
	f.providerCalls++
	if f.providerErr != nil {
		return nil, fmt.Errorf("error fetching fixture provider: %w", f.providerErr)
	}
	return &f.item, nil
}
func (f *metadataFixture) ApplyProviderMetadata(_ context.Context, _ vibe.PlaylistItemMetadataRefresh, metadata vibe.ProviderMetadata, interval time.Duration) (*vibe.MetadataAffectedRooms, error) {
	f.removed = metadata.Unavailable || metadata.Item.PlaybackRestriction == vibe.PlaybackRestrictionEmbedding
	if !f.removed {
		f.interval = interval
	}
	return &vibe.MetadataAffectedRooms{RoomIDs: []string{"room", "second-room"}}, nil
}
func (f *metadataFixture) DeferPlaylistItemMetadataRefresh(_ context.Context, _ vibe.PlaylistItemMetadataRefresh, interval time.Duration) error {
	f.deferred = interval
	return nil
}
func (f *metadataFixture) GetPlaylistItems(context.Context, string) ([]vibe.PlaylistItem, error) {
	return []vibe.PlaylistItem{}, nil
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
			if tt.removed && (len(f.events) != 2 || f.events[0].Type != vibe.PlaylistItemsUpdate || f.events[1].Type != vibe.PlaybackUpdate) {
				t.Fatalf("missing removal/playback events: %v", f.events)
			}
		})
	}
}

func (f *metadataFixture) GetOrClaimMetadata(_ context.Context, provider, sourceID string, force bool) (*vibe.ProviderMetadataLookup, error) {
	if f.cache == nil {
		f.cache = make(map[string]vibe.ProviderMetadata)
	}
	key := provider + ":" + sourceID
	if force {
		delete(f.cache, key)
	}
	metadata, ok := f.cache[key]
	if ok {
		return &vibe.ProviderMetadataLookup{Metadata: metadata}, nil
	}
	return &vibe.ProviderMetadataLookup{Token: "claim"}, nil
}
func (f *metadataFixture) CompleteMetadata(_ context.Context, provider, sourceID, _ string, metadata vibe.ProviderMetadata, _ time.Duration) (bool, error) {
	f.cache[provider+":"+sourceID] = metadata
	return true, nil
}
func (f *metadataFixture) ReleaseMetadata(context.Context, string, string, string) error { return nil }

func TestMetadataDeduplication(t *testing.T) {
	tests := []struct {
		name        string
		entries     int
		unique      int
		force       bool
		unavailable bool
		calls       int
	}{
		{name: "cached sources avoid provider requests", entries: 10000, unique: 5000, calls: 5000},
		{name: "error reports bypass cached result", entries: 2, unique: 1, force: true, calls: 2},
		{name: "unavailable results remove each duplicate", entries: 3, unique: 1, unavailable: true, calls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &metadataFixture{force: tt.force}
			if tt.unavailable {
				f.providerErr = internalerror.ErrProviderItemNotFound{Err: fmt.Errorf("error missing")}
			}
			h := MetaRefresh{DB: f, Provider: f, Events: f, ProviderName: "youtube"}
			for i := 0; i < tt.entries; i++ {
				f.source = fmt.Sprintf("video-%d", i%tt.unique)
				f.removed = false
				err := h.Handle(context.Background(), nil)
				if err != nil {
					t.Fatal(err)
				}
				if f.removed != tt.unavailable {
					t.Fatalf("entry %d removal=%v", i, f.removed)
				}
			}
			if f.providerCalls != tt.calls {
				t.Fatalf("provider calls=%d want=%d", f.providerCalls, tt.calls)
			}
		})
	}
}
