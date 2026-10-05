package database

import (
	"context"
	"fmt"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) GetPlaybackState(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetPlaybackState")
	defer span.End()

	result, err := c.GetPlaybackStateV2(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy playback in GetPlaybackState: %w", err)
	}

	legacy := result.ToPlaybackState()
	return legacy, nil
}

func (c *Client) getPlaybackState(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getPlaybackState")
	defer span.End()

	result, err := c.getPlaybackStateV2(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy playback in getPlaybackState: %w", err)
	}

	legacy := result.ToPlaybackState()
	return legacy, nil
}

func (c *Client) StartPlaybackIfIdle(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "StartPlaybackIfIdle")
	defer span.End()

	result, err := c.StartPlaybackIfIdleV2(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy playback in StartPlaybackIfIdle: %w", err)
	}

	legacy := result.ToPlaybackState()
	return legacy, nil
}

func (c *Client) startPlaybackIfIdle(ctx context.Context, roomID string) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "startPlaybackIfIdle")
	defer span.End()

	result, err := c.startPlaybackIfIdleV2(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy playback in startPlaybackIfIdle: %w", err)
	}

	legacy := result.ToPlaybackState()
	return legacy, nil
}

func (c *Client) UpdatePlayback(ctx context.Context, roomID, userID, action string, positionMs int) (*vibe.PlaybackState, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdatePlayback")
	defer span.End()

	result, err := c.UpdatePlaybackV2(ctx, roomID, userID, action, positionMs)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy playback in UpdatePlayback: %w", err)
	}

	legacy := result.ToPlaybackState()
	return legacy, nil
}

func (c *Client) ProcessNextExpiredPlayback(ctx context.Context) (*vibe.PlaybackAdvance, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ProcessNextExpiredPlayback")
	defer span.End()

	result, err := c.ProcessNextExpiredPlaybackV2(ctx)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy playback in ProcessNextExpiredPlayback: %w", err)
	}

	legacy := result.ToPlaybackAdvance()
	return legacy, nil
}

func (c *Client) SkipRestrictedSong(ctx context.Context, roomID, songID string) (*vibe.PlaybackAdvance, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SkipRestrictedSong")
	defer span.End()

	result, err := c.SkipRestrictedPlaylistItem(ctx, roomID, songID)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy playback in SkipRestrictedSong: %w", err)
	}

	legacy := result.ToPlaybackAdvance()
	return legacy, nil
}

func (c *Client) skipTrack(ctx context.Context, roomID, restrictedSongID string) (*vibe.PlaybackAdvance, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "skipTrack")
	defer span.End()

	result, err := c.skipPlaylistItem(ctx, roomID, restrictedSongID)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy playback in skipTrack: %w", err)
	}

	legacy := result.ToPlaybackAdvance()
	return legacy, nil
}

func (c *Client) UpsertPlaybackState(ctx context.Context, state *vibe.PlaybackState) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpsertPlaybackState")
	defer span.End()

	canonical := state.ToPlaybackStateV2()
	err := c.UpsertPlaybackStateV2(ctx, canonical)
	if err != nil {
		return fmt.Errorf("error mapping legacy playback write: %w", err)
	}
	return nil
}
