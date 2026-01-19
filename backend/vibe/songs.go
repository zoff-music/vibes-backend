package vibe

import (
	"context"
	"errors"
	"time"
)

// ErrAlreadyVoted is returned when a user tries to vote for a song they have already voted for
var ErrAlreadyVoted = errors.New("already voted")

// Song represents a song in the queue
type Song struct {
	ID              string     `json:"id"`
	RoomID          string     `json:"-"`
	SourceType      SourceType `json:"sourceType"`
	SourceID        string     `json:"sourceId"`
	Title           string     `json:"title"`
	Artist          *string    `json:"artist,omitempty"`
	ThumbnailURL    string     `json:"thumbnailUrl"`
	Duration        int        `json:"duration"`
	AddedBy         string     `json:"addedBy"`
	AddedByNickname *string    `json:"addedByNickname,omitempty"`
	AddedAt         time.Time  `json:"addedAt"`
	Position        int        `json:"position"`
	VoteCount       int        `json:"voteCount"`
}

// AddSongRequest is the request payload for adding a song.
type AddSongRequest struct {
	SourceType SourceType `json:"sourceType"`
	SourceID   string     `json:"sourceId"`
	Title      string     `json:"title"`
	Artist     string     `json:"artist,omitempty"`
	Thumbnail  string     `json:"thumbnailUrl"`
	Duration   int        `json:"duration"`
	AddedBy    string     `json:"addedBy"`
}

// ReorderSongsRequest is the request payload for reordering songs.
type ReorderSongsRequest struct {
	NewPosition int `json:"newPosition"`
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
	UpsertPlaybackState(ctx context.Context, state *PlaybackState) error
	GetMaxPosition(ctx context.Context, roomID string) (int, error)
	AddSong(ctx context.Context, song *Song) (*Song, error)
	GetSongs(ctx context.Context, roomID string) ([]Song, error)
}

// SongRemoverGetter removes and gets songs from the queue
type SongRemoverGetter interface {
	GetSongs(ctx context.Context, roomID string) ([]Song, error)
	RemoveSong(ctx context.Context, roomID, songID string) error
}

// SongsReorderer reorders songs in the queue
type SongsReorderer interface {
	ReorderSongs(ctx context.Context, roomID, songID string, newPosition int) error
	GetSongs(ctx context.Context, roomID string) ([]Song, error)
}

// SongVoter votes for a song
type SongVoter interface {
	VoteSong(ctx context.Context, roomID, songID, userID string) error
	GetSongs(ctx context.Context, roomID string) ([]Song, error)
}
