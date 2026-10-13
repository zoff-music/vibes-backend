package vibe

import (
	"context"
	"encoding/json/v2"
	"fmt"
)

type AdminRoomSummaryV2 struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	RoomType          RoomType `json:"roomType"`
	IsPublic          bool     `json:"isPublic"`
	UserCount         int      `json:"userCount"`
	PlaylistItemCount int      `json:"playlistItemCount"`
	ActiveSources     []string `json:"activeSources"`
	HasAdminPassword  bool     `json:"hasAdminPassword"`
}

type AdminRoomResultV2 struct {
	Rooms []AdminRoomSummaryV2 `json:"rooms"`
	From  int                  `json:"from"`
	To    int                  `json:"to"`
	Total int                  `json:"total"`
	Count int                  `json:"count"`
}

type AdminRoomSortV2 string

const AdminRoomSortListenersV2 AdminRoomSortV2 = "listeners"

const AdminRoomSortPlaylistItems AdminRoomSortV2 = "playlistItems"

type AdminRoomSearchV2 struct {
	Query      string
	SortBy     AdminRoomSortV2
	Descending bool
	From       int
	To         int
}

type AdminRoomV2Lister interface {
	ListAdminRoomsV2(ctx context.Context) ([]AdminRoomSummaryV2, error)
}

type AdminRoomV2Searcher interface {
	SearchAdminRoomsV2(ctx context.Context, search AdminRoomSearchV2) (*AdminRoomResultV2, error)
}

type AdminRoomV2UpdaterLister interface {
	AdminRoomV2Lister
	AdminRoomUpdater
}

type AdminRoomV2DeleterLister interface {
	AdminRoomV2Lister
	AdminRoomDeleter
}

type AdminEventV2 struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

type AdminEventRecord struct {
	Version int          `json:"version"`
	Event   AdminEventV2 `json:"event"`
}

func ParseAdminEvent(data []byte) (*AdminEventV2, error) {
	var record AdminEventRecord
	err := json.Unmarshal(data, &record)
	if err != nil {
		return nil, fmt.Errorf("error decoding admin event record: %w", err)
	}

	if record.Version != 2 {
		return nil, fmt.Errorf("error unsupported admin event version: %d", record.Version)
	}

	return &record.Event, nil
}

type AdminEventV2Notifier interface {
	NotifyAdminUpdateV2(ctx context.Context, event AdminEventV2) error
}
