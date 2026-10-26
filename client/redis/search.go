package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) ClaimSearch(ctx context.Context, key string, token string) (*vibe.SearchClaim, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ClaimSearch")
	defer span.End()

	if c.Redis == nil {
		return nil, fmt.Errorf("error claiming search: redis is unavailable")
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection in ClaimSearch: %w", err)
	}
	defer connection.Close()

	_, err = redis.String(redis.DoContext(connection, cctx, "SET", c.getKeyWithPrefix("search-claim:"+key), token, "NX", "EX", 60))
	if errors.Is(err, redis.ErrNil) {
		return &vibe.SearchClaim{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error claiming provider search: %w", err)
	}

	return &vibe.SearchClaim{Key: key, Token: token}, nil
}

func (c *Client) ReleaseSearch(ctx context.Context, claim vibe.SearchClaim) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "ReleaseSearch")
	defer span.End()

	if c.Redis == nil {
		return fmt.Errorf("error releasing search: redis is unavailable")
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return fmt.Errorf("error getting redis connection in ReleaseSearch: %w", err)
	}
	defer connection.Close()

	_, err = redis.DoContext(connection, cctx, "EVAL", releaseSearchScript, 1, c.getKeyWithPrefix("search-claim:"+claim.Key), claim.Token)
	if err != nil {
		return fmt.Errorf("error releasing provider search: %w", err)
	}

	return nil
}

func (c *Client) GetCachedProviderSearches(
	ctx context.Context,
	source string,
	queries []string,
	roomType vibe.RoomType,
) ([]vibe.CachedProviderSearch, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedProviderSearches")
	defer span.End()

	searches, err := c.getProviderSearches(ctx, source, queries, roomType, false, false)
	if err != nil {
		return nil, fmt.Errorf("error reading provider searches: %w", err)
	}

	return searches, nil
}

func (c *Client) GetRetainedProviderSearches(ctx context.Context, source string, queries []string, roomType vibe.RoomType) ([]vibe.CachedProviderSearch, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetRetainedProviderSearches")
	defer span.End()

	searches, err := c.getProviderSearches(ctx, source, queries, roomType, false, true)
	if err != nil {
		return nil, fmt.Errorf("error reading retained provider searches: %w", err)
	}

	return searches, nil
}

func (c *Client) GetCachedProviderResolutions(ctx context.Context, source string, queries []string, roomType vibe.RoomType) ([]vibe.CachedProviderSearch, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedProviderResolutions")
	defer span.End()

	searches, err := c.getProviderSearches(ctx, source, queries, roomType, true, false)
	if err != nil {
		return nil, fmt.Errorf("error reading provider resolutions: %w", err)
	}

	return searches, nil
}

func (c *Client) getProviderSearches(ctx context.Context, source string, queries []string, roomType vibe.RoomType, resolution bool, retained bool) ([]vibe.CachedProviderSearch, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getProviderSearches")
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
		key := c.providerSearchCacheKey(source, query, roomType, resolution)
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
		if search.FetchedAt.IsZero() || time.Since(search.FetchedAt) > retainedSearchExpiration {
			continue
		}
		if !retained && time.Since(search.FetchedAt) > searchCacheExpiration {
			continue
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
	roomType vibe.RoomType,
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
		key := c.providerSearchCacheKey(source, search.Query, roomType, search.Resolution)
		if key == "" {
			continue
		}

		if search.FetchedAt.IsZero() {
			search.FetchedAt = time.Now().UTC()
		}
		retention := retainedSearchExpiration
		if len(search.Items) == 0 {
			retention = emptySearchExpiration
		}
		remaining := time.Until(search.FetchedAt.Add(retention))
		if remaining < time.Second {
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
			int(remaining.Seconds()),
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
			int(providerItemExpiration.Seconds()),
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

func (c *Client) providerSearchCacheKey(source string, query string, roomType vibe.RoomType, resolution bool) string {
	normalizedQuery := vibe.NormalizeSearch(query)
	if source == "" || normalizedQuery == "" || !roomType.IsValid() {
		return ""
	}

	hash := sha256.Sum256([]byte(normalizedQuery))
	version := "v5"
	if source == vibe.SourceTypeSoundCloud {
		version = "v7"
	}
	if resolution {
		version += ":resolution"
	}

	key := c.getKeyWithPrefix(
		"search:" + version + ":" + source + ":" + string(roomType) + ":" + hex.EncodeToString(hash[:]),
	)

	return key
}

func (c *Client) providerItemCacheKey(source string, sourceID string) string {
	hash := sha256.Sum256([]byte(sourceID))
	version := "v4"
	if source == vibe.SourceTypeSoundCloud {
		version = "v6"
	}

	key := c.getKeyWithPrefix(
		"provider-item:" + version + ":" + source + ":" + hex.EncodeToString(hash[:]),
	)
	return key
}

const searchCacheExpiration = 7 * 24 * time.Hour
const retainedSearchExpiration = 14 * 24 * time.Hour
const emptySearchExpiration = 15 * time.Minute
const providerItemExpiration = 24 * time.Hour

const releaseSearchScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('DEL', KEYS[1])
end
return 0
`
