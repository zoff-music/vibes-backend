package handler

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// CreateRemoteControl enables remote control and creates a one-use pairing.
//
//	@Summary	Enable remote control
//	@Tags		remotes
//	@Accept		json
//	@Produce	json
//	@Param		request	body		vibe.RemoteUpdateRequest	true	"Current machine room"
//	@Success	201		{object}	vibe.RemotePairing
//	@Failure	400		{object}	map[string]string
//	@Failure	401		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/remotes [post]
func CreateRemoteControl(
	db vibe.RemoteControlEnabler,
	notifier vibe.RemoteEventNotifier,
	secret string,
	pairingTTL time.Duration,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" || session.AuthType != "cookie" {
			handleError(
				w,
				fmt.Errorf("error cookie session required to enable remote control"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		var request vibe.RemoteUpdateRequest
		err := json.UnmarshalRead(r.Body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding remote control request: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		request.RoomID = strings.TrimSpace(request.RoomID)
		if request.RoomID != "" {
			room, err := db.GetRoom(ctx, request.RoomID, session.UserID)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error getting remote control room: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			if room.IsEmpty() {
				handleError(
					w,
					fmt.Errorf("error remote control room not found"),
					http.StatusNotFound,
					false,
				)
				return
			}
		}

		pairingToken, err := helper.GenerateRemoteToken()
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error generating remote pairing token: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		pairingCode, err := helper.GenerateRemoteCode()
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error generating remote pairing code: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		remoteID := uuid.New().String()
		pairingExpiresAt := time.Now().Add(pairingTTL)
		remote, err := db.CreateRemoteControl(
			ctx,
			remoteID,
			session.UserID,
			helper.HashRemoteCredential(secret, pairingToken),
			helper.HashRemoteCredential(secret, pairingCode),
			request.RoomID,
			pairingExpiresAt,
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error creating remote control: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		event := vibe.RemoteEvent{
			Type:   vibe.RemoteRoomUpdate,
			RoomID: remote.CurrentRoomID,
			Origin: vibe.RemoteOriginMachine,
			Online: true,
			Paired: false,
		}
		err = notifier.NotifyRemoteUpdate(ctx, remote.ID, event)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error notifying remote control update: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(vibe.RemotePairing{
			RemoteControl: *remote,
			PairingToken:  pairingToken,
			PairingCode:   pairingCode,
		})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling remote pairing: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	}
}

// GetOwnedRemoteControl returns the remote capability owned by this machine.
//
//	@Summary	Get owned remote control
//	@Tags		remotes
//	@Produce	json
//	@Success	200	{object}	vibe.RemoteStatus
//	@Failure	401	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/remotes [get]
func GetOwnedRemoteControl(
	fetcher vibe.OwnedRemoteControlFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" || session.AuthType != "cookie" {
			handleError(
				w,
				fmt.Errorf("error cookie session required to get remote control"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		remote, err := fetcher.GetRemoteControlByOwner(ctx, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting owned remote control: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if remote.IsEmpty() {
			body, err := json.Marshal(vibe.RemoteStatus{})
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling disabled remote control: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}

		body, err := json.Marshal(vibe.RemoteStatus{
			Enabled:            true,
			ID:                 remote.ID,
			CurrentRoomID:      remote.CurrentRoomID,
			CurrentSongID:      remote.CurrentSongID,
			PlaybackPositionMs: remote.PlaybackPositionMs,
			PlaybackIsPlaying:  remote.PlaybackIsPlaying,
			PlaybackObservedAt: remote.PlaybackObservedAt,
			Online:             true,
			Paired:             remote.Paired,
		})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling owned remote control: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// PairRemoteControl consumes a one-use pairing credential.
//
//	@Summary	Pair a remote controller
//	@Tags		remotes
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Remote ID"
//	@Param		request	body		vibe.RemotePairingRequest	true	"Pairing credential"
//	@Success	201		{object}	vibe.RemoteSession
//	@Failure	400		{object}	map[string]string
//	@Failure	401		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/remotes/{id}/sessions [post]
func PairRemoteControl(
	pairer vibe.RemoteControlPairer,
	notifier vibe.RemoteEventNotifier,
	secret string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		remoteID := mux.Vars(r)["id"]
		err := uuid.Validate(remoteID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating remote ID: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		var request vibe.RemotePairingRequest
		err = json.UnmarshalRead(r.Body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding remote pairing request: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		request.PairingToken = strings.TrimSpace(request.PairingToken)
		request.PairingCode = strings.ToUpper(strings.TrimSpace(request.PairingCode))
		if request.PairingToken == "" && request.PairingCode == "" {
			handleError(
				w,
				fmt.Errorf("error pairing token or code is required"),
				http.StatusBadRequest,
				false,
			)
			return
		}
		if len(request.PairingToken) > remotePairingTokenMaxLength ||
			len(request.PairingCode) > remotePairingCodeLength {
			handleError(
				w,
				fmt.Errorf("error remote pairing credential is invalid"),
				http.StatusBadRequest,
				false,
			)
			return
		}
		if request.PairingCode != "" && len(request.PairingCode) != remotePairingCodeLength {
			handleError(
				w,
				fmt.Errorf("error remote pairing code is invalid"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		controllerToken, err := helper.GenerateRemoteToken()
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error generating remote controller token: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		pairingTokenHash := ""
		if request.PairingToken != "" {
			pairingTokenHash = helper.HashRemoteCredential(secret, request.PairingToken)
		}
		pairingCodeHash := ""
		if request.PairingCode != "" {
			pairingCodeHash = helper.HashRemoteCredential(secret, request.PairingCode)
		}

		remote, err := pairer.PairRemoteControl(
			ctx,
			remoteID,
			pairingTokenHash,
			pairingCodeHash,
			helper.HashRemoteCredential(secret, controllerToken),
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error pairing remote control: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if remote.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error remote pairing is invalid or expired"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		err = notifier.NotifyRemoteUpdate(ctx, remote.ID, vibe.RemoteEvent{
			Type:               vibe.RemoteStateUpdate,
			RoomID:             remote.CurrentRoomID,
			Origin:             vibe.RemoteOriginController,
			Online:             true,
			Paired:             true,
			CurrentSongID:      remote.CurrentSongID,
			PlaybackPositionMs: remote.PlaybackPositionMs,
			PlaybackIsPlaying:  remote.PlaybackIsPlaying,
			PlaybackObservedAt: remote.PlaybackObservedAt,
		})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error notifying machine of remote pairing: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     remoteSessionCookieName,
			Value:    controllerToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		})

		body, err := json.Marshal(vibe.RemoteSession{
			RemoteStatus: vibe.RemoteStatus{
				Enabled:            true,
				ID:                 remote.ID,
				CurrentRoomID:      remote.CurrentRoomID,
				CurrentSongID:      remote.CurrentSongID,
				PlaybackPositionMs: remote.PlaybackPositionMs,
				PlaybackIsPlaying:  remote.PlaybackIsPlaying,
				PlaybackObservedAt: remote.PlaybackObservedAt,
				Online:             true,
				Paired:             remote.Paired,
			},
			ControllerToken: controllerToken,
		})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling paired remote control: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	}
}

// GetRemoteControl returns a paired remote's current machine state.
//
//	@Summary	Get remote control state
//	@Tags		remotes
//	@Produce	json
//	@Param		id	path		string	true	"Remote ID"
//	@Success	200	{object}	vibe.RemoteStatus
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Router		/api/v1/remotes/{id} [get]
func GetRemoteControl(
	fetcher vibe.RemoteControlFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		remoteID := mux.Vars(r)["id"]
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok ||
			session.UserID == "" ||
			(session.AuthType != "cookie" && session.AuthType != "remote") {
			handleError(
				w,
				fmt.Errorf("error remote session required"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		remote, err := fetcher.GetRemoteControl(ctx, remoteID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting remote control: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if remote.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error remote control not found"),
				http.StatusNotFound,
				false,
			)
			return
		}
		if session.AuthType == "cookie" && remote.OwnerUserID != session.UserID {
			handleError(
				w,
				fmt.Errorf("error remote control access forbidden"),
				http.StatusForbidden,
				false,
			)
			return
		}
		if session.AuthType == "remote" && session.RemoteID != remote.ID {
			handleError(
				w,
				fmt.Errorf("error remote control access forbidden"),
				http.StatusForbidden,
				false,
			)
			return
		}

		body, err := json.Marshal(vibe.RemoteStatus{
			Enabled:            true,
			ID:                 remote.ID,
			CurrentRoomID:      remote.CurrentRoomID,
			CurrentSongID:      remote.CurrentSongID,
			PlaybackPositionMs: remote.PlaybackPositionMs,
			PlaybackIsPlaying:  remote.PlaybackIsPlaying,
			PlaybackObservedAt: remote.PlaybackObservedAt,
			Online:             true,
			Paired:             remote.Paired,
		})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling remote control: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// UpdateRemoteControl updates machine presence or changes its current room.
//
//	@Summary	Update remote control state
//	@Tags		remotes
//	@Accept		json
//	@Param		id		path	string					true	"Remote ID"
//	@Param		request	body	vibe.RemoteUpdateRequest	true	"Current room"
//	@Success	204
//	@Failure	400	{object}	map[string]string
//	@Failure	401	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Router		/api/v1/remotes/{id} [patch]
func UpdateRemoteControl(
	db vibe.RemoteControlRoomUpdater,
	notifier vibe.RemoteEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		remoteID := mux.Vars(r)["id"]
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok ||
			session.UserID == "" ||
			(session.AuthType != "cookie" && session.AuthType != "remote") {
			handleError(
				w,
				fmt.Errorf("error remote control session required"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		var request vibe.RemoteUpdateRequest
		err := json.UnmarshalRead(r.Body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding remote update request: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}
		request.RoomID = strings.TrimSpace(request.RoomID)
		request.CurrentSongID = strings.TrimSpace(request.CurrentSongID)
		if request.PlaybackPositionMs < 0 || request.PlaybackPositionMs > remotePlaybackPositionMaxMs {
			handleError(
				w,
				fmt.Errorf("error remote playback position is invalid"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		if request.RoomID != "" {
			room, err := db.GetRoom(ctx, request.RoomID, session.UserID)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error getting remote update room: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			if room.IsEmpty() {
				handleError(
					w,
					fmt.Errorf("error remote update room not found"),
					http.StatusNotFound,
					false,
				)
				return
			}
		}

		origin := vibe.RemoteOriginMachine
		var remote *vibe.RemoteControl
		if session.AuthType == "remote" {
			origin = vibe.RemoteOriginController
			remote, err = db.UpdatePairedRemoteControl(ctx, remoteID, request)
		} else {
			remote, err = db.UpdateOwnedRemoteControl(ctx, remoteID, session.UserID, request)
		}
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error updating remote control: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if remote.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error remote control not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		eventType := vibe.RemoteStateUpdate
		if origin == vibe.RemoteOriginController && request.RoomID != "" {
			eventType = vibe.RemoteRoomUpdate
		}
		currentSongID := remote.CurrentSongID
		playbackPositionMs := remote.PlaybackPositionMs
		playbackIsPlaying := remote.PlaybackIsPlaying
		if origin == vibe.RemoteOriginController && eventType == vibe.RemoteStateUpdate {
			currentSongID = request.CurrentSongID
			playbackPositionMs = request.PlaybackPositionMs
			playbackIsPlaying = request.PlaybackIsPlaying
		}
		err = notifier.NotifyRemoteUpdate(ctx, remote.ID, vibe.RemoteEvent{
			Type:               eventType,
			RoomID:             remote.CurrentRoomID,
			Origin:             origin,
			Online:             true,
			Paired:             remote.Paired,
			CurrentSongID:      currentSongID,
			PlaybackPositionMs: playbackPositionMs,
			PlaybackIsPlaying:  playbackIsPlaying,
			PlaybackObservedAt: remote.PlaybackObservedAt,
		})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error notifying remote control update: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteRemoteControl revokes a machine's remote capability.
//
//	@Summary	Disable remote control
//	@Tags		remotes
//	@Param		id	path	string	true	"Remote ID"
//	@Success	204
//	@Failure	401	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/remotes/{id} [delete]
func DeleteRemoteControl(deleter vibe.OwnedRemoteControlDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		remoteID := mux.Vars(r)["id"]
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" || session.AuthType != "cookie" {
			handleError(
				w,
				fmt.Errorf("error cookie session required to disable remote control"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		err := deleter.DeleteRemoteControl(ctx, remoteID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error deleting remote control: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

const remoteSessionCookieName = "remote_session"

const remotePairingTokenMaxLength = 128

const remotePairingCodeLength = 8

const remotePlaybackPositionMaxMs = int64((24 * time.Hour) / time.Millisecond)
