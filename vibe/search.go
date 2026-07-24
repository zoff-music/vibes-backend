package vibe

import (
	"context"
	"fmt"
)

// SourceType represents the type of music source
type SourceType string

// MusicTrack represents a generic music track
type MusicTrack struct {
	ID              string     `json:"id"`
	Source          SourceType `json:"source"`
	Title           string     `json:"title"`
	ChannelTitle    string     `json:"channelTitle,omitempty"`
	ThumbnailURL    string     `json:"thumbnailUrl"`
	Duration        string     `json:"duration,omitempty"` // ISO 8601 duration
	DurationSeconds int        `json:"durationSeconds,omitempty"`
	ViewCount       uint64     `json:"viewCount,omitempty"`
	LikeCount       uint64     `json:"likeCount,omitempty"`
}

type CachedSearch struct {
	Query  string       `json:"query"`
	Tracks []MusicTrack `json:"tracks"`
}

func (s CachedSearch) GetMusicTracks() []MusicTrack {
	return append([]MusicTrack{}, s.Tracks...)
}

func GenerateCachedSearch(
	query string,
	tracks []MusicTrack,
) CachedSearch {
	return CachedSearch{
		Query:  query,
		Tracks: append([]MusicTrack{}, tracks...),
	}
}

func (t GeneratedTrack) MusicTrack() MusicTrack {
	return MusicTrack{
		ID:              t.YouTubeID,
		Source:          SourceTypeYouTube,
		Title:           t.Title,
		ChannelTitle:    t.Artist,
		ThumbnailURL:    t.ThumbnailURL,
		Duration:        durationISO8601(t.Duration),
		DurationSeconds: t.Duration,
		ViewCount:       t.ViewCount,
		LikeCount:       t.LikeCount,
	}
}

func (t MusicTrack) GeneratedTrack(query string) GeneratedTrack {
	return GeneratedTrack{
		Artist:       t.ChannelTitle,
		Title:        t.Title,
		YouTubeID:    t.ID,
		ThumbnailURL: t.ThumbnailURL,
		Duration:     t.DurationSeconds,
		ViewCount:    t.ViewCount,
		LikeCount:    t.LikeCount,
		SearchQuery:  query,
	}
}

func durationISO8601(seconds int) string {
	hours := seconds / (60 * 60)
	minutes := (seconds % (60 * 60)) / 60
	remainingSeconds := seconds % 60
	duration := "PT"
	if hours > 0 {
		duration += fmt.Sprintf("%dH", hours)
	}
	if minutes > 0 {
		duration += fmt.Sprintf("%dM", minutes)
	}
	if remainingSeconds > 0 || duration == "PT" {
		duration += fmt.Sprintf("%dS", remainingSeconds)
	}

	return duration
}

// MusicSearcher searches for music
type MusicSearcher interface {
	Search(ctx context.Context, query string) ([]MusicTrack, error)
	GetTrack(ctx context.Context, id string) (*MusicTrack, error)
}

type CachedSearchFetcher interface {
	GetCachedSearches(
		ctx context.Context,
		source SourceType,
		queries []string,
	) ([]CachedSearch, error)
}

type CachedSearchCreator interface {
	CacheSearches(
		ctx context.Context,
		source SourceType,
		searches []CachedSearch,
	) error
}

type CachedSearchFetcherCreator interface {
	CachedSearchFetcher
	CachedSearchCreator
}

const SourceTypeYouTube SourceType = "youtube"

const SourceTypeSpotify SourceType = "spotify"

const SourceTypeSoundCloud SourceType = "soundcloud"
