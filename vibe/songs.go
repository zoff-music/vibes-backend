package vibe

import (
	"context"
	"time"
)

// Song represents a song in the queue
type Song struct {
	ID              string    `json:"id"`
	RoomID          string    `json:"-"`
	SourceType      string    `json:"sourceType"`
	SourceID        string    `json:"sourceId"`
	Title           string    `json:"title"`
	Artist          string    `json:"artist,omitempty"`
	ThumbnailURL    string    `json:"thumbnailUrl"`
	Duration        int       `json:"duration"`
	AddedBy         string    `json:"addedBy"`
	AddedByNickname string    `json:"addedByNickname,omitempty"`
	AddedAt         time.Time `json:"addedAt"`
	VoteCount       int       `json:"voteCount"`
}

// AddSongRequest is the request payload for adding a song.
type AddSongRequest struct {
	SourceType string `json:"sourceType"`
	SourceID   string `json:"sourceId"`
	Title      string `json:"title"`
	Artist     string `json:"artist,omitempty"`
	Thumbnail  string `json:"thumbnailUrl"`
	Duration   int    `json:"duration"`
}

// AddSongResult is the result of adding a song or voting on an existing duplicate.
type AddSongResult struct {
	Song    Song   `json:"song"`
	Outcome string `json:"outcome"`
}

type SongMetadataRefresh struct {
	SongID   string
	RoomID   string
	SourceID string
}

// IsEmpty returns true if the song is empty/not found
func (s *Song) IsEmpty() bool {
	return s.ID == ""
}

// SongsFetcher fetches songs from the queue
type SongsFetcher interface {
	GetSongs(ctx context.Context, roomID string) ([]Song, error)
}

// SongAdder adds songs to the queue
type SongAdder interface {
	AddSong(ctx context.Context, song *Song) (*AddSongResult, error)
}

// SongRemover removes songs from the queue
type SongRemover interface {
	RemoveSong(ctx context.Context, roomID, songID string) error
}

type SongMetadataRefreshStorage interface {
	ClaimSongMetadataRefresh(
		ctx context.Context,
		provider string,
		retryAfter time.Duration,
	) (*SongMetadataRefresh, error)
	RefreshSongMetadata(
		ctx context.Context,
		refresh SongMetadataRefresh,
		track MusicTrack,
		refreshInterval time.Duration,
	) error
	DeferSongMetadataRefresh(
		ctx context.Context,
		songID string,
		retryAfter time.Duration,
	) error
	SongRemover
	SongsFetcher
}

// SongVoter votes for a song
type SongVoter interface {
	VoteSong(ctx context.Context, roomID, songID, userID string) error
}

// SongQueueAdder defines the exact operations used when adding a song.
type SongQueueAdder interface {
	SongAdder
	SongsFetcher
	RoomFetcher
	PlaybackController
}

// SongQueueRemover defines the exact operations used when removing a song.
type SongQueueRemover interface {
	SongRemover
	SongsFetcher
	RoomFetcher
}

// SongQueueVoter defines the exact operations used when voting for a song.
type SongQueueVoter interface {
	SongVoter
	SongsFetcher
}

const AddSongOutcomeAdded = "added"
const AddSongOutcomeDuplicateVoted = "duplicate_voted"
const AddSongOutcomeDuplicateAlreadyVoted = "duplicate_already_voted"
