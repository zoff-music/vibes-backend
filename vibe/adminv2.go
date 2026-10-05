package vibe

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"slices"
)

type AdminRoomSummaryV2 struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	RoomType          RoomType `json:"roomType"`
	UserCount         int      `json:"userCount"`
	PlaylistItemCount int      `json:"playlistItemCount"`
	ActiveSources     []string `json:"activeSources"`
	HasAdminPassword  bool     `json:"hasAdminPassword"`
}

func (r AdminRoomSummaryV2) ToAdminRoomSummary() *AdminRoomSummary {
	return &AdminRoomSummary{
		ID:               r.ID,
		Name:             r.Name,
		UserCount:        r.UserCount,
		SongCount:        r.PlaylistItemCount,
		ActiveSources:    slices.Clone(r.ActiveSources),
		HasAdminPassword: r.HasAdminPassword,
	}
}

func (r AdminRoomSummary) ToAdminRoomSummaryV2() *AdminRoomSummaryV2 {
	return &AdminRoomSummaryV2{
		ID:                r.ID,
		Name:              r.Name,
		RoomType:          RoomTypeMusic,
		UserCount:         r.UserCount,
		PlaylistItemCount: r.SongCount,
		ActiveSources:     slices.Clone(r.ActiveSources),
		HasAdminPassword:  r.HasAdminPassword,
	}
}

type AdminRoomResultV2 struct {
	Rooms []AdminRoomSummaryV2 `json:"rooms"`
	From  int                  `json:"from"`
	To    int                  `json:"to"`
	Total int                  `json:"total"`
	Count int                  `json:"count"`
}

func (r AdminRoomResultV2) ToAdminRoomResult() *AdminRoomResult {
	var rooms []AdminRoomSummary
	if r.Rooms != nil {
		rooms = make([]AdminRoomSummary, len(r.Rooms))
		for index, room := range r.Rooms {
			rooms[index] = *room.ToAdminRoomSummary()
		}
	}

	return &AdminRoomResult{Rooms: rooms, From: r.From, To: r.To, Total: r.Total, Count: r.Count}
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

func (e AdminEvent) ToAdminEventV2() (*AdminEventV2, error) {
	event := AdminEventV2{Type: e.Type, Payload: e.Payload}
	if e.Type != AdminRoomsUpdate {
		return &event, nil
	}

	var legacy []AdminRoomSummary
	err := json.Unmarshal(e.Payload, &legacy)
	if err != nil {
		return nil, fmt.Errorf("error decoding legacy admin room update: %w", err)
	}

	var rooms []AdminRoomSummaryV2
	if legacy != nil {
		rooms = make([]AdminRoomSummaryV2, len(legacy))
		for index, room := range legacy {
			rooms[index] = *room.ToAdminRoomSummaryV2()
		}
	}

	event.Payload, err = json.Marshal(rooms)
	if err != nil {
		return nil, fmt.Errorf("error encoding admin room update: %w", err)
	}

	return &event, nil
}

func (e AdminEventV2) ToAdminEvent() (*AdminEvent, error) {
	event := AdminEvent{Type: e.Type, Payload: e.Payload}
	if e.Type != AdminRoomsUpdate {
		return &event, nil
	}

	var rooms []AdminRoomSummaryV2
	err := json.Unmarshal(e.Payload, &rooms)
	if err != nil {
		return nil, fmt.Errorf("error decoding admin room update: %w", err)
	}

	var legacy []AdminRoomSummary
	if rooms != nil {
		legacy = make([]AdminRoomSummary, len(rooms))
		for index, room := range rooms {
			legacy[index] = *room.ToAdminRoomSummary()
		}
	}

	event.Payload, err = json.Marshal(legacy)
	if err != nil {
		return nil, fmt.Errorf("error encoding legacy admin room update: %w", err)
	}

	return &event, nil
}

func ParseAdminEvent(data []byte) (*AdminEventV2, error) {
	var record AdminEventRecord
	err := json.Unmarshal(data, &record)
	if err != nil {
		return nil, fmt.Errorf("error decoding admin event record: %w", err)
	}
	if record.Version == 2 {
		return &record.Event, nil
	}
	if record.Version != 0 {
		return nil, fmt.Errorf("error unsupported admin event version: %d", record.Version)
	}

	var legacy AdminEvent
	err = json.Unmarshal(data, &legacy)
	if err != nil {
		return nil, fmt.Errorf("error decoding retained legacy admin event: %w", err)
	}

	event, err := legacy.ToAdminEventV2()
	if err != nil {
		return nil, fmt.Errorf("error converting retained legacy admin event: %w", err)
	}

	return event, nil
}

type AdminEventV2Notifier interface {
	NotifyAdminUpdateV2(ctx context.Context, event AdminEventV2) error
}
