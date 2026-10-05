package vibe

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"time"
)

// RemoteEventRecord versions durable events independently from HTTP consumers.
type RemoteEventRecord struct {
	Version int           `json:"version"`
	Event   RemoteEventV2 `json:"event"`
}

func ParseRemoteEvent(data []byte) (*RemoteEventV2, error) {
	var record RemoteEventRecord
	err := json.Unmarshal(data, &record)
	if err != nil {
		return nil, fmt.Errorf("error decoding remote event record: %w", err)
	}

	if record.Version == 2 {
		return &record.Event, nil
	}
	if record.Version != 0 {
		return nil, fmt.Errorf("error unsupported remote event version: %d", record.Version)
	}

	var legacy RemoteEvent
	err = json.Unmarshal(data, &legacy)
	if err != nil {
		return nil, fmt.Errorf("error decoding retained legacy remote event: %w", err)
	}

	event := legacy.ToRemoteEventV2()
	return event, nil
}

type RemoteControlV2 struct {
	ID                    string    `json:"id"`
	OwnerUserID           string    `json:"-"`
	CurrentRoomID         string    `json:"currentRoomId"`
	CurrentPlaylistItemID string    `json:"currentPlaylistItemId"`
	PlaybackPositionMs    int       `json:"playbackPositionMs"`
	PlaybackIsPlaying     bool      `json:"playbackIsPlaying"`
	PlaybackObservedAt    time.Time `json:"playbackObservedAt"`
	Paired                bool      `json:"paired"`
	PairingExpiresAt      time.Time `json:"pairingExpiresAt"`
	LastSeenAt            time.Time `json:"lastSeenAt"`
}

func (r *RemoteControlV2) IsEmpty() bool {
	return r.ID == ""
}

func (r RemoteControlV2) ToRemoteControl() *RemoteControl {
	return &RemoteControl{
		ID:                 r.ID,
		OwnerUserID:        r.OwnerUserID,
		CurrentRoomID:      r.CurrentRoomID,
		CurrentSongID:      r.CurrentPlaylistItemID,
		PlaybackPositionMs: int64(r.PlaybackPositionMs),
		PlaybackIsPlaying:  r.PlaybackIsPlaying,
		PlaybackObservedAt: r.PlaybackObservedAt,
		Paired:             r.Paired,
		PairingExpiresAt:   r.PairingExpiresAt,
		LastSeenAt:         r.LastSeenAt,
	}
}

func (r RemoteControl) ToRemoteControlV2() *RemoteControlV2 {
	return &RemoteControlV2{
		ID:                    r.ID,
		OwnerUserID:           r.OwnerUserID,
		CurrentRoomID:         r.CurrentRoomID,
		CurrentPlaylistItemID: r.CurrentSongID,
		PlaybackPositionMs:    int(r.PlaybackPositionMs),
		PlaybackIsPlaying:     r.PlaybackIsPlaying,
		PlaybackObservedAt:    r.PlaybackObservedAt,
		Paired:                r.Paired,
		PairingExpiresAt:      r.PairingExpiresAt,
		LastSeenAt:            r.LastSeenAt,
	}
}

type RemotePairingV2 struct {
	RemoteControlV2
	PairingToken string `json:"pairingToken"`
	PairingCode  string `json:"pairingCode"`
}

type RemoteStatusV2 struct {
	Enabled               bool      `json:"enabled"`
	ID                    string    `json:"id"`
	CurrentRoomID         string    `json:"currentRoomId"`
	CurrentPlaylistItemID string    `json:"currentPlaylistItemId"`
	PlaybackPositionMs    int       `json:"playbackPositionMs"`
	PlaybackIsPlaying     bool      `json:"playbackIsPlaying"`
	PlaybackObservedAt    time.Time `json:"playbackObservedAt"`
	Online                bool      `json:"online"`
	Paired                bool      `json:"paired"`
}

func (r RemoteStatusV2) ToRemoteStatus() *RemoteStatus {
	return &RemoteStatus{
		Enabled:            r.Enabled,
		ID:                 r.ID,
		CurrentRoomID:      r.CurrentRoomID,
		CurrentSongID:      r.CurrentPlaylistItemID,
		PlaybackPositionMs: int64(r.PlaybackPositionMs),
		PlaybackIsPlaying:  r.PlaybackIsPlaying,
		PlaybackObservedAt: r.PlaybackObservedAt,
		Online:             r.Online,
		Paired:             r.Paired,
	}
}

func (r RemoteStatus) ToRemoteStatusV2() *RemoteStatusV2 {
	return &RemoteStatusV2{
		Enabled:               r.Enabled,
		ID:                    r.ID,
		CurrentRoomID:         r.CurrentRoomID,
		CurrentPlaylistItemID: r.CurrentSongID,
		PlaybackPositionMs:    int(r.PlaybackPositionMs),
		PlaybackIsPlaying:     r.PlaybackIsPlaying,
		PlaybackObservedAt:    r.PlaybackObservedAt,
		Online:                r.Online,
		Paired:                r.Paired,
	}
}

type RemoteSessionV2 struct {
	RemoteStatusV2
	ControllerToken string `json:"controllerToken"`
}

type RemoteUpdateRequestV2 struct {
	RoomID                string `json:"roomId"`
	CurrentPlaylistItemID string `json:"currentPlaylistItemId"`
	PlaybackPositionMs    int    `json:"playbackPositionMs"`
	PlaybackIsPlaying     bool   `json:"playbackIsPlaying"`
}

func (r RemoteUpdateRequestV2) ToRemoteUpdateRequest() *RemoteUpdateRequest {
	return &RemoteUpdateRequest{
		RoomID:             r.RoomID,
		CurrentSongID:      r.CurrentPlaylistItemID,
		PlaybackPositionMs: int64(r.PlaybackPositionMs),
		PlaybackIsPlaying:  r.PlaybackIsPlaying,
	}
}

func (r RemoteUpdateRequest) ToRemoteUpdateRequestV2() *RemoteUpdateRequestV2 {
	return &RemoteUpdateRequestV2{
		RoomID:                r.RoomID,
		CurrentPlaylistItemID: r.CurrentSongID,
		PlaybackPositionMs:    int(r.PlaybackPositionMs),
		PlaybackIsPlaying:     r.PlaybackIsPlaying,
	}
}

type RemoteEventV2 struct {
	Type                  string    `json:"type"`
	RoomID                string    `json:"roomId"`
	Origin                string    `json:"origin"`
	Online                bool      `json:"online"`
	Paired                bool      `json:"paired"`
	CurrentPlaylistItemID string    `json:"currentPlaylistItemId"`
	PlaybackPositionMs    int       `json:"playbackPositionMs"`
	PlaybackIsPlaying     bool      `json:"playbackIsPlaying"`
	PlaybackObservedAt    time.Time `json:"playbackObservedAt"`
}

func (r RemoteEventV2) ToRemoteEvent() *RemoteEvent {
	return &RemoteEvent{
		Type:               r.Type,
		RoomID:             r.RoomID,
		Origin:             r.Origin,
		Online:             r.Online,
		Paired:             r.Paired,
		CurrentSongID:      r.CurrentPlaylistItemID,
		PlaybackPositionMs: int64(r.PlaybackPositionMs),
		PlaybackIsPlaying:  r.PlaybackIsPlaying,
		PlaybackObservedAt: r.PlaybackObservedAt,
	}
}

func (r RemoteEvent) ToRemoteEventV2() *RemoteEventV2 {
	return &RemoteEventV2{
		Type:                  r.Type,
		RoomID:                r.RoomID,
		Origin:                r.Origin,
		Online:                r.Online,
		Paired:                r.Paired,
		CurrentPlaylistItemID: r.CurrentSongID,
		PlaybackPositionMs:    int(r.PlaybackPositionMs),
		PlaybackIsPlaying:     r.PlaybackIsPlaying,
		PlaybackObservedAt:    r.PlaybackObservedAt,
	}
}

type RemoteControlCreatorV2 interface {
	CreateRemoteControlV2(ctx context.Context, remoteID, ownerUserID, pairingTokenHash, pairingCodeHash, roomID string, pairingExpiresAt time.Time) (*RemoteControlV2, error)
}

type RemoteControlEnablerV2 interface {
	RemoteControlCreatorV2
	RoomV2Fetcher
}

type OwnedRemoteControlFetcherV2 interface {
	GetRemoteControlByOwnerV2(ctx context.Context, ownerUserID string) (*RemoteControlV2, error)
}

type RemoteControlFetcherV2 interface {
	GetRemoteControlV2(ctx context.Context, remoteID string) (*RemoteControlV2, error)
}

type RemoteControlPairerV2 interface {
	PairRemoteControlV2(ctx context.Context, remoteID, pairingTokenHash, pairingCodeHash, controllerTokenHash string) (*RemoteControlV2, error)
}

type RemoteControlAuthenticatorV2 interface {
	AuthenticateRemoteControlV2(ctx context.Context, remoteID, controllerTokenHash string) (*RemoteControlV2, error)
}

type OwnedRemoteControlUpdaterV2 interface {
	UpdateOwnedRemoteControlV2(ctx context.Context, remoteID, ownerUserID string, request RemoteUpdateRequestV2) (*RemoteControlV2, error)
}

type PairedRemoteControlUpdaterV2 interface {
	UpdatePairedRemoteControlV2(ctx context.Context, remoteID string, request RemoteUpdateRequestV2) (*RemoteControlV2, error)
}

type RemoteControlRoomUpdaterV2 interface {
	OwnedRemoteControlUpdaterV2
	PairedRemoteControlUpdaterV2
	RoomV2Fetcher
}

type RemoteEventNotifierV2 interface {
	NotifyRemoteUpdateV2(ctx context.Context, remoteID string, event RemoteEventV2) error
}
