package vibe

import (
	"context"
	"encoding/json"
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

	if record.Version != 2 {
		return nil, fmt.Errorf("error unsupported remote event version: %d", record.Version)
	}

	return &record.Event, nil
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
