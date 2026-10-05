package database

import (
	"context"
	"fmt"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) SkipSong(ctx context.Context, roomID, userID string) (*vibe.SkipSongResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SkipSong")
	defer span.End()

	result, err := c.SkipPlaylistItem(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy skip result: %w", err)
	}

	legacy := result.ToSkipSongResult()

	return legacy, nil
}
