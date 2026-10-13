package vibe

import "context"

type SkipPlaylistItemResult struct {
	Action                 string           `json:"action"`
	Skipped                bool             `json:"skipped"`
	Voted                  bool             `json:"voted"`
	AlreadyVoted           bool             `json:"alreadyVoted"`
	CurrentVotes           int              `json:"currentVotes"`
	RequiredVotes          int              `json:"requiredVotes"`
	NextPlaylistItem       *PlaylistItem    `json:"nextPlaylistItem"`
	Playback               *PlaybackStateV2 `json:"playback"`
	PreviousPlaylistItemID string           `json:"-"`
}

type RoomV2Skipper interface {
	RoomV2Fetcher
	SessionProfileFetcherCreator
	PlaybackV2Fetcher
	PlaylistItemsFetcher
	SkipPlaylistItem(ctx context.Context, roomID, userID string) (*SkipPlaylistItemResult, error)
}
