package vibe

import (
	"context"
	"slices"
)

type AdminRoomSummaryV2 struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
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
