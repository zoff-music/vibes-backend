package vibe

import (
	"context"
	"time"
)

// Deprecated: Use RemoteControlV2 for new code. Retained for legacy API compatibility.
type RemoteControl struct {
	ID                 string    `json:"id"`
	OwnerUserID        string    `json:"-"`
	CurrentRoomID      string    `json:"currentRoomId"`
	CurrentSongID      string    `json:"currentSongId"`
	PlaybackPositionMs int64     `json:"playbackPositionMs"`
	PlaybackIsPlaying  bool      `json:"playbackIsPlaying"`
	PlaybackObservedAt time.Time `json:"playbackObservedAt"`
	Paired             bool      `json:"paired"`
	PairingExpiresAt   time.Time `json:"pairingExpiresAt"`
	LastSeenAt         time.Time `json:"lastSeenAt"`
}

func (r *RemoteControl) IsEmpty() bool {
	return r.ID == ""
}

// Deprecated: Use RemotePairingV2 for new code. Retained for legacy API compatibility.
type RemotePairing struct {
	RemoteControl
	PairingToken string `json:"pairingToken"`
	PairingCode  string `json:"pairingCode"`
}

// Deprecated: Use RemoteStatusV2 for new code. Retained for legacy API compatibility.
type RemoteStatus struct {
	Enabled            bool      `json:"enabled"`
	ID                 string    `json:"id"`
	CurrentRoomID      string    `json:"currentRoomId"`
	CurrentSongID      string    `json:"currentSongId"`
	PlaybackPositionMs int64     `json:"playbackPositionMs"`
	PlaybackIsPlaying  bool      `json:"playbackIsPlaying"`
	PlaybackObservedAt time.Time `json:"playbackObservedAt"`
	Online             bool      `json:"online"`
	Paired             bool      `json:"paired"`
}

// Deprecated: Use RemoteSessionV2 for new code. Retained for legacy API compatibility.
type RemoteSession struct {
	RemoteStatus
	ControllerToken string `json:"controllerToken"`
}

type RemotePairingRequest struct {
	PairingToken string `json:"pairingToken"`
	PairingCode  string `json:"pairingCode"`
}

// Deprecated: Use RemoteUpdateRequestV2 for new code. Retained for legacy API compatibility.
type RemoteUpdateRequest struct {
	RoomID             string `json:"roomId"`
	CurrentSongID      string `json:"currentSongId"`
	PlaybackPositionMs int64  `json:"playbackPositionMs"`
	PlaybackIsPlaying  bool   `json:"playbackIsPlaying"`
}

// Deprecated: Use RemoteEventV2 for new code. Retained for legacy API compatibility.
type RemoteEvent struct {
	Type               string    `json:"type"`
	RoomID             string    `json:"roomId"`
	Origin             string    `json:"origin"`
	Online             bool      `json:"online"`
	Paired             bool      `json:"paired"`
	CurrentSongID      string    `json:"currentSongId"`
	PlaybackPositionMs int64     `json:"playbackPositionMs"`
	PlaybackIsPlaying  bool      `json:"playbackIsPlaying"`
	PlaybackObservedAt time.Time `json:"playbackObservedAt"`
}

type RemoteControlCreator interface {
	CreateRemoteControl(ctx context.Context, remoteID, ownerUserID, pairingTokenHash, pairingCodeHash, roomID string, pairingExpiresAt time.Time) (*RemoteControl, error)
}

type RemoteControlEnabler interface {
	RemoteControlCreator
	RoomFetcher
}

type OwnedRemoteControlFetcher interface {
	GetRemoteControlByOwner(ctx context.Context, ownerUserID string) (*RemoteControl, error)
}

type RemoteControlFetcher interface {
	GetRemoteControl(ctx context.Context, remoteID string) (*RemoteControl, error)
}

type RemoteControlPairer interface {
	PairRemoteControl(ctx context.Context, remoteID, pairingTokenHash, pairingCodeHash, controllerTokenHash string) (*RemoteControl, error)
}

type RemoteControlAuthenticator interface {
	AuthenticateRemoteControl(ctx context.Context, remoteID, controllerTokenHash string) (*RemoteControl, error)
}

type OwnedRemoteControlUpdater interface {
	UpdateOwnedRemoteControl(ctx context.Context, remoteID, ownerUserID string, request RemoteUpdateRequest) (*RemoteControl, error)
}

type PairedRemoteControlUpdater interface {
	UpdatePairedRemoteControl(ctx context.Context, remoteID string, request RemoteUpdateRequest) (*RemoteControl, error)
}

type RemoteControlRoomUpdater interface {
	OwnedRemoteControlUpdater
	PairedRemoteControlUpdater
	RoomFetcher
}

type OwnedRemoteControlDeleter interface {
	DeleteRemoteControl(ctx context.Context, remoteID, ownerUserID string) error
}

type RemoteEventNotifier interface {
	NotifyRemoteUpdate(ctx context.Context, remoteID string, event RemoteEvent) error
}

const RemoteRoomUpdate = "remote_room_update"

const RemoteStateUpdate = "remote_state_update"

const RemoteOriginMachine = "machine"

const RemoteOriginController = "controller"
