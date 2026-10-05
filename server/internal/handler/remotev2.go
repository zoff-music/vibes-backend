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

// CreateRemoteControlV2 enables remote control and creates a one-use pairing.
//
//	@Summary	Enable remote control
//	@Tags		remotes
//	@Accept		json
//	@Produce	json
//	@Param		request	body		vibe.RemoteUpdateRequestV2	true	"Current machine room"
//	@Success	201		{object}	vibe.RemotePairingV2
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Router		/api/v2/remotes [post]
func CreateRemoteControlV2(
	db vibe.RemoteControlEnablerV2,
	notifier vibe.RemoteEventNotifierV2,
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

		var request vibe.RemoteUpdateRequestV2
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
			room, err := db.GetRoomV2(ctx, request.RoomID, session.UserID)
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
		remote, err := db.CreateRemoteControlV2(
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

		event := vibe.RemoteEventV2{
			Type:   vibe.RemoteRoomUpdate,
			RoomID: remote.CurrentRoomID,
			Origin: vibe.RemoteOriginMachine,
			Online: true,
			Paired: false,
		}
		err = notifier.NotifyRemoteUpdateV2(ctx, remote.ID, event)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error notifying remote control update: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(vibe.RemotePairingV2{
			RemoteControlV2: *remote,
			PairingToken:    pairingToken,
			PairingCode:     pairingCode,
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

// GetOwnedRemoteControlV2 returns the remote capability owned by this machine.
//
//	@Summary	Get owned remote control
//	@Tags		remotes
//	@Produce	json
//	@Success	200	{object}	vibe.RemoteStatusV2
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/remotes [get]
func GetOwnedRemoteControlV2(
	fetcher vibe.OwnedRemoteControlFetcherV2,
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

		remote, err := fetcher.GetRemoteControlByOwnerV2(ctx, session.UserID)
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
			body, err := json.Marshal(vibe.RemoteStatusV2{})
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

		body, err := json.Marshal(vibe.RemoteStatusV2{
			Enabled:               true,
			ID:                    remote.ID,
			CurrentRoomID:         remote.CurrentRoomID,
			CurrentPlaylistItemID: remote.CurrentPlaylistItemID,
			PlaybackPositionMs:    remote.PlaybackPositionMs,
			PlaybackIsPlaying:     remote.PlaybackIsPlaying,
			PlaybackObservedAt:    remote.PlaybackObservedAt,
			Online:                true,
			Paired:                remote.Paired,
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

// PairRemoteControlV2 consumes a one-use pairing credential.
//
//	@Summary	Pair a remote controller
//	@Tags		remotes
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Remote ID"
//	@Param		request	body		vibe.RemotePairingRequest	true	"Pairing credential"
//	@Success	201		{object}	vibe.RemoteSessionV2
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v2/remotes/{id}/sessions [post]
func PairRemoteControlV2(
	pairer vibe.RemoteControlPairerV2,
	notifier vibe.RemoteEventNotifierV2,
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

		remote, err := pairer.PairRemoteControlV2(
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

		err = notifier.NotifyRemoteUpdateV2(ctx, remote.ID, vibe.RemoteEventV2{
			Type:                  vibe.RemoteStateUpdate,
			RoomID:                remote.CurrentRoomID,
			Origin:                vibe.RemoteOriginController,
			Online:                true,
			Paired:                true,
			CurrentPlaylistItemID: remote.CurrentPlaylistItemID,
			PlaybackPositionMs:    remote.PlaybackPositionMs,
			PlaybackIsPlaying:     remote.PlaybackIsPlaying,
			PlaybackObservedAt:    remote.PlaybackObservedAt,
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

		body, err := json.Marshal(vibe.RemoteSessionV2{
			RemoteStatusV2: vibe.RemoteStatusV2{
				Enabled:               true,
				ID:                    remote.ID,
				CurrentRoomID:         remote.CurrentRoomID,
				CurrentPlaylistItemID: remote.CurrentPlaylistItemID,
				PlaybackPositionMs:    remote.PlaybackPositionMs,
				PlaybackIsPlaying:     remote.PlaybackIsPlaying,
				PlaybackObservedAt:    remote.PlaybackObservedAt,
				Online:                true,
				Paired:                remote.Paired,
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

// GetRemoteControlV2 returns a paired remote's current machine state.
//
//	@Summary	Get remote control state
//	@Tags		remotes
//	@Produce	json
//	@Param		id	path		string	true	"Remote ID"
//	@Success	200	{object}	vibe.RemoteStatusV2
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/remotes/{id} [get]
func GetRemoteControlV2(
	fetcher vibe.RemoteControlFetcherV2,
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

		remote, err := fetcher.GetRemoteControlV2(ctx, remoteID)
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

		body, err := json.Marshal(vibe.RemoteStatusV2{
			Enabled:               true,
			ID:                    remote.ID,
			CurrentRoomID:         remote.CurrentRoomID,
			CurrentPlaylistItemID: remote.CurrentPlaylistItemID,
			PlaybackPositionMs:    remote.PlaybackPositionMs,
			PlaybackIsPlaying:     remote.PlaybackIsPlaying,
			PlaybackObservedAt:    remote.PlaybackObservedAt,
			Online:                true,
			Paired:                remote.Paired,
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

// UpdateRemoteControlV2 updates machine presence or changes its current room.
//
//	@Summary	Update remote control state
//	@Tags		remotes
//	@Accept		json
//	@Param		id		path	string					true	"Remote ID"
//	@Param		request	body	vibe.RemoteUpdateRequestV2	true	"Current room"
//	@Success	204
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/remotes/{id} [patch]
func UpdateRemoteControlV2(
	db vibe.RemoteControlRoomUpdaterV2,
	notifier vibe.RemoteEventNotifierV2,
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

		if session.AuthType == "remote" && session.RemoteID != remoteID {
			handleError(w, fmt.Errorf("error remote control access forbidden"), http.StatusForbidden, false)
			return
		}

		var request vibe.RemoteUpdateRequestV2
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
		request.CurrentPlaylistItemID = strings.TrimSpace(request.CurrentPlaylistItemID)
		if request.PlaybackPositionMs < 0 || request.PlaybackPositionMs > int(remotePlaybackPositionMaxMs) {
			handleError(
				w,
				fmt.Errorf("error remote playback position is invalid"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		if request.RoomID != "" {
			room, err := db.GetRoomV2(ctx, request.RoomID, session.UserID)
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
		var remote *vibe.RemoteControlV2
		if session.AuthType == "remote" {
			origin = vibe.RemoteOriginController
			remote, err = db.UpdatePairedRemoteControlV2(ctx, remoteID, request)
		} else {
			remote, err = db.UpdateOwnedRemoteControlV2(ctx, remoteID, session.UserID, request)
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
		currentPlaylistItemID := remote.CurrentPlaylistItemID
		playbackPositionMs := remote.PlaybackPositionMs
		playbackIsPlaying := remote.PlaybackIsPlaying
		if origin == vibe.RemoteOriginController && eventType == vibe.RemoteStateUpdate {
			currentPlaylistItemID = request.CurrentPlaylistItemID
			playbackPositionMs = request.PlaybackPositionMs
			playbackIsPlaying = request.PlaybackIsPlaying
		}
		err = notifier.NotifyRemoteUpdateV2(ctx, remote.ID, vibe.RemoteEventV2{
			Type:                  eventType,
			RoomID:                remote.CurrentRoomID,
			Origin:                origin,
			Online:                true,
			Paired:                remote.Paired,
			CurrentPlaylistItemID: currentPlaylistItemID,
			PlaybackPositionMs:    playbackPositionMs,
			PlaybackIsPlaying:     playbackIsPlaying,
			PlaybackObservedAt:    remote.PlaybackObservedAt,
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

// RemoteEventsV2 streams machine room changes between a machine and controller.
//
//	@Summary	Subscribe to remote control events
//
// @Description Returns an SSE stream. Each frame has an `event` name and a JSON `data` payload containing RemoteEventV2 fields: type, roomId, origin, online, paired, currentPlaylistItemId, playbackPositionMs, playbackIsPlaying, and playbackObservedAt.
// @Description The first event is `remote_state_update`. Subsequent events report remote room and playback changes. Frames end with a blank line; this is not a single JSON response.
//
//	@Tags		remotes
//	@Produce	text/event-stream
//	@Param		id	path		string	true	"Remote ID"
//	@Success	200	{string}	string "SSE frames with event-specific JSON data; see the stream description"
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/remotes/{id}/events [get]
func RemoteEventsV2(
	subscriber vibe.Subscriber,
	fetcher vibe.RemoteControlFetcherV2,
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
				fmt.Errorf("error remote event session required"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		remote, err := fetcher.GetRemoteControlV2(ctx, remoteID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting remote control for events: %w", err),
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
				fmt.Errorf("error remote event access forbidden"),
				http.StatusForbidden,
				false,
			)
			return
		}
		if session.AuthType == "remote" && session.RemoteID != remote.ID {
			handleError(
				w,
				fmt.Errorf("error remote event access forbidden"),
				http.StatusForbidden,
				false,
			)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		container, err := subscriber.Subscribe(ctx, fmt.Sprintf("remote:%s", remoteID))
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error subscribing to remote events: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		defer container.Subscription.Destroy()

		flusher, ok := w.(http.Flusher)
		if !ok {
			handleError(
				w,
				fmt.Errorf("error remote event streaming not supported"),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		initialEvent := vibe.RemoteEventV2{
			Type:                  vibe.RemoteStateUpdate,
			RoomID:                remote.CurrentRoomID,
			Origin:                vibe.RemoteOriginMachine,
			Online:                true,
			Paired:                remote.Paired,
			CurrentPlaylistItemID: remote.CurrentPlaylistItemID,
			PlaybackPositionMs:    remote.PlaybackPositionMs,
			PlaybackIsPlaying:     remote.PlaybackIsPlaying,
			PlaybackObservedAt:    remote.PlaybackObservedAt,
		}
		data, err := json.Marshal(initialEvent)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling initial remote event: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", initialEvent.Type, data)
		flusher.Flush()

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		messages := container.Subscription.Listen()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Fprint(w, ": heartbeat\n\n")
				flusher.Flush()
			case data, ok := <-messages:
				if !ok {
					return
				}

				event, err := vibe.ParseRemoteEvent(data)
				if err != nil {
					continue
				}
				payload, err := json.Marshal(event)
				if err != nil {
					return
				}

				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, payload)
				flusher.Flush()
			}
		}
	}
}
