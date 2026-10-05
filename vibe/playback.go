package vibe

import (
	"context"
	"time"
)

// PlaybackState represents the current playback state of a room
//
// Deprecated: Use PlaybackStateV2 for new code. Retained for legacy API compatibility.
type PlaybackState struct {
	RoomID       string    `json:"-"`
	CurrentSong  *Song     `json:"currentSong"`
	IsPlaying    bool      `json:"isPlaying"`
	PositionMs   int       `json:"positionMs"`
	UpdatedAt    time.Time `json:"updatedAt"`
	ServerTimeMs int       `json:"serverTimeMs"`
}

// PlaybackAdvance describes one automatic queue advance and the resulting playback state.
//
// Deprecated: Use PlaybackAdvanceV2 for new code. Retained for legacy API compatibility.
type PlaybackAdvance struct {
	Playback       *PlaybackState
	PreviousSongID string
}

func (p *PlaybackState) IsEmpty() bool {
	return p.RoomID == ""
}

// RoomActionRequest is the request payload for room actions.
type RoomActionRequest struct {
	Action     string `json:"action"`
	PositionMs int    `json:"positionMs,omitzero"`
}

// Deprecated: Use PlaybackFailureRequestV2 for new code. Retained for legacy API compatibility.
type PlaybackFailureRequest struct {
	SongID string `json:"songId"`
}

type PlaylistItemPlaybackRestrictionUpdater interface {
	UpdatePlaylistItemPlaybackRestriction(
		ctx context.Context,
		roomID string,
		songID string,
		restriction string,
	) error
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
