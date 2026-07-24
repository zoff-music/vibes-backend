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
	DurationSeconds int        `json:"-"`
	ViewCount       uint64     `json:"-"`
	LikeCount       uint64     `json:"-"`
}

type CachedYouTubeSearch struct {
	Query  string               `json:"query"`
	Tracks []CachedYouTubeTrack `json:"tracks"`
}

type CachedYouTubeTrack struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	ChannelTitle    string `json:"channelTitle"`
	ThumbnailURL    string `json:"thumbnailUrl"`
	Duration        string `json:"duration"`
	DurationSeconds int    `json:"durationSeconds"`
	ViewCount       uint64 `json:"viewCount"`
	LikeCount       uint64 `json:"likeCount"`
}

func (s *CachedYouTubeSearch) MusicTracks() []MusicTrack {
	tracks := make([]MusicTrack, 0, len(s.Tracks))
	for index := range s.Tracks {
		track := s.Tracks[index].MusicTrack()
		tracks = append(tracks, track)
	}

	return tracks
}

func (s *CachedYouTubeSearch) SetMusicTracks(tracks []MusicTrack) {
	s.Tracks = make([]CachedYouTubeTrack, 0, len(tracks))
	for index := range tracks {
		s.Tracks = append(
			s.Tracks,
			tracks[index].CachedYouTubeTrack(),
		)
	}
}

func (t *MusicTrack) CachedYouTubeTrack() CachedYouTubeTrack {
	return CachedYouTubeTrack{
		ID:              t.ID,
		Title:           t.Title,
		ChannelTitle:    t.ChannelTitle,
		ThumbnailURL:    t.ThumbnailURL,
		Duration:        t.Duration,
		DurationSeconds: t.DurationSeconds,
		ViewCount:       t.ViewCount,
		LikeCount:       t.LikeCount,
	}
}

func (t *GeneratedTrack) CachedYouTubeTrack() CachedYouTubeTrack {
	return CachedYouTubeTrack{
		ID:              t.YouTubeID,
		Title:           t.Title,
		ChannelTitle:    t.Artist,
		ThumbnailURL:    t.ThumbnailURL,
		Duration:        durationISO8601(t.Duration),
		DurationSeconds: t.Duration,
		ViewCount:       t.ViewCount,
		LikeCount:       t.LikeCount,
	}
}

func (t *CachedYouTubeTrack) MusicTrack() MusicTrack {
	return MusicTrack{
		ID:              t.ID,
		Source:          SourceTypeYouTube,
		Title:           t.Title,
		ChannelTitle:    t.ChannelTitle,
		ThumbnailURL:    t.ThumbnailURL,
		Duration:        t.Duration,
		DurationSeconds: t.DurationSeconds,
		ViewCount:       t.ViewCount,
		LikeCount:       t.LikeCount,
	}
}

func (t *CachedYouTubeTrack) GeneratedTrack(query string) GeneratedTrack {
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

type CachedYouTubeSearchFetcher interface {
	GetCachedYouTubeSearches(
		ctx context.Context,
		queries []string,
	) ([]CachedYouTubeSearch, error)
}

type CachedYouTubeSearchCreator interface {
	CacheYouTubeSearches(
		ctx context.Context,
		searches []CachedYouTubeSearch,
	) error
}

type CachedYouTubeSearchFetcherCreator interface {
	CachedYouTubeSearchFetcher
	CachedYouTubeSearchCreator
}

const SourceTypeYouTube SourceType = "youtube"
const SourceTypeSpotify SourceType = "spotify"
const SourceTypeSoundCloud SourceType = "soundcloud"
