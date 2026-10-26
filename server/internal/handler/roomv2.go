package handler

import (
	"context"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
	"golang.org/x/crypto/bcrypt"
)

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
		err := jsonv2.UnmarshalRead(r.Body, &req)
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

		_, isModerator := helper.GetAdminUserFromContext(ctx)

		message := vibe.RoomMessage{
			IsModerator: isModerator,
			ID:          uuid.NewString(),
			UserID:      session.UserID,
			Name:        profile.Name,
			IsAdmin:     true,
			Kind:        vibe.MessageKindAdded,
			IsHost:      room.Mode == vibe.RoomModeHost && room.HostID == session.UserID,
			Activity:    true,
			Text:        "a password to the room",
			CreatedAt:   time.Now().UnixMilli(),
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
	notifier vibe.AdminRoomChangeNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req vibe.CreateRoomRequestV2
		err := jsonv2.UnmarshalRead(http.MaxBytesReader(w, r.Body, 8192), &req)
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
			handleError(
				w,
				fmt.Errorf("error creating room: roomType must be MUSIC or WATCH"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		if !req.Validate() {
			handleError(
				w,
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error validating room name"),
					ResponseBody: internalerror.ErrorCodeResponseBody{
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
			handleError(
				w,
				fmt.Errorf("error getting default room settings: %w", err),
				http.StatusInternalServerError,
				true,
			)
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
				handleError(
					w,
					fmt.Errorf("error creating room: provider %q is not allowed for %s", source, req.RoomType),
					http.StatusBadRequest,
					false,
				)
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
				internalerror.ErrorCodeWrapper{
					Err: internalerror.ErrMissingAdminPassword{
						Err: fmt.Errorf("error room must have a password to be public"),
					},
					ResponseBody: internalerror.ErrorCodeResponseBody{
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
					internalerror.ErrorCodeWrapper{
						Err: unavailableError,
						ResponseBody: internalerror.ErrorCodeResponseBody{
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

		err = notifier.NotifyAdminRoomChange(context.WithoutCancel(ctx), created.ID)
		if err != nil {
			log.Printf("error notifying created admin room: %v", err)
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
		err := jsonv2.UnmarshalRead(r.Body, &req)
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
			handleError(
				w,
				fmt.Errorf("error updating room settings: missing session"),
				http.StatusUnauthorized,
				false,
			)
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
			handleError(
				w,
				fmt.Errorf("error updating room settings: admin required"),
				http.StatusForbidden,
				false,
			)
			return
		}

		previousSettings := room.Settings
		previousMode := room.Mode

		if req.Settings != nil {
			for _, source := range req.Settings.EnabledSources {
				if !room.RoomType.AllowsSource(source) {
					handleError(
						w,
						fmt.Errorf("error updating room: provider %q is not allowed for %s", source, room.RoomType),
						http.StatusBadRequest,
						false,
					)
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

		if room.Settings.Public && room.Flagged {
			handleError(
				w,
				fmt.Errorf("error flagged rooms cannot be made public"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		if room.Settings.Public && !room.HasPassword {
			handleError(
				w,
				internalerror.ErrorCodeWrapper{
					Err: internalerror.ErrMissingAdminPassword{
						Err: fmt.Errorf("error room must have a password to be public"),
					},
					ResponseBody: internalerror.ErrorCodeResponseBody{
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
			_, isModerator := helper.GetAdminUserFromContext(ctx)

			message := vibe.RoomMessage{
				IsModerator: isModerator,
				ID:          uuid.NewString(),
				UserID:      session.UserID,
				Name:        profile.Name,
				IsAdmin:     updated.IsAdmin,
				IsHost:      updated.Mode == vibe.RoomModeHost && updated.HostID == session.UserID,
				Kind:        vibe.MessageKindSettings,
				Text:        change,
				CreatedAt:   time.Now().UnixMilli(),
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
		err := jsonv2.UnmarshalRead(r.Body, &req)
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
			var permissionError internalerror.ErrorCodeWrapper
			isHostDenial := errors.As(err, &permissionError) &&
				permissionError.StatusCode == http.StatusForbidden &&
				permissionError.ResponseBody.Error == "playback_host_required"

			handleError(
				w,
				fmt.Errorf("error updating playback: %w", err),
				http.StatusInternalServerError,
				!isHostDenial,
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
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error missing skip session"),
					ResponseBody: internalerror.ErrorCodeResponseBody{
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
					internalerror.ErrorCodeWrapper{
						Err: fmt.Errorf("error host mode skip permission: %w", err),
						ResponseBody: internalerror.ErrorCodeResponseBody{
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
					internalerror.ErrorCodeWrapper{
						Err: fmt.Errorf("error skipping requires room admin: %w", err),
						ResponseBody: internalerror.ErrorCodeResponseBody{
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

		_, isModerator := helper.GetAdminUserFromContext(ctx)

		message := vibe.RoomMessage{
			IsModerator: isModerator,
			ID:          uuid.NewString(),
			UserID:      userID,
			Name:        profile.Name,
			IsAdmin:     room.IsAdmin,
			Kind:        kind,
			Text:        title,
			IsHost:      room.Mode == vibe.RoomModeHost && room.HostID == userID,
			CreatedAt:   time.Now().UnixMilli(),
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
		err := jsonv2.UnmarshalRead(r.Body, &req)
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
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error importing playlist: missing session"),
					ResponseBody: internalerror.ErrorCodeResponseBody{
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
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error playlist import is disabled for this room"),
					ResponseBody: internalerror.ErrorCodeResponseBody{
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
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error only admins can import playlists in this room"),
					ResponseBody: internalerror.ErrorCodeResponseBody{
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
					internalerror.ErrorCodeWrapper{
						Err: fmt.Errorf("error playlist contains a live video"),
						ResponseBody: internalerror.ErrorCodeResponseBody{
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
		seenItems := make(map[vibe.CachedProviderItemKey]bool)
		for _, requestedItem := range req.PlaylistItems {
			key := vibe.CachedProviderItemKey{Provider: requestedItem.SourceType, ID: requestedItem.SourceID}
			if seenItems[key] {
				continue
			}
			seenItems[key] = true

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
			cachedItem := cachedItemsByKey[key]
			if cachedItem.IsEmpty() {
				verificationError := "soundcloud_track_verification_required"
				if requestedItem.SourceType == vibe.SourceTypeYouTube {
					verificationError = "youtube_track_verification_required"
				}

				handleError(
					w,
					internalerror.ErrorCodeWrapper{
						Err: fmt.Errorf("error importing playlist: verified metadata is unavailable"),
						ResponseBody: internalerror.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     verificationError,
							Message:   "This playlist's availability needs to be checked again. Paste its link again before importing it.",
							Propagate: true,
						},
						StatusCode: http.StatusBadRequest,
					},
					http.StatusBadRequest,
					false,
				)
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
						internalerror.ErrorCodeWrapper{
							Err: fmt.Errorf("error playlist contains a live video"),
							ResponseBody: internalerror.ErrorCodeResponseBody{
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
				requestedItem.Title = cachedItem.Title
				requestedItem.Publisher = cachedItem.Publisher
				requestedItem.Thumbnail = cachedItem.ThumbnailURL
				requestedItem.Duration = cachedItem.DurationSeconds
				providerURL = cachedItem.ProviderURL
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
			handleError(
				w,
				fmt.Errorf("error importing playlist: no items are allowed in this room"),
				http.StatusBadRequest,
				false,
			)
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

		err = db.CreatePlaylistImportItems(importCtx, importID, playlistItems)
		if err != nil {
			handleError(
				w,
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error staging playlist items: %w", err),
					ResponseBody: internalerror.ErrorCodeResponseBody{
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

		err = db.CreatePlaylistImport(importCtx, importID, roomID, session.UserID, len(playlistItems))
		if err != nil {
			handleError(
				w,
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error creating playlist import: %w", err),
					ResponseBody: internalerror.ErrorCodeResponseBody{
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
		err := jsonv2.UnmarshalRead(r.Body, &request)
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
			handleError(
				w,
				fmt.Errorf("error marshaling restricted playback: %w", err),
				http.StatusInternalServerError,
				true,
			)
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
	notifier vibe.AdminRoomChangeNotifier,
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
			handleError(
				w,
				fmt.Errorf("error generating room: roomType must be MUSIC or WATCH"),
				http.StatusBadRequest,
				false,
			)
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
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf(
						"error validating room generation in CreateGeneratedRoomV2 handler: active generation already exists",
					),
					ResponseBody: internalerror.ErrorCodeResponseBody{
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
					internalerror.ErrorCodeWrapper{
						Err: unavailableError,
						ResponseBody: internalerror.ErrorCodeResponseBody{
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
			handleError(
				w,
				fmt.Errorf("error getting generated room defaults: %w", err),
				http.StatusInternalServerError,
				true,
			)
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
					internalerror.ErrorCodeWrapper{
						Err: unavailableError,
						ResponseBody: internalerror.ErrorCodeResponseBody{
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

		err = notifier.NotifyAdminRoomChange(context.WithoutCancel(ctx), createdRoom.ID)
		if err != nil {
			log.Printf("error notifying generated admin room: %v", err)
		}

		err = db.CreateRoomGeneration(ctx, createdRoom.ID, request.Prompt)
		if err != nil {
			var busyError internalerror.ErrRoomGenerationBusy
			if errors.As(err, &busyError) {
				w.Header().Set("Retry-After", roomGenerationBusyRetryAfterSeconds)
				handleError(
					w,
					internalerror.ErrorCodeWrapper{
						Err: busyError,
						ResponseBody: internalerror.ErrorCodeResponseBody{
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
					internalerror.ErrorCodeWrapper{
						Err: dailyLimitError,
						ResponseBody: internalerror.ErrorCodeResponseBody{
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
			handleError(
				w,
				fmt.Errorf("error validating room attribution: room ID is too long"),
				http.StatusBadRequest,
				true,
			)
			return
		}

		if utf8.RuneCountInString(query) < minimumSearchQueryLength || utf8.RuneCountInString(query) > 200 {
			handleError(
				w,
				fmt.Errorf(
					"error validating query in SearchYouTubeV2 handler: parameter 'q' must contain between %d and 200 characters",
					minimumSearchQueryLength,
				),
				http.StatusBadRequest,
				true,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error searching room: missing session"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching search room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error searching room: room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}
		if !room.RoomType.AllowsSource(vibe.SourceTypeYouTube) || !slices.Contains(room.Settings.EnabledSources, vibe.SourceTypeYouTube) {
			handleError(
				w,
				fmt.Errorf("error searching room: provider is disabled"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		result, err := cache.GetOrClaimProviderSearch(ctx, vibe.SourceTypeYouTube, query, room.RoomType)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error reading youtube search cache: %w", err),
				http.StatusServiceUnavailable,
				true,
			)
			return
		}

		defer func() {
			releaseErr := cache.ReleaseSearch(ctx, result.Claim)
			if releaseErr != nil {
				log.Printf("error releasing youtube search claim: %v", releaseErr)
			}
		}()

		cachedSearches := result.Searches

		cacheHit := len(cachedSearches) > 0
		items := make([]vibe.ProviderItem, 0)
		usage, err := vibe.GenerateSearchUsage(vibe.SourceTypeYouTube, query, cacheHit)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error generating youtube search usage: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		usage.RoomID = roomID
		if !cacheHit {
			var quotaReset time.Time
			quotaReset, err = cache.GetProviderQuotaReset(ctx, vibe.SourceTypeYouTube, vibe.ProviderQuotaOperationSearch)
			if err == nil && time.Now().Before(quotaReset) {
				log.Printf("event=search_quota_blocked provider=youtube purpose=interactive upstream_attempted=false reset_at=%s", quotaReset.UTC().Format(time.RFC3339))

				err = internalerror.ErrProviderQuotaExceeded{
					Err:      fmt.Errorf("error checking youtube search quota: exhausted"),
					Provider: vibe.SourceTypeYouTube,
					ResetAt:  quotaReset,
				}
			}
			if err == nil {
				items, err = ms.SearchProviderItems(ctx, query, room.RoomType)
				var quotaError internalerror.ErrProviderQuotaExceeded
				if errors.As(err, &quotaError) {
					log.Printf("event=search_quota_exhausted provider=youtube purpose=interactive upstream_attempted=true reset_at=%s", quotaError.ResetAt.UTC().Format(time.RFC3339))

					cacheErr := cache.CacheProviderQuotaReset(ctx, vibe.SourceTypeYouTube, vibe.ProviderQuotaOperationSearch, quotaError.ResetAt)
					if cacheErr != nil {
						log.Printf("error caching youtube quota reset: %v", cacheErr)
					}
					w.Header().Set("Retry-After", quotaError.ResetAt.UTC().Format(http.TimeFormat))
				}
				recordErr := db.CreateSearchUsages(ctx, []vibe.SearchUsage{*usage})
				if recordErr != nil {
					log.Printf("error creating youtube search usage: %v", recordErr)
				}
				if err == nil {
					search := vibe.CachedProviderSearch{Query: query, Items: items, FetchedAt: time.Now().UTC()}
					cacheErr := cache.CacheProviderSearches(ctx, vibe.SourceTypeYouTube, []vibe.CachedProviderSearch{search}, room.RoomType)
					if cacheErr != nil {
						log.Printf("error caching youtube search: %v", cacheErr)
					}
					cacheErr = cache.CacheProviderItems(ctx, vibe.SourceTypeYouTube, items)
					if cacheErr != nil {
						log.Printf("error caching fresh youtube metadata: %v", cacheErr)
					}
				}
			}
		}
		if err != nil {
			searchErr := err
			retained, retainedErr := cache.GetRetainedProviderSearches(ctx, vibe.SourceTypeYouTube, []string{query}, room.RoomType)
			if retainedErr != nil || len(retained) == 0 {
				var quotaError internalerror.ErrProviderQuotaExceeded
				if errors.As(searchErr, &quotaError) {
					handleError(
						w,
						internalerror.ErrorCodeWrapper{
							Err: quotaError,
							ResponseBody: internalerror.ErrorCodeResponseBody{
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
					fmt.Errorf("error searching youtube: %w", searchErr),
					http.StatusServiceUnavailable,
					false,
				)
				return
			}
			cachedSearches = retained
			cacheHit = true
			log.Printf("event=search_retained_fallback provider=youtube purpose=interactive")

			w.Header().Del("Retry-After")
			w.Header().Set("X-Search-Cache", "stale")
		}
		if cacheHit {
			for _, item := range cachedSearches[0].GetProviderItems() {
				if item.AllowedInRoom(room.RoomType) {
					items = append(items, item)
				}
			}
			usage.Cached = true
			err = db.CreateSearchUsages(ctx, []vibe.SearchUsage{*usage})
			if err != nil {
				log.Printf("error recording cached youtube search usage: %v", err)
			}
		}

		if items == nil {
			items = []vibe.ProviderItem{}
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
	cache vibe.ProviderSearchClaimCache,
	db vibe.RoomSearchUsageCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		vars := mux.Vars(r)
		roomID := vars["id"]
		if utf8.RuneCountInString(roomID) > 200 {
			handleError(
				w,
				fmt.Errorf("error validating room attribution: room ID is too long"),
				http.StatusBadRequest,
				true,
			)
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
			handleError(
				w,
				fmt.Errorf("error searching room: missing session"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching search room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error searching room: room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}
		if !room.RoomType.AllowsSource(vibe.SourceTypeSoundCloud) || !slices.Contains(room.Settings.EnabledSources, vibe.SourceTypeSoundCloud) {
			handleError(
				w,
				fmt.Errorf("error searching room: provider is disabled"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		result, err := cache.GetOrClaimProviderSearch(ctx, vibe.SourceTypeSoundCloud, query, room.RoomType)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error reading soundcloud search cache: %w", err),
				http.StatusServiceUnavailable,
				true,
			)
			return
		}

		defer func() {
			releaseErr := cache.ReleaseSearch(ctx, result.Claim)
			if releaseErr != nil {
				log.Printf("error releasing soundcloud search claim: %v", releaseErr)
			}
		}()

		cachedSearches := result.Searches

		items := make([]vibe.ProviderItem, 0)
		cacheHit := len(cachedSearches) > 0
		if cacheHit {
			items = cachedSearches[0].GetProviderItems()
		}
		usage, err := vibe.GenerateSearchUsage(
			vibe.SourceTypeSoundCloud,
			query,
			cacheHit,
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error generating soundcloud search usage: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		usage.RoomID = roomID
		if !cacheHit {
			items, err = ms.SearchProviderItems(ctx, query, room.RoomType)
			if err == nil {
				search := vibe.CachedProviderSearch{Query: query, Items: items, FetchedAt: time.Now().UTC()}
				cacheErr := cache.CacheProviderSearches(ctx, vibe.SourceTypeSoundCloud, []vibe.CachedProviderSearch{search}, room.RoomType)
				if cacheErr != nil {
					log.Printf("error caching soundcloud search: %v", cacheErr)
				}
			}
		}
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error searching soundcloud: %w", err),
				http.StatusServiceUnavailable,
				true,
			)
			return
		}
		if cacheHit {
			items = cachedSearches[0].GetProviderItems()
		}

		usage.Cached = cacheHit
		err = db.CreateSearchUsages(ctx, []vibe.SearchUsage{*usage})
		if err != nil {
			log.Printf("error creating soundcloud search usage: %v", err)
		}

		err = cache.CacheProviderItems(ctx, vibe.SourceTypeSoundCloud, items)
		if err != nil {
			log.Printf("error caching soundcloud item metadata: %v", err)
		}

		if items == nil {
			items = []vibe.ProviderItem{}
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

const minimumSearchQueryLength = 3
