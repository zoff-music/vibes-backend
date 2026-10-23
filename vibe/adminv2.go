package vibe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type AdminRoomSummaryV2 struct {
	Flagged           bool       `json:"flagged"`
	FlaggedAt         *time.Time `json:"flaggedAt,omitempty"`
	CheckedAt         *time.Time `json:"checkedAt,omitempty"`
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	RoomType          RoomType   `json:"roomType"`
	IsPublic          bool       `json:"isPublic"`
	UserCount         int        `json:"userCount"`
	PlaylistItemCount int        `json:"playlistItemCount"`
	ActiveSources     []string   `json:"activeSources"`
	HasAdminPassword  bool       `json:"hasAdminPassword"`
}

// MarshalJSON preserves empty collections in the public JSON contract.
func (v AdminRoomSummaryV2) MarshalJSON() ([]byte, error) {
	type payload AdminRoomSummaryV2
	result := payload(v)

	if result.ActiveSources == nil {
		result.ActiveSources = make([]string, 0)
	}

	body, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling AdminRoomSummaryV2: %w", err)
	}

	return body, nil
}

type AdminRoomResultV2 struct {
	Rooms []AdminRoomSummaryV2 `json:"rooms"`
	From  int                  `json:"from"`
	To    int                  `json:"to"`
	Total int                  `json:"total"`
	Count int                  `json:"count"`
}

// MarshalJSON preserves empty collections in the public JSON contract.
func (v AdminRoomResultV2) MarshalJSON() ([]byte, error) {
	type payload AdminRoomResultV2
	result := payload(v)

	if result.Rooms == nil {
		result.Rooms = make([]AdminRoomSummaryV2, 0)
	}

	body, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling AdminRoomResultV2: %w", err)
	}

	return body, nil
}

type AdminRoomSortV2 string

type AdminRoomSearchV2 struct {
	Flagged    bool
	Query      string
	SortBy     AdminRoomSortV2
	Descending bool
	From       int
	To         int
}

type AdminRoomV2Lister interface {
	ListAdminRoomsV2(ctx context.Context) ([]AdminRoomSummaryV2, error)
}

type AdminRoomV2ListerUserFetcher interface {
	AdminRoomV2Lister
	AdminUserFetcher
}

type AdminRoomV2Searcher interface {
	SearchAdminRoomsV2(ctx context.Context, search AdminRoomSearchV2) (*AdminRoomResultV2, error)
}

type AdminRoomV2UpdaterLister interface {
	AdminRoomV2Lister
	AdminRoomUpdater
	RoomV2Fetcher
}

const AdminRoomSortListenersV2 AdminRoomSortV2 = "listeners"

const AdminRoomSortPlaylistItems AdminRoomSortV2 = "playlistItems"
