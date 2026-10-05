package redis

import (
	"context"
	"fmt"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) GetCachedSearches(ctx context.Context, source string, queries []string) ([]vibe.CachedSearch, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedSearches")
	defer span.End()

	searches, err := c.GetCachedProviderSearches(ctx, source, queries)
	if err != nil {
		return nil, fmt.Errorf("error fetching legacy cached searches: %w", err)
	}

	legacy := make([]vibe.CachedSearch, 0, len(searches))
	for _, search := range searches {
		tracks := make([]vibe.MusicTrack, 0, len(search.Items))
		for _, item := range search.Items {
			tracks = append(tracks, *item.ToMusicTrack())
		}

		legacy = append(legacy, vibe.CachedSearch{Query: search.Query, Tracks: tracks})
	}

	return legacy, nil
}

func (c *Client) CacheSearches(ctx context.Context, source string, searches []vibe.CachedSearch) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheSearches")
	defer span.End()

	canonical := make([]vibe.CachedProviderSearch, 0, len(searches))
	for _, search := range searches {
		items := make([]vibe.ProviderItem, 0, len(search.Tracks))
		for _, track := range search.Tracks {
			items = append(items, *track.ToProviderItem())
		}

		canonical = append(canonical, vibe.CachedProviderSearch{Query: search.Query, Items: items})
	}

	err := c.CacheProviderSearches(ctx, source, canonical)
	if err != nil {
		return fmt.Errorf("error caching legacy searches: %w", err)
	}

	return nil
}

func (c *Client) GetCachedMusicTrack(ctx context.Context, source string, sourceID string) (*vibe.MusicTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedMusicTrack")
	defer span.End()

	item, err := c.GetCachedProviderItem(ctx, source, sourceID)
	if err != nil {
		return nil, fmt.Errorf("error fetching legacy cached item: %w", err)
	}

	legacy := item.ToMusicTrack()
	return legacy, nil
}

func (c *Client) GetCachedMusicTracks(ctx context.Context, keys []vibe.CachedMusicTrackKey) ([]vibe.MusicTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedMusicTracks")
	defer span.End()

	canonicalKeys := make([]vibe.CachedProviderItemKey, 0, len(keys))
	for _, key := range keys {
		canonicalKeys = append(canonicalKeys, vibe.CachedProviderItemKey{Provider: key.Provider, ID: key.ID})
	}

	items, err := c.GetCachedProviderItems(ctx, canonicalKeys)
	if err != nil {
		return nil, fmt.Errorf("error fetching legacy cached items: %w", err)
	}

	legacy := make([]vibe.MusicTrack, 0, len(items))
	for _, item := range items {
		legacy = append(legacy, *item.ToMusicTrack())
	}

	return legacy, nil
}

func (c *Client) CacheMusicTracks(ctx context.Context, source string, tracks []vibe.MusicTrack) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheMusicTracks")
	defer span.End()

	items := make([]vibe.ProviderItem, 0, len(tracks))
	for _, track := range tracks {
		items = append(items, *track.ToProviderItem())
	}

	err := c.CacheProviderItems(ctx, source, items)
	if err != nil {
		return fmt.Errorf("error caching legacy items: %w", err)
	}

	return nil
}
