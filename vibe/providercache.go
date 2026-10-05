package vibe

import "context"

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
