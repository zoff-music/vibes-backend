package vibe

import (
	"context"
	"time"
)

type PlaylistItemMetadataRefresh struct {
	PlaylistItemID string
	RoomID         string
	SourceID       string
}

type PlaylistItemMetadataExpiry struct {
	RoomID string
}

type PlaylistItemMetadataExpiryFetcher interface {
	ExpirePlaylistItemMetadata(ctx context.Context) (*PlaylistItemMetadataExpiry, error)
	PlaylistItemsFetcher
	PlaybackV2Fetcher
}

type PlaylistItemMetadataRefreshStorage interface {
	ClaimPlaylistItemMetadataRefresh(
		ctx context.Context,
		provider string,
		retryAfter time.Duration,
	) (*PlaylistItemMetadataRefresh, error)
	RefreshPlaylistItemMetadata(
		ctx context.Context,
		refresh PlaylistItemMetadataRefresh,
		item ProviderItem,
		refreshInterval time.Duration,
	) error
	DeferPlaylistItemMetadataRefresh(
		ctx context.Context,
		playlistItemID string,
		retryAfter time.Duration,
	) error
	PlaylistItemRemover
	PlaylistItemsFetcher
	PlaybackV2Fetcher
}
