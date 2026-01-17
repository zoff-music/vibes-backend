package vibe

import (
	"context"
	"time"
)

// PlaybackState represents the current playback state of a room
type PlaybackState struct {
	RoomID        string    `json:"-"`
	CurrentSongID *string   `json:"currentSongId"`
	CurrentSong   *Song     `json:"currentSong,omitempty"`
	IsPlaying     bool      `json:"isPlaying"`
	PositionMs    int64     `json:"positionMs"`
	UpdatedAt     time.Time `json:"updatedAt"`
	ServerTimeMs  int64     `json:"serverTimeMs"`
}

// RoomAction represents a playback action
type RoomAction string

const (
	RoomActionPlay  RoomAction = "play"
	RoomActionPause RoomAction = "pause"
	RoomActionSeek  RoomAction = "seek"
	RoomActionSkip  RoomAction = "skip"
	RoomActionVote  RoomAction = "vote"
)

// RoomActionRequest is the request payload for room actions.
type RoomActionRequest struct {
	Action     RoomAction `json:"action"`
	PositionMs int64      `json:"positionMs,omitempty"`
}

// SkipVote tracks skip votes for a song
type SkipVote struct {
	SongID string
	UserID string
}

// PlaybackFetcher fetches playback state
type PlaybackFetcher interface {
	GetPlaybackState(ctx context.Context, roomID string) (*PlaybackState, error)
}

// PlaybackController controls playback
type PlaybackController interface {
	UpsertPlaybackState(ctx context.Context, state *PlaybackState) error
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
}

// RoomActioner performs room actions and related state updates.
type RoomActioner interface {
	RoomFetcher
	PlaybackController
	PlaybackFetcher
	SongsModifier
	SkipVoteFetcher
	SkipVoteManager
	UserFetcher
}
