package vibe

import (
	"context"
	"time"
)

// Song represents a song in the queue
type Song struct {
	ID              string     `json:"id"`
	RoomID          string     `json:"-"`
	SourceType      SourceType `json:"sourceType"`
	SourceID        string     `json:"sourceId"`
	Title           string     `json:"title"`
	Artist          string     `json:"artist,omitempty"`
	ThumbnailURL    string     `json:"thumbnailUrl"`
	Duration        int        `json:"duration"`
	AddedBy         string     `json:"addedBy"`
	AddedByNickname string     `json:"addedByNickname,omitempty"`
	AddedAt         time.Time  `json:"addedAt"`
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
	AddSong(ctx context.Context, song *Song) (*Song, error)
}

// SongRemover removes songs from the queue
type SongRemover interface {
	RemoveSong(ctx context.Context, roomID, songID string) error
}

// SongVoter votes for a song
type SongVoter interface {
	VoteSong(ctx context.Context, roomID, songID, userID string) error
}

// SongController combines interfaces needed for managing songs
type SongController interface {
	SongAdder
	SongRemover
	SongVoter
	SongsFetcher
	PlaybackController
}
