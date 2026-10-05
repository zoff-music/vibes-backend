package vibe

import "context"

type PublicRoomV3 struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	RoomType          RoomType `json:"roomType"`
	ListenerCount     int      `json:"listenerCount"`
	PlaylistItemCount int      `json:"playlistItemCount"`
}

func (r PublicRoomV3) ToPublicRoom() *PublicRoom {
	return &PublicRoom{ID: r.ID, Name: r.Name, ListenerCount: r.ListenerCount, SongCount: r.PlaylistItemCount}
}

type PublicRoomResultV3 struct {
	Rooms []PublicRoomV3 `json:"rooms"`
	From  int            `json:"from"`
	To    int            `json:"to"`
	Total int            `json:"total"`
	Count int            `json:"count"`
}

func (r PublicRoomResultV3) ToPublicRoomResult() *PublicRoomResult {
	rooms := make([]PublicRoom, len(r.Rooms))
	for index, room := range r.Rooms {
		rooms[index] = *room.ToPublicRoom()
	}

	return &PublicRoomResult{Rooms: rooms, From: r.From, To: r.To, Total: r.Total, Count: r.Count}
}

type PublicRoomsV3Searcher interface {
	SearchPublicRoomsV3(ctx context.Context, search PublicRoomSearch) (*PublicRoomResultV3, error)
}
