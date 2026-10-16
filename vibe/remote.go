package vibe

import (
	"context"
)

type RemotePairingRequest struct {
	PairingToken string `json:"pairingToken"`
	PairingCode  string `json:"pairingCode"`
}

type OwnedRemoteControlDeleter interface {
	DeleteRemoteControl(ctx context.Context, remoteID, ownerUserID string) error
}

const RemoteRoomUpdate = "remote_room_update"

const RemoteStateUpdate = "remote_state_update"

const RemotePresentationUpdate = "remote_presentation_update"

const RemoteOriginMachine = "machine"

const RemoteOriginController = "controller"
