package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) GetCachedProviderSearches(
	ctx context.Context,
	source string,
	queries []string,
) ([]vibe.CachedProviderSearch, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedProviderSearches")
	defer span.End()

	searches := make([]vibe.CachedProviderSearch, 0, len(queries))
	if c.Redis == nil || len(queries) == 0 {
		return searches, nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection in GetCachedProviderSearches: %w", err)
	}
	defer connection.Close()

	keys := make(redis.Args, 0, len(queries))
	cachedQueries := make([]string, 0, len(queries))
	for _, query := range queries {
		key := c.providerSearchCacheKey(source, query)
		if key == "" {
			continue
		}
		keys = append(keys, key)
		cachedQueries = append(cachedQueries, query)
	}
	if len(keys) == 0 {
		return searches, nil
	}

	values, err := redis.Values(redis.DoContext(connection, cctx, "MGET", keys...))
	if err != nil {
		return nil, fmt.Errorf("error getting cached searches in GetCachedProviderSearches: %w", err)
	}

	for index, value := range values {
		if value == nil {
			continue
		}

		body, err := redis.Bytes(value, nil)
		if err != nil {
			return nil, fmt.Errorf("error reading cached search in GetCachedProviderSearches: %w", err)
		}

		var search vibe.CachedProviderSearch
		err = json.Unmarshal(body, &search)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling cached search in GetCachedProviderSearches: %w", err)
		}
		search.Query = cachedQueries[index]
		searches = append(searches, search)
	}

	return searches, nil
}

func (c *Client) CacheProviderSearches(
	ctx context.Context,
	source string,
	searches []vibe.CachedProviderSearch,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheProviderSearches")
	defer span.End()

	if c.Redis == nil || len(searches) == 0 {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return fmt.Errorf("error getting redis connection in CacheProviderSearches: %w", err)
	}
	defer connection.Close()

	commandCount := 0
	for _, search := range searches {
		key := c.providerSearchCacheKey(source, search.Query)
		if key == "" {
			continue
		}

		body, err := json.Marshal(search)
		if err != nil {
			return fmt.Errorf("error marshaling cached search in CacheProviderSearches: %w", err)
		}

		err = connection.Send(
			"SET",
			key,
			body,
			"EX",
			int(searchCacheExpiration.Seconds()),
		)
		if err != nil {
			return fmt.Errorf("error queueing cached search in CacheProviderSearches: %w", err)
		}
		commandCount++
	}
	if commandCount == 0 {
		return nil
	}

	err = connection.Flush()
	if err != nil {
		return fmt.Errorf("error flushing cached searches in CacheProviderSearches: %w", err)
	}
	for range commandCount {
		_, err = redis.ReceiveContext(connection, cctx)
		if err != nil {
			return fmt.Errorf("error storing cached search in CacheProviderSearches: %w", err)
		}
	}

	return nil
}

func (c *Client) GetCachedProviderItem(
	ctx context.Context,
	source string,
	sourceID string,
) (*vibe.ProviderItem, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedProviderItem")
	defer span.End()

	if c.Redis == nil || source == "" || sourceID == "" {
		return &vibe.ProviderItem{}, nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection in GetCachedProviderItem: %w", err)
	}
	defer connection.Close()

	body, err := redis.Bytes(redis.DoContext(
		connection,
		cctx,
		"GET",
		c.providerItemCacheKey(source, sourceID),
	))
	if errors.Is(err, redis.ErrNil) {
		return &vibe.ProviderItem{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting cached provider item in GetCachedProviderItem: %w", err)
	}

	var item vibe.ProviderItem
	err = json.Unmarshal(body, &item)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling cached provider item in GetCachedProviderItem: %w", err)
	}

	return &item, nil
}

func (c *Client) GetCachedProviderItems(
	ctx context.Context,
	keys []vibe.CachedProviderItemKey,
) ([]vibe.ProviderItem, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedProviderItems")
	defer span.End()

	items := make([]vibe.ProviderItem, len(keys))
	if c.Redis == nil || len(keys) == 0 {
		return items, nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection in GetCachedProviderItems: %w", err)
	}
	defer connection.Close()

	cacheKeys := make(redis.Args, 0, len(keys))
	for _, key := range keys {
		cacheKeys = append(cacheKeys, c.providerItemCacheKey(key.Provider, key.ID))
	}

	values, err := redis.Values(redis.DoContext(connection, cctx, "MGET", cacheKeys...))
	if err != nil {
		return nil, fmt.Errorf("error getting cached provider items in GetCachedProviderItems: %w", err)
	}

	for index, value := range values {
		if value == nil {
			continue
		}

		body, err := redis.Bytes(value, nil)
		if err != nil {
			return nil, fmt.Errorf("error reading cached provider item in GetCachedProviderItems: %w", err)
		}

		err = json.Unmarshal(body, &items[index])
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling cached provider item in GetCachedProviderItems: %w", err)
		}
	}

	return items, nil
}

func (c *Client) CacheProviderItems(
	ctx context.Context,
	source string,
	items []vibe.ProviderItem,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheProviderItems")
	defer span.End()

	if c.Redis == nil || source == "" || len(items) == 0 {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return fmt.Errorf("error getting redis connection in CacheProviderItems: %w", err)
	}
	defer connection.Close()

	commandCount := 0
	for _, item := range items {
		if item.ID == "" {
			continue
		}

		body, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("error marshaling cached provider item in CacheProviderItems: %w", err)
		}

		err = connection.Send(
			"SET",
			c.providerItemCacheKey(source, item.ID),
			body,
			"EX",
			int(searchCacheExpiration.Seconds()),
		)
		if err != nil {
			return fmt.Errorf("error queueing cached provider item in CacheProviderItems: %w", err)
		}
		commandCount++
	}
	if commandCount == 0 {
		return nil
	}

	err = connection.Flush()
	if err != nil {
		return fmt.Errorf("error flushing cached provider items in CacheProviderItems: %w", err)
	}
	for range commandCount {
		_, err = redis.ReceiveContext(connection, cctx)
		if err != nil {
			return fmt.Errorf("error storing cached provider item in CacheProviderItems: %w", err)
		}
	}

	return nil
}

func (c *Client) providerSearchCacheKey(source string, query string) string {
	normalizedQuery := vibe.NormalizeSearch(query)
	if source == "" || normalizedQuery == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(normalizedQuery))
	key := c.getKeyWithPrefix(
		"search:v3:" + string(source) + ":" + hex.EncodeToString(hash[:]),
	)

	return key
}

func (c *Client) providerItemCacheKey(source string, sourceID string) string {
	hash := sha256.Sum256([]byte(sourceID))
	key := c.getKeyWithPrefix(
		"provider-item:v3:" + source + ":" + hex.EncodeToString(hash[:]),
	)
	return key
}

const searchCacheExpiration = 3 * 24 * time.Hour
