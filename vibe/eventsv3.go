package vibe

import (
	"context"
	"encoding/json/v2"
	"fmt"
)

type RoomEventV3StateFetcherUpdater interface {
	RoomEventParticipantFetcherUpdater
	RoomV2Fetcher
	PlaylistItemsFetcher
	PlaybackV2Fetcher
}

type RoomEventV3Payload struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

type PlaylistItemPositionUpdate struct {
	PlaylistItem PlaylistItem `json:"playlistItem"`
	Position     int          `json:"position"`
}

type PlaylistItemIDUpdate struct {
	ID string `json:"id"`
}

type SkipVoteUpdateV2 struct {
	UserID         string `json:"userId"`
	PlaylistItemID string `json:"playlistItemId"`
	CurrentVotes   int    `json:"currentVotes"`
	RequiredVotes  int    `json:"requiredVotes"`
}

const PlaylistItemAdded = "playlist_item_added"
const PlaylistItemRemoved = "playlist_item_removed"
const PlaylistItemUpdated = "playlist_item_updated"
const PlaylistItemsSnapshot = "playlist_items_snapshot"

type RoomEventV3 struct {
	Version int                 `json:"version"`
	ID      string              `json:"id,omitempty"`
	Type    string              `json:"type"`
	Payload []byte              `json:"payload"`
	UserID  string              `json:"userId,omitempty"`
	Origin  string              `json:"origin,omitempty"`
	Compact *RoomEventV3Payload `json:"compact,omitempty"`
}

const PlaylistItemsUpdate = "playlist_items_update"

func ParseRoomEvent(data []byte) (*RoomEventV3, error) {
	var event RoomEventV3
	err := json.Unmarshal(data, &event)
	if err != nil {
		return nil, fmt.Errorf("error decoding room event: %w", err)
	}

	if event.Version != 3 {
		return nil, fmt.Errorf("error unsupported room event version: %d", event.Version)
	}

	return &event, nil
}

func (e RoomEventV3) ToRoomEventV3Payload() (*RoomEventV3Payload, error) {
	result := RoomEventV3Payload{Type: e.Type, Payload: e.Payload}
	if e.Compact != nil {
		result = *e.Compact
	}
	if result.Type == PlaylistItemsUpdate {
		result.Type = PlaylistItemsSnapshot
	}

	return &result, nil
}

type RoomEventV3Notifier interface {
	NotifyRoomUpdateV3(ctx context.Context, roomID string, event RoomEventV3) error
}

type RoomBatchEventV3Notifier interface {
	NotifyRoomUpdatesV3(ctx context.Context, roomID string, events []RoomEventV3) error
}

type RoomEventV3ReplayNotifier interface {
	ReplaySubscriber
	RoomEventV3Notifier
}

type RoomEventBatchV3Notifier interface {
	RoomEventV3Notifier
	RoomBatchEventV3Notifier
}

type RoomRemoteEventV3Notifier interface {
	RoomEventV3Notifier
	RemoteEventNotifierV2
}
