package vibe

import "context"

// SkipVote tracks skip votes for a song
type SkipVote struct {
	SongID string
	UserID string
}

// SkipSongResult describes the result of a skip request.
type SkipSongResult struct {
	Action        RoomAction     `json:"action"`
	Skipped       bool           `json:"skipped"`
	Voted         bool           `json:"voted"`
	AlreadyVoted  bool           `json:"alreadyVoted"`
	CurrentVotes  int            `json:"currentVotes"`
	RequiredVotes int            `json:"requiredVotes"`
	NextSong      *Song          `json:"nextSong"`
	Playback      *PlaybackState `json:"playback"`
}

// SkipVoteUpdate describes a skip vote event.
type SkipVoteUpdate struct {
	UserID        string `json:"userId"`
	SongID        string `json:"songId"`
	CurrentVotes  int    `json:"currentVotes"`
	RequiredVotes int    `json:"requiredVotes"`
}

// SkipVoteFetcher fetches skip votes
type SkipVoteFetcher interface {
	GetSkipVotes(ctx context.Context, roomID, songID string) ([]SkipVote, error)
	HasUserVoted(ctx context.Context, roomID, songID, userID string) (bool, error)
}

// SkipVoteAdder adds skip votes.
type SkipVoteAdder interface {
	AddSkipVote(ctx context.Context, roomID, songID, userID string) error
}

// RoomSkipper defines actions related to skipping tracks
type RoomSkipper interface {
	GetSongs(ctx context.Context, roomID string) ([]Song, error)
	SkipSong(ctx context.Context, roomID string, userID string, isAdmin bool) (*SkipSongResult, error)
}
