package youtube

import (
	"context"
	"fmt"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) GetPlaylist(ctx context.Context, id string) (*vibe.MusicPlaylist, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetPlaylist")
	defer span.End()

	playlist, err := c.GetProviderPlaylist(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error fetching legacy provider playlist: %w", err)
	}

	legacy := playlist.ToMusicPlaylist()
	return legacy, nil
}

func (c *Client) GetTrack(ctx context.Context, id string) (*vibe.MusicTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetTrack")
	defer span.End()
	item, err := c.GetProviderItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy provider item: %w", err)
	}
	legacy := item.ToMusicTrack()
	return legacy, nil
}

func (c *Client) Search(ctx context.Context, query string) ([]vibe.MusicTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "Search")
	defer span.End()
	items, err := c.SearchProviderItems(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy provider search: %w", err)
	}
	tracks := make([]vibe.MusicTrack, len(items))
	for index, item := range items {
		tracks[index] = *item.ToMusicTrack()
	}
	return tracks, nil
}
