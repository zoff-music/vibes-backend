package database

import (
	"context"
	"fmt"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetSongs maps canonical rows to the unchanged v1 queue contract.
func (c *Client) GetSongs(ctx context.Context, roomID string) ([]vibe.Song, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetSongs")
	defer span.End()

	items, err := c.GetPlaylistItems(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error fetching legacy songs: %w", err)
	}

	songs := make([]vibe.Song, 0, len(items))
	for _, item := range items {
		song := item.ToSong()
		songs = append(songs, *song)
	}

	return songs, nil
}

// GetSong preserves the v1 representation without querying a legacy table.
func (c *Client) GetSong(ctx context.Context, roomID, songID string) (*vibe.Song, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetSong")
	defer span.End()

	item, err := c.GetPlaylistItem(ctx, roomID, songID)
	if err != nil {
		return nil, fmt.Errorf("error fetching legacy song: %w", err)
	}

	song := item.ToSong()

	return song, nil
}

func (r *playlistItemRow) toSong() (*vibe.Song, error) {
	item, err := r.toPlaylistItem()
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy song: %w", err)
	}

	song := item.ToSong()

	return song, nil
}

func (c *Client) AddSong(ctx context.Context, song *vibe.Song) (*vibe.AddSongResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AddSong")
	defer span.End()

	item := song.ToPlaylistItem()
	result, err := c.AddPlaylistItem(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("error adding legacy song: %w", err)
	}

	legacy := result.ToAddSongResult()

	return legacy, nil
}

func (c *Client) AddPlaylistSong(ctx context.Context, song *vibe.Song) (*vibe.AddSongResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AddPlaylistSong")
	defer span.End()

	item := song.ToPlaylistItem()
	result, err := c.AddImportedPlaylistItem(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("error importing legacy song: %w", err)
	}

	legacy := result.ToAddSongResult()

	return legacy, nil
}

func (c *Client) RemoveSong(ctx context.Context, roomID, songID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "RemoveSong")
	defer span.End()

	err := c.RemovePlaylistItem(ctx, roomID, songID)
	if err != nil {
		return fmt.Errorf("error removing legacy song: %w", err)
	}

	return nil
}

func (c *Client) VoteSong(ctx context.Context, roomID, songID, userID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "VoteSong")
	defer span.End()

	err := c.VotePlaylistItem(ctx, roomID, songID, userID)
	if err != nil {
		return fmt.Errorf("error voting for legacy song: %w", err)
	}

	return nil
}

func (r *addPlaylistItemRow) toSong() (*vibe.Song, error) {
	item, err := r.toPlaylistItem()
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy added song: %w", err)
	}

	song := item.ToSong()

	return song, nil
}
