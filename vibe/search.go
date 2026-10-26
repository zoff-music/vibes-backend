package vibe

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"strings"
	"time"
)

func IsLiveVideo(source string, duration int) bool {
	return source == SourceTypeYouTube && duration <= 0
}

type SearchUsage struct {
	RoomID    string
	Provider  string
	QueryHash string
	Cached    bool
}

func GenerateSearchUsage(
	provider string,
	query string,
	cached bool,
) (*SearchUsage, error) {
	normalizedQuery := NormalizeSearch(query)
	hash := sha256.Sum256([]byte(normalizedQuery))

	return &SearchUsage{
		Provider:  provider,
		QueryHash: hex.EncodeToString(hash[:]),
		Cached:    cached,
	}, nil
}

func NormalizeSearch(query string) string {
	value := strings.ToLower(html.UnescapeString(query))
	normalizedQuery := strings.Join(strings.Fields(value), " ")

	return normalizedQuery
}

func (t GeneratedPlaylistItem) ToProviderItem() (*ProviderItem, error) {
	return &ProviderItem{
		CategoryID:          t.CategoryID,
		ID:                  t.YouTubeID,
		Source:              SourceTypeYouTube,
		ProviderURL:         fmt.Sprintf("https://www.youtube.com/watch?v=%s", t.YouTubeID),
		Title:               t.Title,
		Publisher:           t.Publisher,
		ThumbnailURL:        t.ThumbnailURL,
		Duration:            durationISO8601(t.Duration),
		DurationSeconds:     t.Duration,
		ViewCount:           t.ViewCount,
		LikeCount:           t.LikeCount,
		PlaybackRestriction: t.PlaybackRestriction,
	}, nil
}

func (t ProviderItem) ToGeneratedPlaylistItem(query string) (*GeneratedPlaylistItem, error) {
	return &GeneratedPlaylistItem{
		CategoryID:          t.CategoryID,
		Publisher:           t.Publisher,
		Title:               t.Title,
		YouTubeID:           t.ID,
		ThumbnailURL:        t.ThumbnailURL,
		Duration:            t.DurationSeconds,
		ViewCount:           t.ViewCount,
		LikeCount:           t.LikeCount,
		SearchQuery:         query,
		PlaybackRestriction: t.PlaybackRestriction,
	}, nil
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

type ProviderQuotaResetFetcher interface {
	GetProviderQuotaReset(
		ctx context.Context,
		provider string,
		operation string,
	) (time.Time, error)
}

type ProviderQuotaResetCreator interface {
	CacheProviderQuotaReset(
		ctx context.Context,
		provider string,
		operation string,
		resetAt time.Time,
	) error
}

type ProviderQuotaResetFetcherCreator interface {
	ProviderQuotaResetFetcher
	ProviderQuotaResetCreator
}

type SearchUsageCreator interface {
	CreateSearchUsages(ctx context.Context, usages []SearchUsage) error
}

type SearchClaim struct {
	Key   string
	Token string
}

func (c *SearchClaim) IsEmpty() bool {
	return c.Token == ""
}

type SearchClaimCreator interface {
	GetOrClaimProviderSearch(ctx context.Context, source string, query string, roomType RoomType) (*ProviderSearchResult, error)
	ReleaseSearch(ctx context.Context, claim SearchClaim) error
}

type ProviderSearchResult struct {
	Searches []CachedProviderSearch
	Claim    SearchClaim
}

const SourceTypeYouTube = "youtube"
const SourceTypeSoundCloud = "soundcloud"
const ProviderQuotaOperationSearch = "search"
