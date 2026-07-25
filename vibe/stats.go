package vibe

import "context"

// Stats contains public, service-wide usage statistics.
type Stats struct {
	TotalListeners int `json:"totalListeners"`
}

// CachedStats contains a stats cache lookup result.
type CachedStats struct {
	Stats Stats
	Found bool
}

// IsEmpty reports whether the stats cache lookup missed.
func (s *CachedStats) IsEmpty() bool {
	return !s.Found
}

// StatsFetcher fetches public, service-wide usage statistics.
type StatsFetcher interface {
	GetStats(ctx context.Context) (*Stats, error)
}

// CachedStatsFetcher fetches public statistics from a cache.
type CachedStatsFetcher interface {
	GetCachedStats(ctx context.Context) (*CachedStats, error)
}

// CachedStatsCreator stores public statistics in a cache.
type CachedStatsCreator interface {
	CacheStats(ctx context.Context, stats Stats) error
}

// CachedStatsFetcherCreator fetches and stores public statistics in a cache.
type CachedStatsFetcherCreator interface {
	CachedStatsFetcher
	CachedStatsCreator
}
