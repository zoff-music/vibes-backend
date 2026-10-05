package vibe

import "context"

type StatsV2 struct {
	TotalListeners     int `json:"totalListeners"`
	TotalPlaylistItems int `json:"totalPlaylistItems"`
	TotalRooms         int `json:"totalRooms"`
}

func (s StatsV2) ToStats() *Stats {
	return &Stats{TotalListeners: s.TotalListeners, TotalSongs: s.TotalPlaylistItems, TotalRooms: s.TotalRooms}
}

type StatsV2Fetcher interface {
	GetStatsV2(ctx context.Context) (*StatsV2, error)
}

type CachedStatsV2 struct {
	Stats StatsV2
	Found bool
}

func (s *CachedStatsV2) IsEmpty() bool {
	return !s.Found
}

type CachedStatsV2FetcherCreator interface {
	GetCachedStatsV2(ctx context.Context) (*CachedStatsV2, error)
	CacheStatsV2(ctx context.Context, stats StatsV2) error
}
