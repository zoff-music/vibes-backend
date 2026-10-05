package vibe

import (
	"context"
	"time"
)

type PlaybackStateV2 struct {
	RoomID              string        `json:"-"`
	CurrentPlaylistItem *PlaylistItem `json:"currentPlaylistItem"`
	IsPlaying           bool          `json:"isPlaying"`
	PositionMs          int           `json:"positionMs"`
	UpdatedAt           time.Time     `json:"updatedAt"`
	ServerTimeMs        int           `json:"serverTimeMs"`
}

func (p *PlaybackStateV2) IsEmpty() bool {
	return p.RoomID == ""
}

func (p PlaybackStateV2) ToPlaybackState() *PlaybackState {
	var current *Song
	if p.CurrentPlaylistItem != nil {
		current = p.CurrentPlaylistItem.ToSong()
	}

	return &PlaybackState{
		RoomID:       p.RoomID,
		CurrentSong:  current,
		IsPlaying:    p.IsPlaying,
		PositionMs:   p.PositionMs,
		UpdatedAt:    p.UpdatedAt,
		ServerTimeMs: p.ServerTimeMs,
	}
}

func (p PlaybackState) ToPlaybackStateV2() *PlaybackStateV2 {
	var current *PlaylistItem
	if p.CurrentSong != nil {
		current = p.CurrentSong.ToPlaylistItem()
	}

	return &PlaybackStateV2{
		RoomID:              p.RoomID,
		CurrentPlaylistItem: current,
		IsPlaying:           p.IsPlaying,
		PositionMs:          p.PositionMs,
		UpdatedAt:           p.UpdatedAt,
		ServerTimeMs:        p.ServerTimeMs,
	}
}

type PlaybackAdvanceV2 struct {
	Playback               *PlaybackStateV2
	PreviousPlaylistItemID string
}

func (p PlaybackAdvanceV2) ToPlaybackAdvance() *PlaybackAdvance {
	var playback *PlaybackState
	if p.Playback != nil {
		playback = p.Playback.ToPlaybackState()
	}

	return &PlaybackAdvance{
		Playback:       playback,
		PreviousSongID: p.PreviousPlaylistItemID,
	}
}

type PlaybackV2Fetcher interface {
	GetPlaybackStateV2(ctx context.Context, roomID string) (*PlaybackStateV2, error)
}

type PlaybackV2Controller interface {
	UpsertPlaybackStateV2(ctx context.Context, state *PlaybackStateV2) error
}

type RoomV2GetterPlaybackUpdater interface {
	PlaybackV2Fetcher
	RoomV2Fetcher
	UpdatePlaybackV2(ctx context.Context, roomID, userID, action string, positionMs int) (*PlaybackStateV2, error)
}

type PlaybackFailureRequestV2 struct {
	PlaylistItemID string `json:"playlistItemId"`
}

type PlaybackFailureV2Storage interface {
	PlaybackV2Fetcher
	PlaylistItemsFetcher
	PlaylistItemPlaybackRestrictionUpdater
	SkipRestrictedPlaylistItem(ctx context.Context, roomID, playlistItemID string) (*PlaybackAdvanceV2, error)
}

type ExpiredPlaybackPlaylistItemFetcher interface {
	PlaylistItemsFetcher
	ProcessNextExpiredPlaybackV2(ctx context.Context) (*PlaybackAdvanceV2, error)
}
