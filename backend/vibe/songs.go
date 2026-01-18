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
	Artist          *string    `json:"artist,omitempty"`
	ThumbnailURL    string     `json:"thumbnailUrl"`
	Duration        int        `json:"duration"`
	AddedBy         string     `json:"addedBy"`
	AddedByNickname *string    `json:"addedByNickname,omitempty"`
	AddedAt         time.Time  `json:"addedAt"`
	Position        int        `json:"position"`
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
	GetSong(ctx context.Context, roomID, songID string) (*Song, error)
}

// SongsModifier modifies the song queue
type SongsModifier interface {
	GetSongs(ctx context.Context, roomID string) ([]Song, error)
	GetSong(ctx context.Context, roomID, songID string) (*Song, error)
	AddSong(ctx context.Context, song *Song) (*Song, error)
	RemoveSong(ctx context.Context, roomID, songID string) error
	ReorderSongs(ctx context.Context, roomID, songID string, newPosition int) error
	GetNextSong(ctx context.Context, roomID string, currentPosition int) (*Song, error)
	GetMaxPosition(ctx context.Context, roomID string) (int, error)
}
