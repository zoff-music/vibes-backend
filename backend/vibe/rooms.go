package vibe

import (
	"context"
	"time"
)

// RoomSettings holds configuration for a room
type RoomSettings struct {
	SkipAllowed       bool    `json:"skipAllowed"`
	DemocraticSkip    bool    `json:"democraticSkip"`
	SkipVoteThreshold float64 `json:"skipVoteThreshold"`
	MaxContinuousAdds int     `json:"maxContinuousAdds"`
	RemoveOnPlay      bool    `json:"removeOnPlay"`
	LoopQueue         bool    `json:"loopQueue"`
	AllowDuplicates   bool    `json:"allowDuplicates"`
}

// DefaultRoomSettings returns sensible defaults
func DefaultRoomSettings() RoomSettings {
	return RoomSettings{
		SkipAllowed:       true,
		DemocraticSkip:    true,
		SkipVoteThreshold: 0.5,
		MaxContinuousAdds: 3,
		RemoveOnPlay:      true,
		LoopQueue:         false,
		AllowDuplicates:   false,
	}
}

// Room represents a music room
type Room struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Mode              string       `json:"mode"`
	HostID            string       `json:"hostId,omitempty"`
	AdminPasswordHash string       `json:"-"`
	HasPassword       bool         `json:"hasPassword"`
	Settings          RoomSettings `json:"settings"`
	CreatedAt         time.Time    `json:"createdAt"`
	UserCount         int          `json:"userCount,omitempty"`
	ActiveSources     []string     `json:"activeSources"`
}

// RoomHostInfo holds info about a host update
type RoomHostInfo struct {
	RoomID    string
	NewHostID string
}

const (
	// RoomModeServer is the mode where the server controls playback
	RoomModeServer = "server"
	// RoomModeHost is the mode where a host controls playback
	RoomModeHost = "host"
)

// CreateRoomRequest is the request payload for creating a room.
type CreateRoomRequest struct {
	Name     string `json:"name"`
	Mode     string `json:"mode,omitempty"`
	Password string `json:"password,omitempty"`
}

// UpdateRoomRequest is the request payload for updating a room.
type UpdateRoomRequest struct {
	Mode     string        `json:"mode,omitempty"`
	Settings *RoomSettings `json:"settings,omitempty"`
}

// IsEmpty returns true if the room is empty/not found
func (r *Room) IsEmpty() bool {
	return r.ID == ""
}

// RoomFetcher fetches room data
type RoomFetcher interface {
	GetRoom(ctx context.Context, id string) (*Room, error)
}

// RoomCreator creates rooms
type RoomCreator interface {
	CreateRoom(ctx context.Context, room *Room) (*Room, error)
	GetRoomByName(ctx context.Context, name string) (*Room, error)
}

// RoomUpdater updates room data
type RoomUpdater interface {
	UpdateRoom(ctx context.Context, room *Room) (*Room, error)
}

// RoomSettingsUpdater fetches and updates room data
type RoomSettingsUpdater interface {
	RoomFetcher
	RoomUpdater
}
