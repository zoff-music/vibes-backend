package vibe

import (
	"context"
	"time"
)

// PlaybackState represents the current playback state of a room
type PlaybackState struct {
	RoomID        string    `json:"-"`
	CurrentSongID *string   `json:"currentSongId"`
	CurrentSong   *Song     `json:"currentSong"`
	IsPlaying     bool      `json:"isPlaying"`
	PositionMs    int64     `json:"positionMs"`
	UpdatedAt     time.Time `json:"updatedAt"`
	ServerTimeMs  int64     `json:"serverTimeMs"`
}

// RoomAction represents a playback action
type RoomAction string

const (
	RoomActionPlay  = "play"
	RoomActionPause = "pause"
	RoomActionSeek  = "seek"
	RoomActionSkip  = "skip"
	RoomActionVote  = "vote"
)

// RoomActionRequest is the request payload for room actions.
type RoomActionRequest struct {
	Action     RoomAction `json:"action"`
	PositionMs int64      `json:"positionMs,omitempty"`
}

// PlaybackFetcher fetches playback state
type PlaybackFetcher interface {
	GetPlaybackState(ctx context.Context, roomID string) (*PlaybackState, error)
}

// PlaybackStateUpdater defines the interface for updating playback state
type PlaybackStateUpdater interface {
	RoomFetcher
	PlaybackFetcher
	UpdatePlayback(ctx context.Context, roomID string, userID string, action RoomAction, positionMs int64) (*PlaybackState, error)
}

// PlaybackController controls playback
type PlaybackController interface {
	UpsertPlaybackState(ctx context.Context, state *PlaybackState) error
}

// ExpiredPlaybackProcessor defines interfaces needed for background room playback automation
type ExpiredPlaybackProcessor interface {
	ProcessNextExpiredPlayback(ctx context.Context) (*PlaybackState, error)
}

// AbandonnedHostProcessor defines interfaces needed for background host management
type AbandonnedHostProcessor interface {
	ProcessNextAbandonedHost(ctx context.Context) (*RoomHostInfo, error)
}
