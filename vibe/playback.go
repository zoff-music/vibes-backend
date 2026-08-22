package vibe

import (
	"context"
	"time"
)

// PlaybackState represents the current playback state of a room
type PlaybackState struct {
	RoomID       string    `json:"-"`
	CurrentSong  *Song     `json:"currentSong"`
	IsPlaying    bool      `json:"isPlaying"`
	PositionMs   int       `json:"positionMs"`
	UpdatedAt    time.Time `json:"updatedAt"`
	ServerTimeMs int       `json:"serverTimeMs"`
}

func (p *PlaybackState) IsEmpty() bool {
	return p.RoomID == ""
}

// RoomActionRequest is the request payload for room actions.
type RoomActionRequest struct {
	Action     string `json:"action"`
	PositionMs int    `json:"positionMs,omitzero"`
}

type PlaybackFailureRequest struct {
	SongID string `json:"songId"`
}

// PlaybackFetcher fetches playback state
type PlaybackFetcher interface {
	GetPlaybackState(ctx context.Context, roomID string) (*PlaybackState, error)
}

// PlaybackStateUpdater defines the interface for updating playback state
type RoomGetterPlaybackUpdater interface {
	PlaybackFetcher
	GetRoom(ctx context.Context, roomID string, userID string) (*Room, error)
	UpdatePlayback(ctx context.Context, roomID string, userID string, action string, positionMs int) (*PlaybackState, error)
}

// PlaybackController controls playback
type PlaybackController interface {
	UpsertPlaybackState(ctx context.Context, state *PlaybackState) error
}

type RestrictedPlaybackSkipper interface {
	SkipRestrictedSong(ctx context.Context, roomID string, songID string) (*PlaybackState, error)
}

type SongPlaybackRestrictionUpdater interface {
	UpdateSongPlaybackRestriction(
		ctx context.Context,
		roomID string,
		songID string,
		restriction string,
	) error
}

type PlaybackFailureStorage interface {
	PlaybackFetcher
	RestrictedPlaybackSkipper
	SongsFetcher
	SongPlaybackRestrictionUpdater
}

// ExpiredPlaybackProcessor defines interfaces needed for background room playback automation
type ExpiredPlaybackProcessor interface {
	ProcessNextExpiredPlayback(ctx context.Context) (*PlaybackState, error)
}

type ExpiredPlaybackSongFetcher interface {
	ExpiredPlaybackProcessor
	SongsFetcher
}

// AbandonedHostProcessor defines interfaces needed for background host management
type AbandonedHostProcessor interface {
	ProcessNextAbandonedHost(ctx context.Context) (*RoomHostInfo, error)
}

const RoomActionPlay = "play"
const RoomActionPause = "pause"
const RoomActionSeek = "seek"
const RoomActionSkip = "skip"
const RoomActionVote = "vote"
