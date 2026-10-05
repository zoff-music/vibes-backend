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

func (s SkipPlaylistItemResult) ToSkipSongResult() *SkipSongResult {
	var next *Song
	if s.NextPlaylistItem != nil {
		next = s.NextPlaylistItem.ToSong()
	}

	var playback *PlaybackState
	if s.Playback != nil {
		playback = s.Playback.ToPlaybackState()
	}

	return &SkipSongResult{
		Action:         s.Action,
		Skipped:        s.Skipped,
		Voted:          s.Voted,
		AlreadyVoted:   s.AlreadyVoted,
		CurrentVotes:   s.CurrentVotes,
		RequiredVotes:  s.RequiredVotes,
		NextSong:       next,
		Playback:       playback,
		PreviousSongID: s.PreviousPlaylistItemID,
	}
}

type RoomV2Skipper interface {
	RoomV2Fetcher
	SessionProfileFetcherCreator
	PlaybackV2Fetcher
	PlaylistItemsFetcher
	SkipPlaylistItem(ctx context.Context, roomID, userID string) (*SkipPlaylistItemResult, error)
}
