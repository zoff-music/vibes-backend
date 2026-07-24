package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/gomodule/redigo/redis"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) GetCachedYouTubeSearches(
	ctx context.Context,
	queries []string,
) ([]vibe.CachedYouTubeSearch, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedYouTubeSearches")
	defer span.End()

	searches := make([]vibe.CachedYouTubeSearch, 0, len(queries))
	if c.Redis == nil || len(queries) == 0 {
		return searches, nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection in GetCachedYouTubeSearches: %w", err)
	}
	defer connection.Close()

	keys := make(redis.Args, 0, len(queries))
	cachedQueries := make([]string, 0, len(queries))
	for _, query := range queries {
		key := c.youtubeSearchCacheKey(query)
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
		return nil, fmt.Errorf("error getting cached youtube searches in GetCachedYouTubeSearches: %w", err)
	}

	for index, value := range values {
		if value == nil {
			continue
		}

		body, err := redis.Bytes(value, nil)
		if err != nil {
			return nil, fmt.Errorf("error reading cached youtube search in GetCachedYouTubeSearches: %w", err)
		}

		var search vibe.CachedYouTubeSearch
		err = json.Unmarshal(body, &search)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling cached youtube search in GetCachedYouTubeSearches: %w", err)
		}
		search.Query = cachedQueries[index]
		searches = append(searches, search)
	}

	return searches, nil
}

func (c *Client) CacheYouTubeSearches(
	ctx context.Context,
	searches []vibe.CachedYouTubeSearch,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheYouTubeSearches")
	defer span.End()

	if c.Redis == nil || len(searches) == 0 {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return fmt.Errorf("error getting redis connection in CacheYouTubeSearches: %w", err)
	}
	defer connection.Close()

	commandCount := 0
	for _, search := range searches {
		key := c.youtubeSearchCacheKey(search.Query)
		if key == "" {
			continue
		}

		body, err := json.Marshal(search)
		if err != nil {
			return fmt.Errorf("error marshaling cached youtube search in CacheYouTubeSearches: %w", err)
		}

		err = connection.Send(
			"SET",
			key,
			body,
			"EX",
			int(youtubeSearchCacheExpiration.Seconds()),
		)
		if err != nil {
			return fmt.Errorf("error queueing cached youtube search in CacheYouTubeSearches: %w", err)
		}
		commandCount++
	}
	if commandCount == 0 {
		return nil
	}

	err = connection.Flush()
	if err != nil {
		return fmt.Errorf("error flushing cached youtube searches in CacheYouTubeSearches: %w", err)
	}
	for range commandCount {
		_, err = redis.ReceiveContext(connection, cctx)
		if err != nil {
			return fmt.Errorf("error storing cached youtube search in CacheYouTubeSearches: %w", err)
		}
	}

	return nil
}

func (c *Client) youtubeSearchCacheKey(query string) string {
	normalizedQuery := normalizeYouTubeSearch(query)
	if normalizedQuery == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(normalizedQuery))

	return c.getKeyWithPrefix(
		"youtube:search:" + hex.EncodeToString(hash[:]),
	)
}

func normalizeYouTubeSearch(query string) string {
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
		if !isYouTubeSearchNoise(word) {
			tokens = append(tokens, word)
		}
		token.Reset()
	}
	if token.Len() > 0 {
		word := token.String()
		allTokens = append(allTokens, word)
		if !isYouTubeSearchNoise(word) {
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

func isYouTubeSearchNoise(value string) bool {
	switch value {
	case "4k", "a", "an", "and", "audio", "feat", "featuring", "ft", "hd",
		"lyric", "lyrics", "music", "official", "the", "video", "visualizer":
		return true
	default:
		return false
	}
}

const youtubeSearchCacheExpiration = 7 * 24 * time.Hour
