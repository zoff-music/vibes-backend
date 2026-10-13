package vibe

import "context"

type PublicRoomV3 struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	RoomType          RoomType `json:"roomType"`
	ListenerCount     int      `json:"listenerCount"`
	PlaylistItemCount int      `json:"playlistItemCount"`
}

type PublicRoomResultV3 struct {
	Rooms []PublicRoomV3 `json:"rooms,format:emitempty"`
	From  int            `json:"from"`
	To    int            `json:"to"`
	Total int            `json:"total"`
	Count int            `json:"count"`
}

type PublicRoomsV3Searcher interface {
	SearchPublicRoomsV3(ctx context.Context, search PublicRoomSearch) (*PublicRoomResultV3, error)
}
