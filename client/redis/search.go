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

func (c *Client) GetCachedSearches(
	ctx context.Context,
	source string,
	queries []string,
) ([]vibe.CachedSearch, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedSearches")
	defer span.End()

	searches := make([]vibe.CachedSearch, 0, len(queries))
	if c.Redis == nil || len(queries) == 0 {
		return searches, nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection in GetCachedSearches: %w", err)
	}
	defer connection.Close()

	keys := make(redis.Args, 0, len(queries))
	cachedQueries := make([]string, 0, len(queries))
	for _, query := range queries {
		key := c.searchCacheKey(source, query)
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
		return nil, fmt.Errorf("error getting cached searches in GetCachedSearches: %w", err)
	}

	for index, value := range values {
		if value == nil {
			continue
		}

		body, err := redis.Bytes(value, nil)
		if err != nil {
			return nil, fmt.Errorf("error reading cached search in GetCachedSearches: %w", err)
		}

		var search vibe.CachedSearch
		err = json.Unmarshal(body, &search)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling cached search in GetCachedSearches: %w", err)
		}
		search.Query = cachedQueries[index]
		searches = append(searches, search)
	}

	return searches, nil
}

func (c *Client) CacheSearches(
	ctx context.Context,
	source string,
	searches []vibe.CachedSearch,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheSearches")
	defer span.End()

	if c.Redis == nil || len(searches) == 0 {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return fmt.Errorf("error getting redis connection in CacheSearches: %w", err)
	}
	defer connection.Close()

	commandCount := 0
	for _, search := range searches {
		key := c.searchCacheKey(source, search.Query)
		if key == "" {
			continue
		}

		body, err := json.Marshal(search)
		if err != nil {
			return fmt.Errorf("error marshaling cached search in CacheSearches: %w", err)
		}

		err = connection.Send(
			"SET",
			key,
			body,
			"EX",
			int(searchCacheExpiration.Seconds()),
		)
		if err != nil {
			return fmt.Errorf("error queueing cached search in CacheSearches: %w", err)
		}
		commandCount++
	}
	if commandCount == 0 {
		return nil
	}

	err = connection.Flush()
	if err != nil {
		return fmt.Errorf("error flushing cached searches in CacheSearches: %w", err)
	}
	for range commandCount {
		_, err = redis.ReceiveContext(connection, cctx)
		if err != nil {
			return fmt.Errorf("error storing cached search in CacheSearches: %w", err)
		}
	}

	return nil
}

func (c *Client) GetCachedMusicTrack(
	ctx context.Context,
	source string,
	sourceID string,
) (*vibe.MusicTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetCachedMusicTrack")
	defer span.End()

	if c.Redis == nil || source == "" || sourceID == "" {
		return &vibe.MusicTrack{}, nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error getting redis connection in GetCachedMusicTrack: %w", err)
	}
	defer connection.Close()

	body, err := redis.Bytes(redis.DoContext(
		connection,
		cctx,
		"GET",
		c.musicTrackCacheKey(source, sourceID),
	))
	if errors.Is(err, redis.ErrNil) {
		return &vibe.MusicTrack{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting cached music track in GetCachedMusicTrack: %w", err)
	}

	var track vibe.MusicTrack
	err = json.Unmarshal(body, &track)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling cached music track in GetCachedMusicTrack: %w", err)
	}

	return &track, nil
}

func (c *Client) CacheMusicTracks(
	ctx context.Context,
	source string,
	tracks []vibe.MusicTrack,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CacheMusicTracks")
	defer span.End()

	if c.Redis == nil || source == "" || len(tracks) == 0 {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		return fmt.Errorf("error getting redis connection in CacheMusicTracks: %w", err)
	}
	defer connection.Close()

	commandCount := 0
	for _, track := range tracks {
		if track.ID == "" {
			continue
		}

		body, err := json.Marshal(track)
		if err != nil {
			return fmt.Errorf("error marshaling cached music track in CacheMusicTracks: %w", err)
		}

		err = connection.Send(
			"SET",
			c.musicTrackCacheKey(source, track.ID),
			body,
			"EX",
			int(searchCacheExpiration.Seconds()),
		)
		if err != nil {
			return fmt.Errorf("error queueing cached music track in CacheMusicTracks: %w", err)
		}
		commandCount++
	}
	if commandCount == 0 {
		return nil
	}

	err = connection.Flush()
	if err != nil {
		return fmt.Errorf("error flushing cached music tracks in CacheMusicTracks: %w", err)
	}
	for range commandCount {
		_, err = redis.ReceiveContext(connection, cctx)
		if err != nil {
			return fmt.Errorf("error storing cached music track in CacheMusicTracks: %w", err)
		}
	}

	return nil
}

func (c *Client) searchCacheKey(source string, query string) string {
	normalizedQuery := vibe.NormalizeSearch(query)
	if source == "" || normalizedQuery == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(normalizedQuery))
	key := c.getKeyWithPrefix(
		"search:" + string(source) + ":" + hex.EncodeToString(hash[:]),
	)

	return key
}

func (c *Client) musicTrackCacheKey(source string, sourceID string) string {
	hash := sha256.Sum256([]byte(sourceID))
	key := c.getKeyWithPrefix(
		"track:" + source + ":" + hex.EncodeToString(hash[:]),
	)
	return key
}

const searchCacheExpiration = 3 * 24 * time.Hour
