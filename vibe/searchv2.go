package vibe

import (
	"context"
)

// ProviderItem describes a provider result before it becomes a room entry.
type ProviderItem struct {
	ID                  string `json:"id"`
	Source              string `json:"source"`
	ProviderURL         string `json:"providerUrl,omitempty"`
	Title               string `json:"title"`
	Publisher           string `json:"publisher,omitempty"`
	ThumbnailURL        string `json:"thumbnailUrl"`
	Duration            string `json:"duration,omitempty"`
	DurationSeconds     int    `json:"durationSeconds,omitzero"`
	ViewCount           uint64 `json:"viewCount,omitzero"`
	LikeCount           uint64 `json:"likeCount,omitzero"`
	PlaybackRestriction string `json:"playbackRestriction,omitempty"`
}

func (p *ProviderItem) IsEmpty() bool {
	return p.ID == ""
}

func (p ProviderItem) ToMusicTrack() *MusicTrack {
	return &MusicTrack{
		ID:                  p.ID,
		Source:              p.Source,
		ProviderURL:         p.ProviderURL,
		Title:               p.Title,
		ChannelTitle:        p.Publisher,
		ThumbnailURL:        p.ThumbnailURL,
		Duration:            p.Duration,
		DurationSeconds:     p.DurationSeconds,
		ViewCount:           p.ViewCount,
		LikeCount:           p.LikeCount,
		PlaybackRestriction: p.PlaybackRestriction,
	}
}

type ProviderItemFetcher interface {
	GetProviderItem(ctx context.Context, id string) (*ProviderItem, error)
}

type ProviderItemsSearcher interface {
	SearchProviderItems(ctx context.Context, query string) ([]ProviderItem, error)
}

type ProviderItemResolver interface {
	ResolveProviderItem(ctx context.Context, providerURL string) (*ProviderItem, error)
}

func (t MusicTrack) ToProviderItem() *ProviderItem {
	return &ProviderItem{
		ID:                  t.ID,
		Source:              t.Source,
		ProviderURL:         t.ProviderURL,
		Title:               t.Title,
		Publisher:           t.ChannelTitle,
		ThumbnailURL:        t.ThumbnailURL,
		Duration:            t.Duration,
		DurationSeconds:     t.DurationSeconds,
		ViewCount:           t.ViewCount,
		LikeCount:           t.LikeCount,
		PlaybackRestriction: t.PlaybackRestriction,
	}
}

type RoomSearchUsageCreator interface {
	RoomV2Fetcher
	SearchUsageCreator
}

type CachedProviderSearch struct {
	Query string         `json:"query"`
	Items []ProviderItem `json:"items"`
}

func (s CachedProviderSearch) GetProviderItems() []ProviderItem {
	items := append([]ProviderItem{}, s.Items...)

	return items
}

type CachedProviderItemKey struct {
	Provider string
	ID       string
}

type CachedProviderSearchFetcherCreator interface {
	GetCachedProviderSearches(ctx context.Context, source string, queries []string) ([]CachedProviderSearch, error)
	CacheProviderSearches(ctx context.Context, source string, searches []CachedProviderSearch) error
}

type CachedProviderItemFetcher interface {
	GetCachedProviderItem(ctx context.Context, source string, sourceID string) (*ProviderItem, error)
}

type CachedProviderItemsFetcher interface {
	GetCachedProviderItems(ctx context.Context, keys []CachedProviderItemKey) ([]ProviderItem, error)
}

type CachedProviderItemsCreator interface {
	CacheProviderItems(ctx context.Context, source string, items []ProviderItem) error
}

type CachedProviderItemFetcherCreator interface {
	CachedProviderItemFetcher
	CachedProviderItemsCreator
}

type CachedProviderSearchItemFetcherCreator interface {
	CachedProviderSearchFetcherCreator
	CachedProviderItemsCreator
}

type ProviderSearchCache interface {
	CachedProviderSearchItemFetcherCreator
	ProviderQuotaResetFetcherCreator
}

type CachedProviderItemRoomEventNotifier interface {
	CachedProviderItemFetcher
	RoomEventV3Notifier
}
