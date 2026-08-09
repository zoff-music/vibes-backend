package vibe

import (
	"context"
	"time"
)

type RemoteControl struct {
	ID                 string    `json:"id"`
	OwnerUserID        string    `json:"-"`
	CurrentRoomID      string    `json:"currentRoomId"`
	CurrentSongID      string    `json:"currentSongId"`
	PlaybackPositionMs int64     `json:"playbackPositionMs"`
	PlaybackIsPlaying  bool      `json:"playbackIsPlaying"`
	PlaybackObservedAt time.Time `json:"playbackObservedAt"`
	Paired             bool      `json:"paired"`
	PairingExpiresAt   time.Time `json:"pairingExpiresAt,omitempty"`
	LastSeenAt         time.Time `json:"lastSeenAt"`
}

func (r *RemoteControl) IsEmpty() bool {
	return r.ID == ""
}

func (r *RemoteControl) IsOnline(timeout time.Duration) bool {
	return r.LastSeenAt.After(time.Now().Add(-timeout))
}

type RemotePairing struct {
	RemoteControl
	PairingToken string `json:"pairingToken"`
	PairingCode  string `json:"pairingCode"`
}

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

type RemoteSession struct {
	RemoteStatus
	ControllerToken string `json:"controllerToken"`
}

type RemotePairingRequest struct {
	PairingToken string `json:"pairingToken"`
	PairingCode  string `json:"pairingCode"`
}

type RemoteUpdateRequest struct {
	RoomID             string `json:"roomId"`
	CurrentSongID      string `json:"currentSongId"`
	PlaybackPositionMs int64  `json:"playbackPositionMs"`
	PlaybackIsPlaying  bool   `json:"playbackIsPlaying"`
}

type RemoteEvent struct {
	Type               string    `json:"type"`
	RoomID             string    `json:"roomId"`
	Origin             string    `json:"origin"`
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
	AuthenticateRemoteControl(ctx context.Context, remoteID, controllerTokenHash string, presenceTimeout time.Duration) (*RemoteControl, error)
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
