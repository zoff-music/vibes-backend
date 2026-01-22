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

// MaxPositionGetter gets the max position in a queue
type MaxPositionGetter interface {
	GetMaxPosition(ctx context.Context, roomID string) (int, error)
}

// PlaybackStateUpserter upserts playback state
type PlaybackStateUpserter interface {
	UpsertPlaybackState(ctx context.Context, state *PlaybackState) error
}

// SongAdder adds songs to the queue
type SongAdder interface {
	AddSong(ctx context.Context, song *Song) (*Song, error)
}

// SongRemover removes songs from the queue
type SongRemover interface {
	RemoveSong(ctx context.Context, roomID, songID string) error
}

// SongsReorderer reorders songs in the queue
type SongsReorderer interface {
	ReorderSongs(ctx context.Context, roomID, songID string, newPosition int) error
}

// SongVoter votes for a song
type SongVoter interface {
	VoteSong(ctx context.Context, roomID, songID, userID string) error
}

// SongAdderDB combines interfaces needed for adding songs
type SongAdderDB interface {
	SongAdder
	MaxPositionGetter
	SongsFetcher
	PlaybackStateUpserter
}

// SongRemoverDB combines interfaces needed for removing songs
type SongRemoverDB interface {
	SongRemover
	SongsFetcher
}

// SongVoterDB combines interfaces needed for voting on songs
type SongVoterDB interface {
	SongVoter
	SongsFetcher
}

// SongsReordererDB combines interfaces needed for reordering songs
type SongsReordererDB interface {
	SongsReorderer
	SongsFetcher
}
