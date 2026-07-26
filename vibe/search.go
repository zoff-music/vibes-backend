package vibe

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"sort"
	"strings"
	"unicode"
)

// MusicTrack represents a generic music track
type MusicTrack struct {
	ID              string `json:"id"`
	Source          string `json:"source"`
	ProviderURL     string `json:"providerUrl,omitempty"`
	Title           string `json:"title"`
	ChannelTitle    string `json:"channelTitle,omitempty"`
	ThumbnailURL    string `json:"thumbnailUrl"`
	Duration        string `json:"duration,omitempty"` // ISO 8601 duration
	DurationSeconds int    `json:"durationSeconds,omitempty"`
	ViewCount       uint64 `json:"viewCount,omitempty"`
	LikeCount       uint64 `json:"likeCount,omitempty"`
}

type CachedSearch struct {
	Query  string       `json:"query"`
	Tracks []MusicTrack `json:"tracks"`
}

type SearchUsage struct {
	Provider  string
	QueryHash string
	Cached    bool
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

func GenerateSearchUsage(
	provider string,
	query string,
	cached bool,
) SearchUsage {
	normalizedQuery := NormalizeSearch(query)
	hash := sha256.Sum256([]byte(normalizedQuery))

	return SearchUsage{
		Provider:  provider,
		QueryHash: hex.EncodeToString(hash[:]),
		Cached:    cached,
	}
}

func NormalizeSearch(query string) string {
	value := strings.ToLower(html.UnescapeString(query))
	allTokens := make([]string, 0)
	tokens := make([]string, 0)
	var token strings.Builder
	for _, character := range value {
		if unicode.IsLetter(character) || unicode.IsNumber(character) {
			token.WriteRune(character)
			continue
		}
		if token.Len() == 0 {
			continue
		}

		word := token.String()
		allTokens = append(allTokens, word)
		if !isSearchNoise(word) {
			tokens = append(tokens, word)
		}
		token.Reset()
	}
	if token.Len() > 0 {
		word := token.String()
		allTokens = append(allTokens, word)
		if !isSearchNoise(word) {
			tokens = append(tokens, word)
		}
	}
	if len(tokens) == 0 {
		tokens = allTokens
	}

	sort.Strings(tokens)
	normalizedTokens := make([]string, 0, len(tokens))
	for _, current := range tokens {
		if len(normalizedTokens) > 0 &&
			normalizedTokens[len(normalizedTokens)-1] == current {
			continue
		}
		normalizedTokens = append(normalizedTokens, current)
	}

	return strings.Join(normalizedTokens, " ")
}

func isSearchNoise(value string) bool {
	switch value {
	case "4k", "a", "an", "and", "audio", "feat", "featuring", "ft", "hd",
		"lyric", "lyrics", "music", "official", "the", "video", "visualizer":
		return true
	default:
		return false
	}
}

func (t GeneratedTrack) MusicTrack() MusicTrack {
	return MusicTrack{
		ID:              t.YouTubeID,
		Source:          SourceTypeYouTube,
		ProviderURL:     fmt.Sprintf("https://www.youtube.com/watch?v=%s", t.YouTubeID),
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
}

type MusicTrackFetcher interface {
	GetTrack(ctx context.Context, id string) (*MusicTrack, error)
}

type MusicTrackResolver interface {
	ResolveTrack(ctx context.Context, providerURL string) (*MusicTrack, error)
}

type CachedSearchFetcher interface {
	GetCachedSearches(
		ctx context.Context,
		source string,
		queries []string,
	) ([]CachedSearch, error)
}

type CachedSearchCreator interface {
	CacheSearches(
		ctx context.Context,
		source string,
		searches []CachedSearch,
	) error
}

type CachedSearchFetcherCreator interface {
	CachedSearchFetcher
	CachedSearchCreator
}

type SearchUsageCreator interface {
	CreateSearchUsages(ctx context.Context, usages []SearchUsage) error
}

const SourceTypeYouTube = "youtube"
const SourceTypeSpotify = "spotify"
const SourceTypeSoundCloud = "soundcloud"
