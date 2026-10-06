package handler

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
	"golang.org/x/crypto/bcrypt"
)

// GetPlaylistItems lists room queue entries using the v2 contract.
//
//	@Summary	List room playlist items
//	@Tags		playlist-items
//	@Produce	json
//	@Param		id	path		string	true	"Room ID"
//	@Success	200	{array}		vibe.PlaylistItem
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/playlist-items [get]
func GetPlaylistItems(db vibe.PlaylistItemsFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		items, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(w, fmt.Errorf("error fetching playlist items: %w", err), http.StatusInternalServerError, true)
			return
		}

		body, err := json.Marshal(items)
		if err != nil {
			handleError(w, fmt.Errorf("error marshaling playlist items: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// CreateSessionV2 handles POST /api/v2/rooms/:id/sessions
//
//	@Summary	Create a room session
//	@Tags		rooms
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Room ID"
//	@Param		request	body		vibe.CreateSessionRequest	true	"Session payload"
//	@Success	200		{object}	vibe.SessionResponseV2
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	403		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/sessions [post]
func CreateSessionV2(
	db vibe.RoomAdminSessionV2Creator,
	notifier vibe.RoomEventV3Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.CreateSessionRequest
		err := json.UnmarshalRead(r.Body, &req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error unauthorized: missing session"),
				http.StatusUnauthorized,
				true,
			)
			return
		}

		if req.Password == "" {
			handleError(
				w,
				fmt.Errorf("error password required"),
				http.StatusForbidden,
				false,
			)
			return
		}

		authResult, err := db.AuthenticateAdmin(ctx, roomID, session.UserID, req.Password)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error authentication failed: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if !authResult.IsAdmin {
			handleError(
				w,
				fmt.Errorf("error incorrect password"),
				http.StatusForbidden,
				false,
			)
			return
		}

		isFirstTimeSetup := authResult.IsFirstTimeSetup

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if isFirstTimeSetup {
			neutralRoom, err := db.GetRoomV2(ctx, roomID, "")
			if err != nil {
				log.Printf("CreateSessionV2: failed to fetch neutral room for notification: %v", err)
			}
			if err == nil {
				body, err := json.Marshal(neutralRoom)
				if err != nil {
					log.Printf("CreateSessionV2: failed to marshal room for notification: %v", err)
				}
				if err == nil {
					err = notifier.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
						Type:    vibe.SettingsUpdate,
						Payload: body,
						UserID:  session.UserID, // Include the user who set the password
						Origin:  session.EventOrigin,
					})
					if err != nil {
						log.Printf("CreateSessionV2: failed to notify room password setup: %v", err)
					}
				}
			}
		}

		resp := vibe.SessionResponseV2{
			UserID:    session.UserID,
			SessionID: session.UserID,
			IsAdmin:   true,
			Room:      room,
		}

		body, err := json.Marshal(resp)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling room session response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)

		if !isFirstTimeSetup {
			return
		}

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			log.Printf("error fetching room password chat author: %v", err)
			return
		}

		message := vibe.RoomMessage{
			ID:        uuid.NewString(),
			UserID:    session.UserID,
			Name:      profile.Name,
			IsAdmin:   true,
			Kind:      vibe.MessageKindAdded,
			IsHost:    room.Mode == vibe.RoomModeHost && room.HostID == session.UserID,
			Activity:  true,
			Text:      "a password to the room",
			CreatedAt: time.Now().UnixMilli(),
		}

		payload, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling room password chat activity: %v", err)
			return
		}

		err = notifier.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
			Type:    vibe.MessageEvent,
			Payload: payload,
		})
		if err != nil {
			log.Printf("error publishing room password chat activity: %v", err)
		}
	}
}

// DeleteRoomAdminSessionV2 handles DELETE /api/v2/rooms/:id/sessions.
//
//	@Summary	Log out of a room admin session
//	@Tags		rooms
//	@Produce	json
//	@Param		id	path		string	true	"Room ID"
//	@Success	200	{object}	vibe.SessionResponseV2
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/sessions [delete]
func DeleteRoomAdminSessionV2(db vibe.RoomAdminSessionV2Deleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		session, ok := helper.GetSessionFromContext(ctx)
		isCookieSession := session.AuthType == "" || session.AuthType == "cookie"
		if !ok || session.UserID == "" || !isCookieSession {
			handleError(
				w,
				fmt.Errorf("error unauthorized: missing session"),
				http.StatusUnauthorized,
				true,
			)
			return
		}

		err := db.ClearRoomAdmin(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error clearing room admin in DeleteRoomAdminSessionV2: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room in DeleteRoomAdminSessionV2: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		resp := vibe.SessionResponseV2{
			UserID:    session.UserID,
			SessionID: session.UserID,
			IsAdmin:   false,
			Room:      room,
		}

		body, err := json.Marshal(resp)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling response in DeleteRoomAdminSessionV2: %w", err),
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

// CreateRoomV2 handles POST /api/v2/rooms
//
//	@Summary		Create a room
//	@Description	Creates an immutable MUSIC or WATCH room. MUSIC is the default; WATCH supports YouTube only.
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			request	body		vibe.CreateRoomRequestV2	true	"Room creation payload"
//	@Success		201		{object}	vibe.RoomV2
//	@Failure		400		{object}	vibe.ErrorResponse
//	@Failure		409		{object}	vibe.ErrorResponse
//	@Failure		500		{object}	vibe.ErrorResponse
//	@Router			/api/v2/rooms [post]
func CreateRoomV2(
	db vibe.RoomV2Creator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req vibe.CreateRoomRequestV2
		err := json.UnmarshalRead(http.MaxBytesReader(w, r.Body, 8192), &req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		if req.RoomType == "" {
			req.RoomType = vibe.RoomTypeMusic
		}

		if !req.RoomType.IsValid() {
			handleError(w, fmt.Errorf("error creating room: roomType must be MUSIC or WATCH"), http.StatusBadRequest, false)
			return
		}

		if !req.Validate() {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error validating room name"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "room_name_invalid",
						Message:   fmt.Sprintf("Use between 1 and %d characters for the room name.", vibe.RoomNameMaxLength),
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		req.Name = strings.TrimSpace(req.Name)

		session, _ := helper.GetSessionFromContext(ctx)

		slug := helper.Slugify(req.Name)
		if slug == "" {
			handleError(
				w,
				fmt.Errorf("error invalid room name"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		roomExists, err := db.RoomExists(ctx, slug)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error checking room existence: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if roomExists {
			handleError(
				w,
				fmt.Errorf("error room name already exists"),
				http.StatusConflict,
				false,
			)
			return
		}

		var passwordHash string
		if req.Password != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error hashing password: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			passwordHash = string(hash)
		}

		mode := req.Mode
		if mode == "" {
			mode = vibe.RoomModeServer
		}

		settings, err := vibe.DefaultRoomSettingsV2()
		if err != nil {
			handleError(w, fmt.Errorf("error getting default room settings: %w", err), http.StatusInternalServerError, true)
			return
		}

		if req.Settings != nil {
			settings = req.Settings
		}

		if req.RoomType == vibe.RoomTypeWatch && req.Settings == nil {
			settings.EnabledSources = []string{vibe.SourceTypeYouTube}
		}

		for _, source := range settings.EnabledSources {
			if !req.RoomType.AllowsSource(source) {
				handleError(w, fmt.Errorf("error creating room: provider %q is not allowed for %s", source, req.RoomType), http.StatusBadRequest, false)
				return
			}
		}

		if req.Settings != nil && req.Settings.OnlyAdminAddPlaylistItems && req.Password == "" {
			handleError(
				w,
				fmt.Errorf("error admin password is required when enabling 'only admin add playlist items'"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		if req.Settings != nil && req.Settings.Public && req.Password == "" {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: internalerror.ErrMissingAdminPassword{
						Err: fmt.Errorf("error room must have a password to be public"),
					},
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "public_room_password_required",
						Message:   "Add an admin password before making this room public.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		room := &vibe.RoomV2{
			ID:                slug,
			Name:              req.Name,
			RoomType:          req.RoomType,
			Mode:              mode,
			HostID:            session.UserID,
			AdminPasswordHash: passwordHash,
			HasPassword:       passwordHash != "",
			Settings:          *settings,
			CreatedAt:         time.Now(),
			ActiveSources:     []string{},
		}

		created, err := db.CreateRoomV2(ctx, room, req.ReservationToken)
		if err != nil {
			var unavailableError internalerror.ErrRoomNameUnavailable
			if errors.As(err, &unavailableError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: unavailableError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_name_unavailable",
							Message:   "This room name is unavailable or its reservation expired.",
							Propagate: true,
						},
						StatusCode: http.StatusConflict,
					},
					http.StatusConflict,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf("error creating room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(created)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
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

// GetRoomV2 handles GET /api/v2/rooms/{id}
//
//	@Summary	Get a room
//	@Tags		rooms
//	@Produce	json
//	@Param		id	path		string	true	"Room ID"
//	@Success	200	{object}	vibe.RoomV2
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id} [get]
func GetRoomV2(
	db vibe.RoomV2Fetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		session, _ := helper.GetSessionFromContext(ctx)

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		body, err := json.Marshal(room)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
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

// UpdateRoomSettingsV2 handles PATCH /rooms/{id}/settings
//
//	@Summary	Update room settings
//	@Tags		rooms
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Room ID"
//	@Param		request	body		vibe.UpdateRoomRequestV2	true	"Room update payload"
//	@Success	200		{object}	vibe.RoomV2
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	404		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/settings [patch]
func UpdateRoomSettingsV2(
	db vibe.RoomV2SettingsUpdater,
	notifier vibe.RoomBatchEventV3Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.UpdateRoomRequestV2
		err := json.UnmarshalRead(r.Body, &req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error invalid request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(w, fmt.Errorf("error updating room settings: missing session"), http.StatusUnauthorized, false)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error failed to fetch room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		if room.HasPassword && !room.IsAdmin {
			handleError(w, fmt.Errorf("error updating room settings: admin required"), http.StatusForbidden, false)
			return
		}

		previousSettings := room.Settings
		previousMode := room.Mode

		if req.Settings != nil {
			for _, source := range req.Settings.EnabledSources {
				if !room.RoomType.AllowsSource(source) {
					handleError(w, fmt.Errorf("error updating room: provider %q is not allowed for %s", source, room.RoomType), http.StatusBadRequest, false)
					return
				}
			}
		}

		if req.Settings != nil && !req.Settings.IsEmpty() {
			room.Settings = *req.Settings
		}
		if req.Mode != "" {
			room.Mode = req.Mode
		}

		if room.Settings.OnlyAdminAddPlaylistItems && !room.HasPassword {
			handleError(
				w,
				internalerror.ErrMissingAdminPassword{Err: fmt.Errorf("error room must have a password to enable 'only admin add playlist items'")},
				http.StatusBadRequest,
				false,
			)
			return
		}

		if room.Settings.Public && !room.HasPassword {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: internalerror.ErrMissingAdminPassword{
						Err: fmt.Errorf("error room must have a password to be public"),
					},
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "public_room_password_required",
						Message:   "Add an admin password before making this room public.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		updated, err := db.UpdateRoomV2(ctx, room)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error failed to update room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(updated)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = notifier.NotifyRoomUpdatesV3(context.WithoutCancel(ctx), roomID, []vibe.RoomEventV3{{
			Type:    vibe.SettingsUpdate,
			Payload: body,
			Origin:  session.EventOrigin,
		}})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error failed to notify room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)

		changes := []string{}
		if req.Settings != nil && previousSettings.RemoveOnPlay != updated.Settings.RemoveOnPlay &&
			room.Settings.RemoveOnPlay == updated.Settings.RemoveOnPlay {
			value := "OFF"
			if updated.Settings.RemoveOnPlay {
				value = "ON"
			}

			changes = append(changes, "set REMOVE AFTER PLAY to "+value)
		}

		if req.Settings != nil && previousSettings.OnlyAdminAddPlaylistItems != updated.Settings.OnlyAdminAddPlaylistItems &&
			room.Settings.OnlyAdminAddPlaylistItems == updated.Settings.OnlyAdminAddPlaylistItems {
			value := "OFF"
			if updated.Settings.OnlyAdminAddPlaylistItems {
				value = "ON"
			}

			changes = append(changes, "set ADMINS ONLY ADD to "+value)
		}

		if req.Settings != nil && previousSettings.SkipAllowed != updated.Settings.SkipAllowed &&
			room.Settings.SkipAllowed == updated.Settings.SkipAllowed {
			value := "OFF"
			if !updated.Settings.SkipAllowed {
				value = "ON"
			}

			changes = append(changes, "set ADMINS ONLY SKIP to "+value)
		}

		if req.Settings != nil && previousSettings.DemocraticSkip != updated.Settings.DemocraticSkip &&
			room.Settings.DemocraticSkip == updated.Settings.DemocraticSkip {
			value := "OFF"
			if updated.Settings.DemocraticSkip {
				value = "ON"
			}

			changes = append(changes, "set VOTE TO SKIP to "+value)
		}

		if req.Settings != nil && previousSettings.AllowDuplicates != updated.Settings.AllowDuplicates &&
			room.Settings.AllowDuplicates == updated.Settings.AllowDuplicates {
			value := "OFF"
			if updated.Settings.AllowDuplicates {
				value = "ON"
			}

			changes = append(changes, "set ALLOW DUPLICATES to "+value)
		}

		if req.Settings != nil && previousSettings.Public != updated.Settings.Public &&
			room.Settings.Public == updated.Settings.Public {
			value := "OFF"
			if updated.Settings.Public {
				value = "ON"
			}

			changes = append(changes, "set PUBLIC to "+value)
		}

		if req.Settings != nil && previousSettings.PlaylistImport != updated.Settings.PlaylistImport &&
			room.Settings.PlaylistImport == updated.Settings.PlaylistImport {
			value := "OFF"
			if updated.Settings.PlaylistImport {
				value = "ON"
			}

			changes = append(changes, "set PLAYLIST IMPORT to "+value)
		}

		if req.Settings != nil && previousSettings.SkipVoteThreshold != updated.Settings.SkipVoteThreshold &&
			room.Settings.SkipVoteThreshold == updated.Settings.SkipVoteThreshold {
			changes = append(changes, fmt.Sprintf("set SKIP VOTE THRESHOLD to %g%%", updated.Settings.SkipVoteThreshold*100))
		}

		if req.Settings != nil && previousSettings.MaxContinuousAdds != updated.Settings.MaxContinuousAdds &&
			room.Settings.MaxContinuousAdds == updated.Settings.MaxContinuousAdds {
			changes = append(changes, fmt.Sprintf("set MAX CONTINUOUS ADDS to %d", updated.Settings.MaxContinuousAdds))
		}

		previousSources := slices.Clone(previousSettings.EnabledSources)
		updatedSources := slices.Clone(updated.Settings.EnabledSources)
		slices.Sort(previousSources)
		slices.Sort(updatedSources)
		if req.Settings != nil && !slices.Equal(previousSources, updatedSources) {
			value := strings.Join(updatedSources, ", ")
			if value == "" {
				value = "none"
			}

			changes = append(changes, "set MUSIC PROVIDERS to "+value)
		}

		if req.Mode != "" && previousMode != updated.Mode && req.Mode == updated.Mode {
			changes = append(changes, "set ROOM MODE to "+strings.ToUpper(updated.Mode))
		}

		if len(changes) == 0 {
			return
		}

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			log.Printf("error fetching room settings chat author: %v", err)
			return
		}

		events := []vibe.RoomEventV3{}
		for _, change := range changes {
			message := vibe.RoomMessage{
				ID:        uuid.NewString(),
				UserID:    session.UserID,
				Name:      profile.Name,
				IsAdmin:   updated.IsAdmin,
				IsHost:    updated.Mode == vibe.RoomModeHost && updated.HostID == session.UserID,
				Kind:      vibe.MessageKindSettings,
				Text:      change,
				CreatedAt: time.Now().UnixMilli(),
			}

			payload, err := json.Marshal(message)
			if err != nil {
				log.Printf("error marshaling room settings chat activity: %v", err)
				return
			}

			events = append(events, vibe.RoomEventV3{Type: vibe.SettingsActivityEvent, Payload: payload})
		}

		err = notifier.NotifyRoomUpdatesV3(context.WithoutCancel(ctx), roomID, events)
		if err != nil {
			log.Printf("error publishing room settings chat activity: %v", err)
		}
	}
}

// AddPlaylistItem handles the addition of a new playlistItem to the room
//
//	@Summary	Add a playlist item
//	@Tags		playlist-items
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Room ID"
//	@Param		request	body		vibe.AddPlaylistItemRequest	true	"Playlist item payload"
//	@Success	200		{object}	vibe.AddPlaylistItemResult
//	@Success	201		{object}	vibe.AddPlaylistItemResult
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	403		{object}	vibe.ErrorResponse
//	@Failure	404		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/playlist-items [post]
func AddPlaylistItem(
	db vibe.PlaylistItemQueueAdder,
	events vibe.CachedProviderItemRoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.AddPlaylistItemRequest
		err := json.UnmarshalRead(r.Body, &req)
		if err != nil {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error decoding request body: %w", err),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_request_invalid",
						Message:   "The playlist item details could not be read. Search for the playlist item again and retry.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				true,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error unauthorized"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_session_required",
						Message:   "Your room session is missing. Rejoin the room and try adding the playlist item again.",
						Propagate: true,
					},
					StatusCode: http.StatusUnauthorized,
				},
				http.StatusUnauthorized,
				false,
			)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if room.IsEmpty() {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error room not found"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_room_not_found",
						Message:   "This room no longer exists. Join another room to add playlist items.",
						Propagate: true,
					},
					StatusCode: http.StatusNotFound,
				},
				http.StatusNotFound,
				false,
			)
			return
		}

		if room.Settings.OnlyAdminAddPlaylistItems && !room.IsAdmin {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error only admins can add playlist items in this room"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_room_admin_required",
						Message:   "Only room admins can add playlist items here. Log in as a room admin in room settings, or ask an admin to allow everyone to add playlist items.",
						Propagate: true,
					},
					StatusCode: http.StatusForbidden,
				},
				http.StatusForbidden,
				false,
			)
			return
		}

		sourceEnabled := false
		for _, source := range room.Settings.EnabledSources {
			if req.SourceType == source && room.RoomType.AllowsSource(source) {
				sourceEnabled = true
				break
			}
		}

		if !sourceEnabled {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error source type %s is not enabled for this room", req.SourceType),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_provider_disabled",
						Message:   "This music provider is not enabled for this room. Choose another provider or ask a room admin to enable it.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		if vibe.IsLiveVideo(req.SourceType, req.Duration) {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error live videos cannot be added to rooms"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "youtube_live_video_not_supported",
						Message:   "Live videos cannot be added to rooms.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		providerURL, err := req.CanonicalProviderURL()
		if err != nil {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error getting canonical provider URL: %w", err),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_provider_url_invalid",
						Message:   "The playlist item link is not valid for this provider. Search for the playlist item again or paste a valid track link.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		publisher := req.Publisher
		playbackRestriction := ""
		cachedItem, err := events.GetCachedProviderItem(ctx, req.SourceType, req.SourceID)
		if err != nil {
			log.Printf("error getting cached provider track metadata: %v", err)
		}

		if req.SourceType == vibe.SourceTypeYouTube && (err != nil || cachedItem.IsEmpty()) {
			handleError(w, client.ErrorCodeWrapper{
				Err: fmt.Errorf("error adding youtube track: verified metadata is unavailable"),
				ResponseBody: client.ErrorCodeResponseBody{
					Namespace: "vibes-backend",
					Error:     "youtube_track_verification_required",
					Message:   "This video's availability needs to be checked again. Search for it or paste its YouTube link again before adding it.",
					Propagate: true,
				},
				StatusCode: http.StatusBadRequest,
			}, http.StatusBadRequest, false)
			return
		}

		if err == nil && !cachedItem.IsEmpty() {
			if !cachedItem.AllowedInRoom(room.RoomType) {
				handleError(w, client.ErrorCodeWrapper{
					Err: fmt.Errorf("error adding item: provider metadata is not allowed for %s", room.RoomType),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "playlist_item_not_allowed",
						Message:   "This item cannot be played in this room. Music rooms accept music videos; all videos must allow embedded playback and must not be live.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				}, http.StatusBadRequest, false)
				return
			}

			if vibe.IsLiveVideo(req.SourceType, cachedItem.DurationSeconds) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: fmt.Errorf("error live videos cannot be added to rooms"),
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "youtube_live_video_not_supported",
							Message:   "Live videos cannot be added to rooms.",
							Propagate: true,
						},
						StatusCode: http.StatusBadRequest,
					},
					http.StatusBadRequest,
					false,
				)
				return
			}

			playbackRestriction = cachedItem.PlaybackRestriction
			if req.SourceType == vibe.SourceTypeYouTube {
				req.Title = cachedItem.Title
				publisher = cachedItem.Publisher
				req.Thumbnail = cachedItem.ThumbnailURL
				req.Duration = cachedItem.DurationSeconds
			}
		}

		playlistItem := &vibe.PlaylistItem{
			ID:                  uuid.New().String(),
			RoomID:              roomID,
			SourceType:          req.SourceType,
			SourceID:            req.SourceID,
			ProviderURL:         providerURL,
			PlaybackRestriction: playbackRestriction,
			Title:               req.Title,
			Publisher:           publisher,
			ThumbnailURL:        req.Thumbnail,
			Duration:            req.Duration,
			AddedBySessionID:    session.UserID,
			AddedAt:             time.Now(),
		}

		result, err := db.AddPlaylistItem(ctx, playlistItem)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error adding playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItems, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching playlist items: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItemsPayload, err := json.Marshal(playlistItems)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling playlist items payload in add playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		var compactEvent *vibe.RoomEventV3Payload
		if result.Outcome == vibe.AddPlaylistItemOutcomeAdded || result.Outcome == vibe.AddPlaylistItemOutcomeDuplicateVoted {
			for position, playlistItem := range playlistItems {
				if playlistItem.ID != result.PlaylistItem.ID {
					continue
				}

				compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemPositionUpdate{
					PlaylistItem: playlistItem,
					Position:     position,
				})
				if marshalErr != nil {
					handleError(
						w,
						fmt.Errorf("error marshaling compact positioned playlist item event in add playlist item: %w", marshalErr),
						http.StatusInternalServerError,
						true,
					)
					return
				}

				compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemUpdated, Payload: compactPayload}
				break
			}

			if compactEvent == nil {
				compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemIDUpdate{ID: result.PlaylistItem.ID})
				if marshalErr != nil {
					handleError(
						w,
						fmt.Errorf("error marshaling compact playlist item removal fallback in add playlist item: %w", marshalErr),
						http.StatusInternalServerError,
						true,
					)
					return
				}

				compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemRemoved, Payload: compactPayload}
			}
		}

		err = events.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: playlistItemsPayload,
			Origin:  session.EventOrigin,
			Compact: compactEvent,
		})
		if err != nil {
			log.Printf("failed to notify room: %v", err)
		}

		if result.Outcome == vibe.AddPlaylistItemOutcomeAdded && len(playlistItems) == 1 {
			playbackState := &vibe.PlaybackStateV2{
				RoomID:              roomID,
				CurrentPlaylistItem: &result.PlaylistItem,
				IsPlaying:           true,
				PositionMs:          0,
				UpdatedAt:           time.Now(),
				ServerTimeMs:        int(time.Now().UnixMilli()),
			}

			err := db.UpsertPlaybackStateV2(ctx, playbackState)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error auto-playing first playlist item: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			playbackPayload, err := json.Marshal(playbackState)
			if err != nil {
				handleError(w,
					fmt.Errorf("error marshaling playback payload in add playlist item: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			err = events.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
				Type:    vibe.PlaybackUpdate,
				Payload: playbackPayload,
				Origin:  session.EventOrigin,
			})
			if err != nil {
				log.Printf("failed to notify room: %v", err)
			}

		}

		body, err := json.Marshal(result)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		status := http.StatusOK
		if result.Outcome == vibe.AddPlaylistItemOutcomeAdded {
			status = http.StatusCreated
		}

		w.WriteHeader(status)
		_, _ = w.Write(body)

		// Activity is best-effort after the playlistItem and playback changes succeed.
		if result.Outcome == vibe.AddPlaylistItemOutcomeDuplicateAlreadyVoted {
			return
		}

		kind := vibe.MessageKindAdded
		if result.Outcome == vibe.AddPlaylistItemOutcomeDuplicateVoted {
			kind = vibe.MessageKindVoted
		}

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			log.Printf("error fetching AddPlaylistItem chat author: %v", err)
			return
		}

		message := vibe.RoomMessage{
			ID:        uuid.NewString(),
			UserID:    session.UserID,
			Name:      profile.Name,
			IsAdmin:   room.IsAdmin,
			Kind:      kind,
			Text:      result.PlaylistItem.Title,
			IsHost:    room.Mode == vibe.RoomModeHost && room.HostID == session.UserID,
			CreatedAt: time.Now().UnixMilli(),
		}

		chatPayload, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling AddPlaylistItem chat activity: %v", err)
			return
		}

		err = events.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
			Type:    vibe.MessageEvent,
			Payload: chatPayload,
		})
		if err != nil {
			log.Printf("error publishing AddPlaylistItem chat activity: %v", err)
		}
	}
}

// RemovePlaylistItem handles the removal of a playlist item from the room
//
//	@Summary	Remove a playlist item
//	@Tags		playlist-items
//	@Param		id		path	string	true	"Room ID"
//	@Param		playlistItemId	path	string	true	"Playlist item ID"
//	@Success	204
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/playlist-items/{playlistItemId} [delete]
func RemovePlaylistItem(
	db vibe.PlaylistItemQueueRemover,
	notifier vibe.RoomEventV3Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		playlistItemID := vars["playlistItemId"]

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		if !room.IsAdmin {
			handleError(
				w,
				fmt.Errorf("error forbidden"),
				http.StatusForbidden,
				false,
			)
			return
		}

		removedPlaylistItem, err := db.GetPlaylistItem(ctx, roomID, playlistItemID)
		if err != nil {
			handleError(w, fmt.Errorf("error finding playlist item to remove: %w", err), http.StatusInternalServerError, true)
			return
		}

		if removedPlaylistItem.IsEmpty() {
			handleError(w, fmt.Errorf("error finding playlist item to remove"), http.StatusNotFound, false)
			return
		}

		err = db.RemovePlaylistItem(ctx, roomID, playlistItemID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error removing playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItems, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetch playlist items in remove playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItemsPayload, err := json.Marshal(playlistItems)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling playlist items payload in remove playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		compactPayload, err := json.Marshal(vibe.PlaylistItemIDUpdate{ID: playlistItemID})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling compact remove playlist item event: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		compactEvent := &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemRemoved, Payload: compactPayload}

		err = notifier.NotifyRoomUpdateV3(ctx, roomID, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: playlistItemsPayload,
			Origin:  session.EventOrigin,
			Compact: compactEvent,
		})
		if err != nil {
			log.Printf("failed to notify room in remove playlist item: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			log.Printf("error fetching RemovePlaylistItem chat author: %v", err)
			return
		}

		message := vibe.RoomMessage{
			ID:        uuid.NewString(),
			UserID:    session.UserID,
			Name:      profile.Name,
			IsAdmin:   room.IsAdmin,
			Kind:      vibe.MessageKindDeleted,
			Text:      removedPlaylistItem.Title,
			IsHost:    room.Mode == vibe.RoomModeHost && room.HostID == session.UserID,
			CreatedAt: time.Now().UnixMilli(),
		}

		chatPayload, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling RemovePlaylistItem chat activity: %v", err)
			return
		}

		err = notifier.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
			Type:    vibe.MessageEvent,
			Payload: chatPayload,
		})
		if err != nil {
			log.Printf("error publishing RemovePlaylistItem chat activity: %v", err)
		}
	}
}

// VotePlaylistItem handles voting for a playlist item
//
//	@Summary	Vote for a playlist item
//	@Tags		playlist-items
//	@Param		id		path	string	true	"Room ID"
//	@Param		playlistItemId	path	string	true	"Playlist item ID"
//	@Success	204
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	409	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/playlist-items/{playlistItemId} [post]
func VotePlaylistItem(
	db vibe.PlaylistItemQueueVoter,
	notifier vibe.RoomEventV3Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		playlistItemID := vars["playlistItemId"]

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		userID := session.UserID

		err := db.VotePlaylistItem(ctx, roomID, playlistItemID, userID)
		if err != nil {
			var alreadyVotedError internalerror.ErrAlreadyVoted
			if errors.As(err, &alreadyVotedError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: alreadyVotedError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "song_vote_already_exists",
							Message:   "Your vote is already counted for this playlist item.",
							Propagate: true,
						},
						StatusCode: http.StatusConflict,
					},
					http.StatusConflict,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf("error voting for playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItems, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetch playlist items in vote playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItemsPayload, err := json.Marshal(playlistItems)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling playlist items payload: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		var compactEvent *vibe.RoomEventV3Payload
		votedTitle := "playlist item"
		for position, playlistItem := range playlistItems {
			if playlistItem.ID != playlistItemID {
				continue
			}

			compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemPositionUpdate{
				PlaylistItem: playlistItem,
				Position:     position,
			})
			if marshalErr != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling compact vote playlist item event: %w", marshalErr),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemUpdated, Payload: compactPayload}
			votedTitle = playlistItem.Title
			break
		}

		if compactEvent == nil {
			compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemIDUpdate{ID: playlistItemID})
			if marshalErr != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling compact vote playlist item removal fallback: %w", marshalErr),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemRemoved, Payload: compactPayload}
		}

		err = notifier.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: playlistItemsPayload,
			Origin:  session.EventOrigin,
			Compact: compactEvent,
		})
		if err != nil {
			log.Printf("failed to notify room in vote playlist item: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)

		chatRoom, err := db.GetRoomV2(ctx, roomID, userID)
		if err != nil {
			log.Printf("error fetching VotePlaylistItem chat room: %v", err)
			return
		}

		if chatRoom.IsEmpty() {
			return
		}

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			log.Printf("error fetching VotePlaylistItem chat author: %v", err)
			return
		}

		message := vibe.RoomMessage{
			ID:        uuid.NewString(),
			UserID:    session.UserID,
			Name:      profile.Name,
			IsAdmin:   chatRoom.IsAdmin,
			IsHost:    chatRoom.Mode == vibe.RoomModeHost && chatRoom.HostID == session.UserID,
			Kind:      vibe.MessageKindVoted,
			Text:      votedTitle,
			CreatedAt: time.Now().UnixMilli(),
		}

		chatPayload, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling VotePlaylistItem chat activity: %v", err)
			return
		}

		err = notifier.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
			Type:    vibe.MessageEvent,
			Payload: chatPayload,
		})
		if err != nil {
			log.Printf("error publishing VotePlaylistItem chat activity: %v", err)
		}
	}
}

// GetPlaybackStateV2 handles GET /rooms/{id}/states
//
//	@Summary	Get playback state
//	@Tags		playback
//	@Produce	json
//	@Param		id	path		string	true	"Room ID"
//	@Success	200	{object}	vibe.PlaybackStateV2
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/states [get]
func GetPlaybackStateV2(
	db vibe.PlaybackV2Fetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		state, err := db.GetPlaybackStateV2(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching playback state: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		state.ServerTimeMs = int(time.Now().UnixMilli())

		body, err := json.Marshal(state)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
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

// UpdatePlaybackStateV2 handles PUT /rooms/{id}/states
//
//	@Summary	Update playback state
//	@Tags		playback
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Room ID"
//	@Param		request	body		vibe.RoomActionRequest	true	"Playback action"
//	@Success	200		{object}	vibe.PlaybackStateV2
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	403		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/states [put]
func UpdatePlaybackStateV2(
	db vibe.RoomV2GetterPlaybackUpdater,
	events vibe.RoomRemoteEventV3Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["id"]

		var req vibe.RoomActionRequest
		err := json.UnmarshalRead(r.Body, &req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		switch req.Action {
		case vibe.RoomActionPlay, vibe.RoomActionPause, vibe.RoomActionSeek:
			// Allowed
		default:
			handleError(
				w,
				fmt.Errorf("error invalid action for state update: %s", req.Action),
				http.StatusBadRequest,
				false,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}
		userID := session.UserID

		room, err := db.GetRoomV2(ctx, roomID, userID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		var state *vibe.PlaybackStateV2

		if session.AuthType == "remote" && room.Mode == vibe.RoomModeServer {
			state, err = db.GetPlaybackStateV2(ctx, roomID)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error fetching playback for remote update: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			state.PositionMs = req.PositionMs
			if req.Action == vibe.RoomActionPlay {
				state.IsPlaying = true
			}
			if req.Action == vibe.RoomActionPause {
				state.IsPlaying = false
			}
			observedAt := time.Now()
			state.UpdatedAt = observedAt
			state.ServerTimeMs = int(observedAt.UnixMilli())
			currentPlaylistItemID := ""
			if state.CurrentPlaylistItem != nil {
				currentPlaylistItemID = state.CurrentPlaylistItem.ID
			}

			err = events.NotifyRemoteUpdateV2(context.WithoutCancel(ctx), session.RemoteID, vibe.RemoteEventV2{
				Type:                  vibe.RemoteStateUpdate,
				RoomID:                roomID,
				Origin:                vibe.RemoteOriginController,
				Online:                true,
				Paired:                true,
				CurrentPlaylistItemID: currentPlaylistItemID,
				PlaybackPositionMs:    state.PositionMs,
				PlaybackIsPlaying:     state.IsPlaying,
				PlaybackObservedAt:    observedAt,
			})
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error notifying machine of remote playback update: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			body, err := json.Marshal(state)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshalling remote playback response in update playback state handler: %w", err),
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

		if room.Mode == vibe.RoomModeServer {
			handleError(
				w,
				fmt.Errorf("error shared playback controls are unavailable in server mode"),
				http.StatusForbidden,
				false,
			)
			return
		}

		state, err = db.UpdatePlaybackV2(ctx, roomID, userID, req.Action, req.PositionMs)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error updating playback: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		state.ServerTimeMs = int(time.Now().UnixMilli())

		if room.Mode == vibe.RoomModeHost {
			statePayload, err := json.Marshal(state)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshalling playback state payload in update playback state handler: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			err = events.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
				Type:    vibe.PlaybackUpdate,
				Payload: statePayload,
				Origin:  session.EventOrigin,
			})
			if err != nil {
				log.Printf("error notifying room of playback update: %v", err)
			}
		}

		body, err := json.Marshal(state)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshalling response in update playback state handler: %w", err),
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

// SkipPlaylistItem handles POST /rooms/{id}/skips
//
//	@Summary	Skip or vote to skip the current playlistItem
//	@Tags		playback
//	@Produce	json
//	@Param		id	path		string	true	"Room ID"
//	@Success	200	{object}	vibe.SkipPlaylistItemResult
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/skips [post]
func SkipPlaylistItem(
	db vibe.RoomV2Skipper,
	notifier vibe.RoomEventBatchV3Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error missing skip session"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "skip_session_required",
						Message:   "Rejoin the room before skipping playlist items.",
						Propagate: true,
					},
					StatusCode: http.StatusUnauthorized,
				},
				http.StatusUnauthorized,
				false,
			)
			return
		}

		userID := session.UserID

		previous, previousErr := db.GetPlaybackStateV2(ctx, roomID)
		if previousErr != nil {
			log.Printf("error fetching previous playback for skip chat activity: %v", previousErr)
		}

		result, err := db.SkipPlaylistItem(ctx, roomID, userID)
		if err != nil {
			var errHostMode internalerror.ErrHostModeSkipOnly
			if errors.As(err, &errHostMode) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: fmt.Errorf("error host mode skip permission: %w", err),
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "skip_host_required",
							Message:   "Only the host or a room admin can skip playlist items in host mode.",
							Propagate: true,
						},
						StatusCode: http.StatusForbidden,
					},
					http.StatusForbidden,
					false,
				)
				return
			}

			var errDisabled internalerror.ErrSkipDisabled
			if errors.As(err, &errDisabled) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: fmt.Errorf("error skipping requires room admin: %w", err),
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "skip_room_admin_required",
							Message:   "Only room admins can skip playlist items here. Log in as a room admin in room settings.",
							Propagate: true,
						},
						StatusCode: http.StatusForbidden,
					},
					http.StatusForbidden,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf("error skip failed: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if result.Playback != nil {
			result.Playback.ServerTimeMs = int(time.Now().UnixMilli())
		}

		if !result.Skipped {
			payload := vibe.SkipVoteUpdateV2{
				UserID:        userID,
				CurrentVotes:  result.CurrentVotes,
				RequiredVotes: result.RequiredVotes,
			}

			if result.Playback != nil && result.Playback.CurrentPlaylistItem != nil {
				payload.PlaylistItemID = result.Playback.CurrentPlaylistItem.ID
			}

			votePayload, err := json.Marshal(payload)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling skip vote payload: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			err = notifier.NotifyRoomUpdatesV3(context.WithoutCancel(ctx), roomID, []vibe.RoomEventV3{
				{
					Type:    vibe.SkipVoteEvent,
					Payload: votePayload,
					UserID:  userID,
					Origin:  session.EventOrigin,
				},
			})
			if err != nil {
				log.Printf("failed to notify skip vote: %v", err)
			}
		}

		if result.Skipped {
			playlistItems, err := db.GetPlaylistItems(ctx, roomID)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error fetching playlist items: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			playlistItemsPayload, err := json.Marshal(playlistItems)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling playlist items payload: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			statePayload, err := json.Marshal(result.Playback)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling playback state payload: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			var compactEvent *vibe.RoomEventV3Payload
			for position, playlistItem := range playlistItems {
				if playlistItem.ID != result.PreviousPlaylistItemID {
					continue
				}

				compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemPositionUpdate{
					PlaylistItem: playlistItem,
					Position:     position,
				})
				if marshalErr != nil {
					handleError(
						w,
						fmt.Errorf("error marshaling compact skipped playlist item event: %w", marshalErr),
						http.StatusInternalServerError,
						true,
					)
					return
				}

				compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemUpdated, Payload: compactPayload}
				break
			}

			if compactEvent == nil {
				compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemIDUpdate{ID: result.PreviousPlaylistItemID})
				if marshalErr != nil {
					handleError(
						w,
						fmt.Errorf("error marshaling compact skipped playlist item removal fallback: %w", marshalErr),
						http.StatusInternalServerError,
						true,
					)
					return
				}

				compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemRemoved, Payload: compactPayload}
			}

			err = notifier.NotifyRoomUpdatesV3(context.WithoutCancel(ctx), roomID, []vibe.RoomEventV3{
				{
					Type:    vibe.PlaylistItemsUpdate,
					Payload: playlistItemsPayload,
					Origin:  session.EventOrigin,
					Compact: compactEvent,
				},
				{
					Type:    vibe.PlaybackUpdate,
					Payload: statePayload,
					Origin:  session.EventOrigin,
				},
			})
			if err != nil {
				log.Printf("failed to notify room updates: %v", err)
			}
		}

		body, err := json.Marshal(result)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)

		if result.AlreadyVoted || (!result.Skipped && !result.Voted) {
			return
		}

		kind := vibe.MessageKindSkipVoted
		title := "the current playlist item"
		if result.Skipped {
			kind = vibe.MessageKindSkipped
		}

		if previousErr == nil && previous.CurrentPlaylistItem != nil &&
			(!result.Skipped || previous.CurrentPlaylistItem.ID == result.PreviousPlaylistItemID) {
			title = previous.CurrentPlaylistItem.Title
		}

		room, err := db.GetRoomV2(ctx, roomID, userID)
		if err != nil {
			log.Printf("error fetching skip chat room: %v", err)
			return
		}

		if room.IsEmpty() {
			return
		}

		profile, err := db.GetOrCreateSessionProfile(ctx, userID)
		if err != nil {
			log.Printf("error fetching skip chat author: %v", err)
			return
		}

		message := vibe.RoomMessage{
			ID:        uuid.NewString(),
			UserID:    userID,
			Name:      profile.Name,
			IsAdmin:   room.IsAdmin,
			Kind:      kind,
			Text:      title,
			IsHost:    room.Mode == vibe.RoomModeHost && room.HostID == userID,
			CreatedAt: time.Now().UnixMilli(),
		}

		payload, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling skip chat activity: %v", err)
			return
		}

		err = notifier.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
			Type:    vibe.MessageEvent,
			Payload: payload,
		})
		if err != nil {
			log.Printf("error publishing skip chat activity: %v", err)
		}
	}
}

// AddPlaylistV2 handles importing resolved playlist tracks into a room.
//
//	@Summary	Add a playlist to a room
//	@Tags		playlists
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Room ID"
//	@Param		request	body		vibe.AddPlaylistRequestV2	true	"Playlist tracks"
//	@Success	202		{object}	vibe.AddPlaylistResult
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	403		{object}	vibe.ErrorResponse
//	@Failure	404		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/playlists [post]
func AddPlaylistV2(
	db vibe.PlaylistImportRoomV2Creator,
	cache vibe.CachedProviderItemsFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.AddPlaylistRequestV2
		err := json.UnmarshalRead(r.Body, &req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding playlist request body: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}
		if len(req.PlaylistItems) == 0 || len(req.PlaylistItems) > playlistImportItemLimit {
			handleError(
				w,
				fmt.Errorf(
					"error playlist must contain between 1 and %d songs",
					playlistImportItemLimit,
				),
				http.StatusBadRequest,
				false,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error importing playlist: missing session"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "playlist_session_required",
						Message:   "Rejoin the room before importing a playlist.",
						Propagate: true,
					},
					StatusCode: http.StatusUnauthorized,
				},
				http.StatusUnauthorized,
				false,
			)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room for playlist import: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}
		if !room.Settings.PlaylistImport {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error playlist import is disabled for this room"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "room_playlist_import_disabled",
						Message:   "Playlist importing is disabled in this room.",
						Propagate: true,
					},
					StatusCode: http.StatusForbidden,
				},
				http.StatusForbidden,
				false,
			)
			return
		}
		if room.Settings.OnlyAdminAddPlaylistItems && !room.IsAdmin {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error only admins can import playlists in this room"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_room_admin_required",
						Message:   "Only room admins can import playlists here. Log in as a room admin in room settings.",
						Propagate: true,
					},
					StatusCode: http.StatusForbidden,
				},
				http.StatusForbidden,
				false,
			)
			return
		}

		cacheKeys := make([]vibe.CachedProviderItemKey, 0, len(req.PlaylistItems))
		for _, requestedItem := range req.PlaylistItems {
			if requestedItem.SourceID == "" || requestedItem.SourceType == "" {
				handleError(
					w,
					fmt.Errorf("error playlist contains an empty song"),
					http.StatusBadRequest,
					false,
				)
				return
			}

			sourceEnabled := false
			for _, source := range room.Settings.EnabledSources {
				if requestedItem.SourceType == source && room.RoomType.AllowsSource(source) {
					sourceEnabled = true
					break
				}
			}
			if !sourceEnabled {
				handleError(
					w,
					fmt.Errorf(
						"error source type %s is not enabled for this room",
						requestedItem.SourceType,
					),
					http.StatusBadRequest,
					false,
				)
				return
			}
			if vibe.IsLiveVideo(requestedItem.SourceType, requestedItem.Duration) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: fmt.Errorf("error playlist contains a live video"),
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "youtube_live_video_not_supported",
							Message:   "Live videos cannot be added to rooms.",
							Propagate: true,
						},
						StatusCode: http.StatusBadRequest,
					},
					http.StatusBadRequest,
					false,
				)
				return
			}

			cacheKeys = append(cacheKeys, vibe.CachedProviderItemKey{
				Provider: requestedItem.SourceType,
				ID:       requestedItem.SourceID,
			})
		}

		cachedItems, err := cache.GetCachedProviderItems(ctx, cacheKeys)
		if err != nil {
			log.Printf("error getting cached provider playlist track metadata: %v", err)
			cachedItems = nil
		}

		cachedItemsByKey := make(map[vibe.CachedProviderItemKey]vibe.ProviderItem, len(cachedItems))
		for _, cachedItem := range cachedItems {
			key := vibe.CachedProviderItemKey{Provider: cachedItem.Source, ID: cachedItem.ID}
			cachedItemsByKey[key] = cachedItem
		}

		playlistItems := make([]vibe.PlaylistItem, 0, len(req.PlaylistItems))
		for _, requestedItem := range req.PlaylistItems {

			providerURL, err := requestedItem.CanonicalProviderURL()
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error getting canonical provider URL: %w", err),
					http.StatusBadRequest,
					false,
				)
				return
			}

			playbackRestriction := ""
			key := vibe.CachedProviderItemKey{Provider: requestedItem.SourceType, ID: requestedItem.SourceID}
			cachedItem := cachedItemsByKey[key]
			if requestedItem.SourceType == vibe.SourceTypeYouTube && cachedItem.IsEmpty() {
				handleError(w, client.ErrorCodeWrapper{
					Err: fmt.Errorf("error importing youtube playlist: verified metadata is unavailable"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "youtube_track_verification_required",
						Message:   "This playlist's availability needs to be checked again. Paste its link again before importing it.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				}, http.StatusBadRequest, false)
				return
			}

			if !cachedItem.IsEmpty() && !cachedItem.AllowedInRoom(room.RoomType) {
				continue
			}

			if !cachedItem.IsEmpty() {
				if vibe.IsLiveVideo(
					requestedItem.SourceType,
					cachedItem.DurationSeconds,
				) {
					handleError(
						w,
						client.ErrorCodeWrapper{
							Err: fmt.Errorf("error playlist contains a live video"),
							ResponseBody: client.ErrorCodeResponseBody{
								Namespace: "vibes-backend",
								Error:     "youtube_live_video_not_supported",
								Message:   "Live videos cannot be added to rooms.",
								Propagate: true,
							},
							StatusCode: http.StatusBadRequest,
						},
						http.StatusBadRequest,
						false,
					)
					return
				}
				playbackRestriction = cachedItem.PlaybackRestriction
				if requestedItem.SourceType == vibe.SourceTypeYouTube {
					requestedItem.Title = cachedItem.Title
					requestedItem.Publisher = cachedItem.Publisher
					requestedItem.Thumbnail = cachedItem.ThumbnailURL
					requestedItem.Duration = cachedItem.DurationSeconds
				}
			}

			playlistItems = append(playlistItems, vibe.PlaylistItem{
				ID:                  uuid.New().String(),
				RoomID:              roomID,
				SourceType:          requestedItem.SourceType,
				SourceID:            requestedItem.SourceID,
				ProviderURL:         providerURL,
				PlaybackRestriction: playbackRestriction,
				Title:               requestedItem.Title,
				Publisher:           requestedItem.Publisher,
				ThumbnailURL:        requestedItem.Thumbnail,
				Duration:            requestedItem.Duration,
				AddedBySessionID:    session.UserID,
				AddedAt:             time.Now(),
			})
		}

		if len(playlistItems) == 0 {
			handleError(w, fmt.Errorf("error importing playlist: no items are allowed in this room"), http.StatusBadRequest, false)
			return
		}

		importID := uuid.NewString()
		// Staged items are invisible to the worker until the import is published.
		// Bound the entire enqueue so disconnected requests cannot leave work running.
		importCtx, cancelImport := context.WithTimeout(ctx, 2*time.Minute)
		defer cancelImport()

		published := false
		defer func() {
			if published {
				return
			}

			cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
			defer cancelCleanup()

			cleanupErr := db.DeletePlaylistImport(cleanupCtx, importID)
			if cleanupErr != nil {
				log.Printf("error cleaning incomplete playlist import %s: %v", importID, cleanupErr)
			}
		}()

		for position, playlistItem := range playlistItems {
			err = db.CreatePlaylistImportItem(importCtx, importID, position, playlistItem)
			if err != nil {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: fmt.Errorf("error staging playlist item %d: %w", position, err),
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "playlist_import_failed",
							Message:   "The playlist could not be queued. Please try again.",
							Propagate: true,
						},
						StatusCode: http.StatusInternalServerError,
					},
					http.StatusInternalServerError,
					true,
				)
				return
			}
		}

		err = db.CreatePlaylistImport(importCtx, importID, roomID, session.UserID, len(playlistItems))
		if err != nil {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error creating playlist import: %w", err),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "playlist_import_failed",
						Message:   "The playlist could not be queued. Please try again.",
						Propagate: true,
					},
					StatusCode: http.StatusInternalServerError,
				},
				http.StatusInternalServerError,
				true,
			)
			return
		}

		published = true

		response := vibe.AddPlaylistResult{
			ImportID:    importID,
			QueuedCount: len(playlistItems),
		}
		body, err := json.Marshal(response)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling playlist import response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(body)
	}
}

// ReportPlaybackFailureV2 handles POST /rooms/{id}/failures
//
//	@Summary	Report a restricted playback failure
//	@Tags		playback
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Room ID"
//	@Param		request	body		vibe.PlaybackFailureRequestV2	true	"Failed playlist item"
//	@Success	200		{object}	vibe.PlaybackStateV2
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	409		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/failures [post]
func ReportPlaybackFailureV2(
	db vibe.PlaybackFailureV2Storage,
	itemFetcher vibe.ProviderItemFetcher,
	notifier vibe.RoomBatchEventV3Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["id"]

		var request vibe.PlaybackFailureRequestV2
		err := json.UnmarshalRead(r.Body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding playback failure request: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}
		if request.PlaylistItemID == "" {
			handleError(
				w,
				fmt.Errorf("error playlist item ID is required"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" || session.AuthType != "cast" {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		state, err := db.GetPlaybackStateV2(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching playback state for playback failure: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if state.CurrentPlaylistItem == nil || state.CurrentPlaylistItem.ID != request.PlaylistItemID {
			handleError(
				w,
				fmt.Errorf("error playlist item is no longer the current playlist item"),
				http.StatusConflict,
				false,
			)
			return
		}

		if state.CurrentPlaylistItem.PlaybackRestriction == "" {
			if state.CurrentPlaylistItem.SourceType != vibe.SourceTypeYouTube {
				handleError(
					w,
					fmt.Errorf("error provider does not support playback restriction verification"),
					http.StatusConflict,
					false,
				)
				return
			}

			track, err := itemFetcher.GetProviderItem(ctx, state.CurrentPlaylistItem.SourceID)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error verifying playback restriction: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			if track.PlaybackRestriction == "" {
				handleError(
					w,
					fmt.Errorf("error provider metadata does not restrict playback"),
					http.StatusConflict,
					false,
				)
				return
			}

			err = db.UpdatePlaylistItemPlaybackRestriction(
				ctx,
				roomID,
				request.PlaylistItemID,
				track.PlaybackRestriction,
			)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error storing playback restriction: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
		}

		advance, err := db.SkipRestrictedPlaylistItem(ctx, roomID, request.PlaylistItemID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error skipping failed restricted playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if advance.Playback.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error playlist item is no longer the current restricted playlist item"),
				http.StatusConflict,
				false,
			)
			return
		}

		advance.Playback.ServerTimeMs = int(time.Now().UnixMilli())
		stateBody, err := json.Marshal(advance.Playback)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling restricted playback state: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItems, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching playlist items after restricted playback skip: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		playlistItemsBody, err := json.Marshal(playlistItems)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling playlist items after restricted playback skip: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		var compactEvent *vibe.RoomEventV3Payload
		for position, playlistItem := range playlistItems {
			if playlistItem.ID != advance.PreviousPlaylistItemID {
				continue
			}

			compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemPositionUpdate{
				PlaylistItem: playlistItem,
				Position:     position,
			})
			if marshalErr != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling compact restricted skip event: %w", marshalErr),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemUpdated, Payload: compactPayload}
			break
		}
		if compactEvent == nil {
			compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemIDUpdate{ID: advance.PreviousPlaylistItemID})
			if marshalErr != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling compact restricted skip removal fallback: %w", marshalErr),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemRemoved, Payload: compactPayload}
		}

		playbackBody, err := json.Marshal(advance.Playback)
		if err != nil {
			handleError(w, fmt.Errorf("error marshaling restricted playback: %w", err), http.StatusInternalServerError, true)
			return
		}

		err = notifier.NotifyRoomUpdatesV3(context.WithoutCancel(ctx), roomID, []vibe.RoomEventV3{
			{
				Type:    vibe.PlaylistItemsUpdate,
				Payload: playlistItemsBody,
				Compact: compactEvent,
			},
			{
				Type:    vibe.PlaybackUpdate,
				Payload: playbackBody,
			},
		})
		if err != nil {
			log.Printf("error notifying room of restricted playback skip: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(stateBody)
	}
}

// CreateGeneratedRoomV2 handles POST /api/v2/rooms/generation.
//
//	@Summary	Create a room and queue playlist generation
//	@Tags		rooms
//	@Accept		json
//	@Produce	json
//	@Param		request	body		vibe.GeneratedRoomRequestV2	true	"Playlist prompt and immutable room type"
//	@Success	201	{object}	vibe.RoomV2
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	429	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Failure	409	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/generation [post]
func CreateGeneratedRoomV2(
	db vibe.GeneratedRoomV2Creator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error reading playlist request in CreateGeneratedRoomV2 handler: %w",
					err,
				),
				http.StatusBadRequest,
				false,
			)
			return
		}

		var request vibe.GeneratedRoomRequestV2
		err = json.Unmarshal(body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error decoding playlist request in CreateGeneratedRoomV2 handler: %w",
					err,
				),
				http.StatusBadRequest,
				false,
			)
			return
		}

		if request.RoomType == "" {
			request.RoomType = vibe.RoomTypeMusic
		}

		if !request.RoomType.IsValid() {
			handleError(w, fmt.Errorf("error generating room: roomType must be MUSIC or WATCH"), http.StatusBadRequest, false)
			return
		}

		request.Prompt = strings.TrimSpace(request.Prompt)
		if request.Prompt == "" {
			handleError(
				w,
				fmt.Errorf(
					"error validating playlist request in CreateGeneratedRoomV2 handler: prompt is required",
				),
				http.StatusBadRequest,
				false,
			)
			return
		}
		if utf8.RuneCountInString(request.Prompt) > generatedPlaylistPromptMaxLength {
			handleError(
				w,
				fmt.Errorf(
					"error validating playlist request in CreateGeneratedRoomV2 handler: prompt exceeds %d characters",
					generatedPlaylistPromptMaxLength,
				),
				http.StatusBadRequest,
				false,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf(
					"error getting session in CreateGeneratedRoomV2 handler: session is missing",
				),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		hasActiveGeneration, err := db.HasActiveRoomGeneration(ctx)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error checking active room generation in CreateGeneratedRoomV2 handler: %w",
					err,
				),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if hasActiveGeneration {
			w.Header().Set("Retry-After", roomGenerationBusyRetryAfterSeconds)
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf(
						"error validating room generation in CreateGeneratedRoomV2 handler: active generation already exists",
					),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "room_generation_busy",
						Message:   "A playlist is already being generated. Please wait and try again.",
						Propagate: true,
					},
					StatusCode: http.StatusTooManyRequests,
				},
				http.StatusTooManyRequests,
				false,
			)
			return
		}

		reservation, err := db.ReserveSuggestedRoomName(ctx, session.UserID)
		if err != nil {
			var unavailableError internalerror.ErrRoomNameUnavailable
			if errors.As(err, &unavailableError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: unavailableError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_name_unavailable",
							Message:   "This room name is unavailable or its reservation expired.",
							Propagate: true,
						},
						StatusCode: http.StatusConflict,
					},
					http.StatusConflict,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf(
					"error reserving generated room name in CreateGeneratedRoomV2 handler: %w",
					err,
				),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		settings, err := vibe.DefaultRoomSettingsV2()
		if err != nil {
			handleError(w, fmt.Errorf("error getting generated room defaults: %w", err), http.StatusInternalServerError, true)
			return
		}

		if request.RoomType == vibe.RoomTypeWatch {
			settings.EnabledSources = []string{vibe.SourceTypeYouTube}
		}

		room := vibe.RoomV2{
			ID:            helper.Slugify(reservation.Name),
			Name:          reservation.Name,
			RoomType:      request.RoomType,
			Mode:          vibe.RoomModeServer,
			HostID:        session.UserID,
			Settings:      *settings,
			CreatedAt:     time.Now(),
			ActiveSources: settings.EnabledSources,
		}
		createdRoom, err := db.CreateRoomV2(ctx, &room, reservation.Token)
		if err != nil {
			var unavailableError internalerror.ErrRoomNameUnavailable
			if errors.As(err, &unavailableError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: unavailableError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_name_unavailable",
							Message:   "This room name is unavailable or its reservation expired.",
							Propagate: true,
						},
						StatusCode: http.StatusConflict,
					},
					http.StatusConflict,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf(
					"error creating generated room in CreateGeneratedRoomV2 handler: %w",
					err,
				),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = db.CreateRoomGeneration(ctx, createdRoom.ID, request.Prompt)
		if err != nil {
			var busyError internalerror.ErrRoomGenerationBusy
			if errors.As(err, &busyError) {
				w.Header().Set("Retry-After", roomGenerationBusyRetryAfterSeconds)
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: busyError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_generation_busy",
							Message:   "A playlist is already being generated. Please wait and try again.",
							Propagate: true,
						},
						StatusCode: http.StatusTooManyRequests,
					},
					http.StatusTooManyRequests,
					false,
				)
				return
			}

			var dailyLimitError internalerror.ErrRoomGenerationDailyLimit
			if errors.As(err, &dailyLimitError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: dailyLimitError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_generation_daily_limit",
							Message:   "This room has reached its daily playlist generation limit.",
							Propagate: true,
						},
						StatusCode: http.StatusTooManyRequests,
					},
					http.StatusTooManyRequests,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf(
					"error queueing room generation in CreateGeneratedRoomV2 handler: %w",
					err,
				),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		createdRoom.IsGenerating = true
		createdRoom.GenerationCount = 1

		body, err = json.Marshal(createdRoom)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error marshaling generated room in CreateGeneratedRoomV2 handler: %w",
					err,
				),
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

// SearchYouTubeV2 handles GET /api/v2/rooms/{id}/search/youtube
//
//	@Summary	Search YouTube items
//	@Tags		providers
//	@Produce	json
//	@Param		q	query		string	true	"Search query"
//	@Param		id	path	string	true	"Room ID"
//	@Success	200	{array}		vibe.ProviderItem
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Failure	503	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/search/youtube [get]
func SearchYouTubeV2(
	ms vibe.ProviderItemsSearcher,
	cache vibe.ProviderSearchCache,
	db vibe.RoomSearchUsageCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		vars := mux.Vars(r)
		roomID := vars["id"]
		if utf8.RuneCountInString(roomID) > 200 {
			handleError(w, fmt.Errorf("error validating room attribution: room ID is too long"), http.StatusBadRequest, true)
			return
		}

		if utf8.RuneCountInString(query) < minimumSearchQueryLength {
			handleError(
				w,
				fmt.Errorf(
					"error validating query in SearchYouTubeV2 handler: parameter 'q' must contain at least %d characters",
					minimumSearchQueryLength,
				),
				http.StatusBadRequest,
				true,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(w, fmt.Errorf("error searching room: missing session"), http.StatusUnauthorized, false)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(w, fmt.Errorf("error fetching search room: %w", err), http.StatusInternalServerError, true)
			return
		}
		if room.IsEmpty() {
			handleError(w, fmt.Errorf("error searching room: room not found"), http.StatusNotFound, false)
			return
		}
		if !room.RoomType.AllowsSource(vibe.SourceTypeYouTube) || !slices.Contains(room.Settings.EnabledSources, vibe.SourceTypeYouTube) {
			handleError(w, fmt.Errorf("error searching room: provider is disabled"), http.StatusBadRequest, false)
			return
		}

		cachedSearches, err := cache.GetCachedProviderSearches(
			ctx,
			vibe.SourceTypeYouTube,
			[]string{query},
			room.RoomType,
		)
		if err != nil {
			log.Printf("error getting cached youtube search: %v", err)
			cachedSearches = []vibe.CachedProviderSearch{}
		}

		items := make([]vibe.ProviderItem, 0)
		cacheHit := len(cachedSearches) > 0
		if cacheHit {
			cachedItems := cachedSearches[0].GetProviderItems()
			for _, item := range cachedItems {
				if vibe.IsLiveVideo(item.Source, item.DurationSeconds) ||
					item.PlaybackRestriction == vibe.PlaybackRestrictionEmbedding {
					continue
				}
				items = append(items, item)
			}
		}
		usage := vibe.GenerateSearchUsage(
			vibe.SourceTypeYouTube,
			query,
			cacheHit,
		)
		usage.RoomID = roomID

		err = db.CreateSearchUsages(ctx, []vibe.SearchUsage{usage})
		if err != nil {
			log.Printf("error creating youtube search usage: %v", err)
		}
		if !cacheHit {
			var quotaReset time.Time
			quotaReset, err = cache.GetProviderQuotaReset(
				ctx,
				vibe.SourceTypeYouTube,
				vibe.ProviderQuotaOperationSearch,
			)
			quotaCheckSucceeded := err == nil
			if !quotaCheckSucceeded {
				log.Printf("error getting cached youtube quota reset: %v", err)
			}
			quotaExceeded := time.Now().Before(quotaReset)
			if quotaCheckSucceeded && quotaExceeded {
				err = internalerror.ErrProviderQuotaExceeded{
					Err: fmt.Errorf(
						"error checking youtube search quota in SearchYouTubeV2 handler: cached until %s",
						quotaReset.Format(time.RFC3339),
					),
					Provider: vibe.SourceTypeYouTube,
					ResetAt:  quotaReset,
				}
			}
			if !quotaCheckSucceeded || !quotaExceeded {
				items, err = ms.SearchProviderItems(ctx, query, room.RoomType)
			}
		}
		if err != nil {
			var quotaError internalerror.ErrProviderQuotaExceeded
			if errors.As(err, &quotaError) {
				err = cache.CacheProviderQuotaReset(
					ctx,
					vibe.SourceTypeYouTube,
					vibe.ProviderQuotaOperationSearch,
					quotaError.ResetAt,
				)
				if err != nil {
					log.Printf("error caching youtube quota reset: %v", err)
				}

				w.Header().Set("Retry-After", quotaError.ResetAt.UTC().Format(http.TimeFormat))

				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: quotaError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "youtube_search_quota_exhausted",
							Message:   vibe.RoomGenerationYouTubeQuotaFailure,
							Propagate: true,
						},
						StatusCode: http.StatusServiceUnavailable,
					},
					http.StatusServiceUnavailable,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf("error searching music in SearchYouTubeV2 handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if !cacheHit {
			search := vibe.CachedProviderSearch{Query: query, Items: items}
			err = cache.CacheProviderSearches(
				ctx,
				vibe.SourceTypeYouTube,
				[]vibe.CachedProviderSearch{
					search,
				},
				room.RoomType,
			)
			if err != nil {
				log.Printf("error caching youtube search: %v", err)
			}
		}

		err = cache.CacheProviderItems(ctx, vibe.SourceTypeYouTube, items)
		if err != nil {
			log.Printf("error caching youtube item metadata: %v", err)
		}

		body, err := json.Marshal(items)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling response in SearchYouTubeV2 handler: %w", err),
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

// SearchSoundCloudV2 handles GET /api/v2/rooms/{id}/search/soundcloud
//
//	@Summary	Search SoundCloud items
//	@Tags		providers
//	@Produce	json
//	@Param		q	query		string	true	"Search query"
//	@Param		id	path	string	true	"Room ID"
//	@Success	200	{array}		vibe.ProviderItem
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/search/soundcloud [get]
func SearchSoundCloudV2(
	ms vibe.ProviderItemsSearcher,
	cache vibe.CachedProviderSearchItemFetcherCreator,
	db vibe.RoomSearchUsageCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		vars := mux.Vars(r)
		roomID := vars["id"]
		if utf8.RuneCountInString(roomID) > 200 {
			handleError(w, fmt.Errorf("error validating room attribution: room ID is too long"), http.StatusBadRequest, true)
			return
		}

		if utf8.RuneCountInString(query) < minimumSearchQueryLength {
			handleError(
				w,
				fmt.Errorf(
					"error validating query in SearchSoundCloudV2 handler: parameter 'q' must contain at least %d characters",
					minimumSearchQueryLength,
				),
				http.StatusBadRequest,
				true,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(w, fmt.Errorf("error searching room: missing session"), http.StatusUnauthorized, false)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(w, fmt.Errorf("error fetching search room: %w", err), http.StatusInternalServerError, true)
			return
		}
		if room.IsEmpty() {
			handleError(w, fmt.Errorf("error searching room: room not found"), http.StatusNotFound, false)
			return
		}
		if !room.RoomType.AllowsSource(vibe.SourceTypeSoundCloud) || !slices.Contains(room.Settings.EnabledSources, vibe.SourceTypeSoundCloud) {
			handleError(w, fmt.Errorf("error searching room: provider is disabled"), http.StatusBadRequest, false)
			return
		}

		cachedSearches, err := cache.GetCachedProviderSearches(
			ctx,
			vibe.SourceTypeSoundCloud,
			[]string{query},
			room.RoomType,
		)
		if err != nil {
			log.Printf("error getting cached soundcloud search: %v", err)
			cachedSearches = []vibe.CachedProviderSearch{}
		}

		items := make([]vibe.ProviderItem, 0)
		cacheHit := len(cachedSearches) > 0
		if cacheHit {
			items = cachedSearches[0].GetProviderItems()
		}
		usage := vibe.GenerateSearchUsage(
			vibe.SourceTypeSoundCloud,
			query,
			cacheHit,
		)
		usage.RoomID = roomID

		err = db.CreateSearchUsages(ctx, []vibe.SearchUsage{usage})
		if err != nil {
			log.Printf("error creating soundcloud search usage: %v", err)
		}
		if !cacheHit {
			items, err = ms.SearchProviderItems(ctx, query, room.RoomType)
		}
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error searching music in SearchSoundCloudV2 handler: %w",
					err,
				),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if !cacheHit {
			search := vibe.CachedProviderSearch{Query: query, Items: items}
			err = cache.CacheProviderSearches(
				ctx,
				vibe.SourceTypeSoundCloud,
				[]vibe.CachedProviderSearch{
					search,
				},
				room.RoomType,
			)
			if err != nil {
				log.Printf("error caching soundcloud search: %v", err)
			}
		}

		err = cache.CacheProviderItems(ctx, vibe.SourceTypeSoundCloud, items)
		if err != nil {
			log.Printf("error caching soundcloud item metadata: %v", err)
		}

		body, err := json.Marshal(items)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error marshaling response in SearchSoundCloudV2 handler: %w",
					err,
				),
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

const publicRoomPageSize = 10

const publicRoomMaximumPageSize = 100

const publicRoomMaximumQueryLength = 100

// GetPublicRoomsV2 handles GET /api/v2/rooms/public.
//
//	@Summary		Browse public rooms
//	@Description	Returns public rooms ordered by listener count, song count, then ID, all descending. Ranges are zero-based and inclusive. Only rooms with protected admin controls are listed.
//	@Tags			rooms
//	@Produce		json
//	@Param			q		query		string	false	"Case-insensitive room name search (literal substring, up to 100 characters)"
//	@Param			live	query		boolean	false	"Only rooms with active listeners" default(false)
//	@Param			from	query		int		false	"First row, zero-based" minimum(0) default(0)
//	@Param			to		query		int		false	"Last row, inclusive. Defaults to from + 9; at most 100 rooms per page" minimum(0)
//	@Success		200		{object}	vibe.PublicRoomResult
//	@Failure		400		{object}	vibe.ErrorResponse
//	@Failure		500		{object}	vibe.ErrorResponse
//	@Router			/api/v2/rooms/public [get]
//
// Deprecated: Use GET /api/v3/rooms/public. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use GET /api/v3/rooms/public for the playlist-item contract. This endpoint retains its existing payloads.
func GetPublicRoomsV2(db vibe.PublicRoomsV3Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if !utf8.ValidString(query) || strings.ContainsRune(query, '\x00') {
			handleError(w, fmt.Errorf("error invalid room search text"), http.StatusBadRequest, false)
			return
		}

		if utf8.RuneCountInString(query) > publicRoomMaximumQueryLength {
			handleError(w, fmt.Errorf("error room search is too long"), http.StatusBadRequest, false)
			return
		}

		live := r.URL.Query().Get("live")
		if live != "" && live != "true" && live != "false" {
			handleError(w, fmt.Errorf("error invalid live room filter"), http.StatusBadRequest, false)
			return
		}

		var from int64
		var err error
		fromValue := r.URL.Query().Get("from")
		if fromValue != "" {
			from, err = strconv.ParseInt(fromValue, 10, 32)
			if err != nil {
				handleError(w, fmt.Errorf("error parsing first room row: %w", err), http.StatusBadRequest, false)
				return
			}

			if from < 0 {
				handleError(w, fmt.Errorf("error invalid first room row"), http.StatusBadRequest, false)
				return
			}
		}

		to := from + publicRoomPageSize - 1
		toValue := r.URL.Query().Get("to")
		if toValue != "" {
			to, err = strconv.ParseInt(toValue, 10, 32)
			if err != nil {
				handleError(w, fmt.Errorf("error parsing last room row: %w", err), http.StatusBadRequest, false)
				return
			}

			if to < from {
				handleError(w, fmt.Errorf("error invalid last room row"), http.StatusBadRequest, false)
				return
			}
		}

		if to-from+1 > publicRoomMaximumPageSize {
			handleError(w, fmt.Errorf("error room page is too large"), http.StatusBadRequest, false)
			return
		}

		result, err := db.SearchPublicRoomsV3(ctx, vibe.PublicRoomSearch{
			Query: query,
			Live:  live == "true",
			From:  int(from),
			To:    int(to),
		})
		if err != nil {
			handleError(w, fmt.Errorf("error searching public rooms: %w", err), http.StatusInternalServerError, true)
			return
		}

		legacy := result.ToPublicRoomResult()

		body, err := json.Marshal(legacy)
		if err != nil {
			handleError(w, fmt.Errorf("error marshaling public rooms: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
