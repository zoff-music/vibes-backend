package vibe

import (
	"context"
	"time"
)

type RemoteControl struct {
	ID               string    `json:"id"`
	OwnerUserID      string    `json:"-"`
	CurrentRoomID    string    `json:"currentRoomId"`
	PairingExpiresAt time.Time `json:"pairingExpiresAt,omitempty"`
	LastSeenAt       time.Time `json:"lastSeenAt"`
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
	Enabled       bool   `json:"enabled"`
	ID            string `json:"id"`
	CurrentRoomID string `json:"currentRoomId"`
	Online        bool   `json:"online"`
}

type RemotePairingRequest struct {
	PairingToken string `json:"pairingToken"`
	PairingCode  string `json:"pairingCode"`
}

type RemoteUpdateRequest struct {
	RoomID string `json:"roomId"`
}

type RemoteEvent struct {
	Type   string `json:"type"`
	RoomID string `json:"roomId"`
	Origin string `json:"origin"`
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
	UpdateOwnedRemoteControl(ctx context.Context, remoteID, ownerUserID, roomID string) (*RemoteControl, error)
}

type PairedRemoteControlUpdater interface {
	UpdatePairedRemoteControl(ctx context.Context, remoteID, roomID string) (*RemoteControl, error)
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

const RemoteOriginMachine = "machine"

const RemoteOriginController = "controller"
