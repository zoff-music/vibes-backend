package vibe

import "context"

// SkipVote tracks skip votes for a song
type SkipVote struct {
	SongID string
	UserID string
}

// SkipVoteFetcher fetches skip votes
type SkipVoteFetcher interface {
	GetSkipVotes(ctx context.Context, roomID, songID string) ([]SkipVote, error)
	HasUserVoted(ctx context.Context, roomID, songID, userID string) (bool, error)
}

// SkipVoteManager manages skip votes
type SkipVoteManager interface {
	AddSkipVote(ctx context.Context, roomID, songID, userID string) error
	ClearSkipVotes(ctx context.Context, roomID, songID string) error
	VoteToSkip(ctx context.Context, roomID, userID string) (*PlaybackState, error)
}

// RoomSkipper defines actions related to skipping tracks
type RoomSkipper interface {
	RoomFetcher
	SongsFetcher
	SkipVoteManager
	SkipTrack(ctx context.Context, roomID string, userID string) (*PlaybackState, error)
}
