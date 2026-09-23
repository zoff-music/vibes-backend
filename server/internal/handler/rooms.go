package handler

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
	"golang.org/x/crypto/bcrypt"
)

// CreateRoom handles POST /api/v1/rooms
//
//	@Summary		Create a room
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			request	body		vibe.CreateRoomRequest	true	"Room creation payload"
//	@Success		201		{object}	vibe.Room
//	@Failure		400		{object}	vibe.ErrorResponse
//	@Failure		409		{object}	vibe.ErrorResponse
//	@Failure		500		{object}	vibe.ErrorResponse
//	@Router			/api/v1/rooms [post]
func CreateRoom(
	db vibe.RoomCreatorExistenceChecker,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req vibe.CreateRoomRequest
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

		if !req.Validate() {
			handleError(
				w,
				vibe.PublicError{
					Err:        fmt.Errorf("error validating room name"),
					Kind:       vibe.PublicRoomNameInvalid,
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

		settings, err := vibe.DefaultRoomSettings()
		if err != nil {
			handleError(w, fmt.Errorf("error getting default room settings: %w", err), http.StatusInternalServerError, true)
			return
		}

		if req.Settings != nil {
			settings = req.Settings
		}

		if req.Settings != nil && req.Settings.OnlyAdminAddSongs && req.Password == "" {
			handleError(
				w,
				fmt.Errorf("error admin password is required when enabling 'only admin add songs'"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		if req.Settings != nil && req.Settings.Public && req.Password == "" {
			handleError(
				w,
				vibe.PublicError{
					Err: internalerror.ErrMissingAdminPassword{
						Err: fmt.Errorf("error room must have a password to be public"),
					},
					Kind:       vibe.PublicRoomPasswordRequired,
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		room := &vibe.Room{
			ID:                slug,
			Name:              req.Name,
			Mode:              mode,
			HostID:            session.UserID,
			AdminPasswordHash: passwordHash,
			HasPassword:       passwordHash != "",
			Settings:          *settings,
			CreatedAt:         time.Now(),
			ActiveSources:     []string{},
		}

		created, err := db.CreateRoom(ctx, room, req.ReservationToken)
		if err != nil {
			var unavailableError internalerror.ErrRoomNameUnavailable
			if errors.As(err, &unavailableError) {
				handleError(
					w,
					vibe.PublicError{
						Err:        unavailableError,
						Kind:       vibe.PublicRoomNameUnavailable,
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

// ReserveRoomName handles POST /api/v1/rooms/reservations.
//
//	@Summary		Reserve a custom or generated room name
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			request	body		vibe.RoomNameReservationRequest	true	"Room name reservation request"
//	@Success		201		{object}	vibe.RoomNameReservation
//	@Failure		400		{object}	vibe.ErrorResponse
//	@Failure		409		{object}	vibe.ErrorResponse
//	@Failure		500		{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Router			/api/v1/rooms/reservations [post]
func ReserveRoomName(db vibe.RoomNameReserver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req vibe.RoomNameReservationRequest
		err := json.UnmarshalRead(http.MaxBytesReader(w, r.Body, 4096), &req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding room name reservation request: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		if !req.Validate() {
			handleError(w, vibe.PublicError{
				Err:        fmt.Errorf("error validating room name reservation"),
				Kind:       vibe.PublicRoomNameInvalid,
				StatusCode: http.StatusBadRequest,
			}, http.StatusBadRequest, false)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error getting room name reservation session"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		var reservation *vibe.RoomNameReservation
		if req.Name == "" {
			reservation, err = db.ReserveSuggestedRoomName(
				ctx,
				session.UserID,
			)
		} else {
			roomID := helper.Slugify(req.Name)
			if roomID == "" {
				handleError(
					w,
					fmt.Errorf("error invalid room name"),
					http.StatusBadRequest,
					false,
				)
				return
			}

			reservation, err = db.ReserveRoomName(
				ctx,
				roomID,
				session.UserID,
			)
		}
		if err != nil {
			var unavailableError internalerror.ErrRoomNameUnavailable
			if errors.As(err, &unavailableError) {
				handleError(
					w,
					vibe.PublicError{
						Err:        unavailableError,
						Kind:       vibe.PublicRoomNameUnavailable,
						StatusCode: http.StatusConflict,
					},
					http.StatusConflict,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf("error reserving room name: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(reservation)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling room name reservation: %w", err),
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

// SuggestRoomName handles GET /api/v1/rooms/suggestions.
//
//	@Summary		Suggest an available, memorable room name
//	@Tags			rooms
//	@Produce		json
//	@Success		200	{object}	vibe.RoomNameReservation
//	@Failure		500	{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	409	{object}	vibe.ErrorResponse
//	@Router			/api/v1/rooms/suggestions [get]
func SuggestRoomName(db vibe.RoomNameSuggester) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error getting room name suggestion session"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		suggestion, err := db.ReserveSuggestedRoomName(ctx, session.UserID)
		if err != nil {
			var unavailableError internalerror.ErrRoomNameUnavailable
			if errors.As(err, &unavailableError) {
				handleError(
					w,
					vibe.PublicError{
						Err:        unavailableError,
						Kind:       vibe.PublicRoomNameUnavailable,
						StatusCode: http.StatusConflict,
					},
					http.StatusConflict,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf("error suggesting room name: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(suggestion)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling room name suggestion: %w", err),
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

// RoomExists handles HEAD /api/v1/rooms/{id}.
//
//	@Summary		Check whether a room exists
//	@Tags			rooms
//	@Param			id	path	string	true	"Room ID"
//	@Success		200
//	@Failure		404
//	@Failure		500
//	@Router			/api/v1/rooms/{id} [head]
func RoomExists(db vibe.RoomExistenceChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		exists, err := db.RoomExists(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error checking room existence: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// GetRoom handles GET /api/v1/rooms/{id}
//
//	@Summary	Get a room
//	@Tags		rooms
//	@Produce	json
//	@Param		id	path		string	true	"Room ID"
//	@Success	200	{object}	vibe.Room
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/rooms/{id} [get]
func GetRoom(
	db vibe.RoomFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		session, _ := helper.GetSessionFromContext(ctx)

		room, err := db.GetRoom(ctx, roomID, session.UserID)
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

// GetPublicRooms handles GET /api/v1/rooms/public
//
//	@Summary	Get public rooms with active listeners
//	@Tags		rooms
//	@Produce	json
//	@Success	200	{array}		vibe.PublicRoom
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/rooms/public [get]
func GetPublicRooms(db vibe.PublicRoomFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rooms, err := db.GetPublicRooms(ctx)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching public rooms: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(rooms)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling public rooms: %w", err),
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

// UpdateRoomSettings handles PATCH /rooms/{id}/settings
//
//	@Summary	Update room settings
//	@Tags		rooms
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Room ID"
//	@Param		request	body		vibe.UpdateRoomRequest	true	"Room update payload"
//	@Success	200		{object}	vibe.Room
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	404		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Router		/api/v1/rooms/{id}/settings [patch]
func UpdateRoomSettings(
	db vibe.RoomSettingsUpdater,
	notifier vibe.RoomBatchEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.UpdateRoomRequest
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

		room, err := db.GetRoom(ctx, roomID, session.UserID)
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

		if req.Settings != nil && !req.Settings.IsEmpty() {
			room.Settings = *req.Settings
		}
		if req.Mode != "" {
			room.Mode = req.Mode
		}

		if room.Settings.OnlyAdminAddSongs && !room.HasPassword {
			handleError(
				w,
				internalerror.ErrMissingAdminPassword{Err: fmt.Errorf("error room must have a password to enable 'only admin add songs'")},
				http.StatusBadRequest,
				false,
			)
			return
		}

		if room.Settings.Public && !room.HasPassword {
			handleError(
				w,
				vibe.PublicError{
					Err: internalerror.ErrMissingAdminPassword{
						Err: fmt.Errorf("error room must have a password to be public"),
					},
					Kind:       vibe.PublicRoomPasswordRequired,
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		updated, err := db.UpdateRoom(ctx, room)
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

		err = notifier.NotifyRoomUpdates(context.WithoutCancel(ctx), roomID, []vibe.RoomEvent{{
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

		if req.Settings != nil && previousSettings.OnlyAdminAddSongs != updated.Settings.OnlyAdminAddSongs &&
			room.Settings.OnlyAdminAddSongs == updated.Settings.OnlyAdminAddSongs {
			value := "OFF"
			if updated.Settings.OnlyAdminAddSongs {
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

		events := []vibe.RoomEvent{}
		for _, change := range changes {
			message := vibe.RoomMessage{
				ID:        uuid.NewString(),
				UserID:    session.UserID,
				Name:      profile.Name,
				IsAdmin:   updated.IsAdmin,
				Kind:      vibe.MessageKindSettings,
				Text:      change,
				CreatedAt: time.Now().UnixMilli(),
			}

			payload, err := json.Marshal(message)
			if err != nil {
				log.Printf("error marshaling room settings chat activity: %v", err)
				return
			}

			events = append(events, vibe.RoomEvent{Type: vibe.SettingsActivityEvent, Payload: payload})
		}

		err = notifier.NotifyRoomUpdates(context.WithoutCancel(ctx), roomID, events)
		if err != nil {
			log.Printf("error publishing room settings chat activity: %v", err)
		}
	}
}

// CreateSession handles POST /api/v1/rooms/:id/sessions
//
//	@Summary	Create a room session
//	@Tags		rooms
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Room ID"
//	@Param		request	body		vibe.CreateSessionRequest	true	"Session payload"
//	@Success	200		{object}	vibe.SessionResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	403		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Router		/api/v1/rooms/{id}/sessions [post]
func CreateSession(
	db vibe.AdminSessionCreator,
	notifier vibe.RoomEventNotifier,
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

		room, err := db.GetRoom(ctx, roomID, session.UserID)
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
			neutralRoom, err := db.GetRoom(ctx, roomID, "")
			if err != nil {
				log.Printf("CreateSession: failed to fetch neutral room for notification: %v", err)
			}
			if err == nil {
				body, err := json.Marshal(neutralRoom)
				if err != nil {
					log.Printf("CreateSession: failed to marshal room for notification: %v", err)
				}
				if err == nil {
					err = notifier.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
						Type:    vibe.SettingsUpdate,
						Payload: body,
						UserID:  session.UserID, // Include the user who set the password
						Origin:  session.EventOrigin,
					})
					if err != nil {
						log.Printf("CreateSession: failed to notify room password setup: %v", err)
					}
				}
			}
		}

		resp := vibe.SessionResponse{
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
			Activity:  true,
			Text:      "a password to the room",
			CreatedAt: time.Now().UnixMilli(),
		}

		payload, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling room password chat activity: %v", err)
			return
		}

		err = notifier.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
			Type:    vibe.MessageEvent,
			Payload: payload,
		})
		if err != nil {
			log.Printf("error publishing room password chat activity: %v", err)
		}
	}
}

// DeleteRoomAdminSession handles DELETE /api/v1/rooms/:id/sessions.
//
//	@Summary	Log out of a room admin session
//	@Tags		rooms
//	@Produce	json
//	@Param		id	path		string	true	"Room ID"
//	@Success	200	{object}	vibe.SessionResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/rooms/{id}/sessions [delete]
func DeleteRoomAdminSession(db vibe.RoomAdminSessionDeleter) http.HandlerFunc {
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
				fmt.Errorf("error clearing room admin in DeleteRoomAdminSession: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		room, err := db.GetRoom(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room in DeleteRoomAdminSession: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		resp := vibe.SessionResponse{
			UserID:    session.UserID,
			SessionID: session.UserID,
			IsAdmin:   false,
			Room:      room,
		}

		body, err := json.Marshal(resp)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling response in DeleteRoomAdminSession: %w", err),
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
