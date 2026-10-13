package vibe

import (
	"context"
	"encoding/json"
	"fmt"
)

type PublicRoomV3 struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	RoomType          RoomType `json:"roomType"`
	ListenerCount     int      `json:"listenerCount"`
	PlaylistItemCount int      `json:"playlistItemCount"`
}

type PublicRoomResultV3 struct {
	Rooms []PublicRoomV3 `json:"rooms"`
	From  int            `json:"from"`
	To    int            `json:"to"`
	Total int            `json:"total"`
	Count int            `json:"count"`
}

// MarshalJSON preserves empty collections in the public JSON contract.
func (v PublicRoomResultV3) MarshalJSON() ([]byte, error) {
	type payload PublicRoomResultV3
	result := payload(v)

	if result.Rooms == nil {
		result.Rooms = make([]PublicRoomV3, 0)
	}

	body, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling PublicRoomResultV3: %w", err)
	}

	return body, nil
}

type PublicRoomsV3Searcher interface {
	SearchPublicRoomsV3(ctx context.Context, search PublicRoomSearch) (*PublicRoomResultV3, error)
}
