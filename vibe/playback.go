package vibe

import (
	"context"
)

// RoomActionRequest is the request payload for room actions.
type RoomActionRequest struct {
	Action     string `json:"action"`
	PositionMs int    `json:"positionMs,omitzero"`
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

type AbandonedHostProcessorProfileFetcher interface {
	AbandonedHostProcessor
	SessionProfileFetcherCreator
}

const RoomActionPlay = "play"
const RoomActionPause = "pause"
const RoomActionSeek = "seek"
const RoomActionSkip = "skip"
const RoomActionVote = "vote"
