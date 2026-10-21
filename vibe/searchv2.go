package vibe

import (
	"context"
	"encoding/json"
	"fmt"
)

// ProviderItem describes a provider result before it becomes a room entry.
type ProviderItem struct {
	ID                  string `json:"id"`
	Source              string `json:"source"`
	ProviderURL         string `json:"providerUrl,omitempty"`
	Title               string `json:"title"`
	Publisher           string `json:"publisher,omitempty"`
	CategoryID          string `json:"categoryId,omitempty"`
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

func (p ProviderItem) AllowedInRoom(roomType RoomType) bool {
	if !roomType.AllowsSource(p.Source) {
		return false
	}

	if p.Source != SourceTypeYouTube {
		return true
	}

	if IsLiveVideo(p.Source, p.DurationSeconds) || p.PlaybackRestriction == PlaybackRestrictionEmbedding {
		return false
	}

	return roomType == RoomTypeWatch || p.CategoryID == YouTubeMusicCategoryID
}

type ProviderItemFetcher interface {
	GetProviderItem(ctx context.Context, id string) (*ProviderItem, error)
}

type ProviderItemsSearcher interface {
	SearchProviderItems(ctx context.Context, query string, roomType RoomType) ([]ProviderItem, error)
}

type ProviderItemResolver interface {
	ResolveProviderItem(ctx context.Context, providerURL string) (*ProviderItem, error)
}

type RoomSearchUsageCreator interface {
	RoomV2Fetcher
	SearchUsageCreator
}

type CachedProviderSearch struct {
	Query string         `json:"query"`
	Items []ProviderItem `json:"items"`
}

// MarshalJSON preserves empty collections in the public JSON contract.
func (v CachedProviderSearch) MarshalJSON() ([]byte, error) {
	type payload CachedProviderSearch
	result := payload(v)

	if result.Items == nil {
		result.Items = make([]ProviderItem, 0)
	}

	body, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling CachedProviderSearch: %w", err)
	}

	return body, nil
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
	GetCachedProviderSearches(ctx context.Context, source string, queries []string, roomType RoomType) ([]CachedProviderSearch, error)
	CacheProviderSearches(ctx context.Context, source string, searches []CachedProviderSearch, roomType RoomType) error
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

const YouTubeMusicCategoryID = "10"
