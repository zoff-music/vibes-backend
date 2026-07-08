package vibe

import "context"

// SourceType represents the type of music source
type SourceType string

const SourceTypeYouTube SourceType = "youtube"
const SourceTypeSpotify SourceType = "spotify"
const SourceTypeSoundCloud SourceType = "soundcloud"

// MusicTrack represents a generic music track
type MusicTrack struct {
	ID           string     `json:"id"`
	Source       SourceType `json:"source"`
	Title        string     `json:"title"`
	ChannelTitle string     `json:"channelTitle,omitempty"`
	ThumbnailURL string     `json:"thumbnailUrl"`
	Duration     string     `json:"duration,omitempty"` // ISO 8601 duration
}

// MusicSearcher searches for music
type MusicSearcher interface {
	Search(ctx context.Context, query string) ([]MusicTrack, error)
	GetTrack(ctx context.Context, id string) (*MusicTrack, error)
}
