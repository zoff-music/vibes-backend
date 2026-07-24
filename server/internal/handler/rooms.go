package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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
//	@Failure		400		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/rooms [post]
func CreateRoom(
	db vibe.RoomCreatorAdminRoomLister,
	ips vibe.AdminEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req vibe.CreateRoomRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		if req.Name == "" {
			handleError(
				w,
				fmt.Errorf("error room name is required"),
				http.StatusBadRequest,
				false,
			)
			return
		}

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

		settings := vibe.DefaultRoomSettings()
		if req.Settings != nil {
			settings = *req.Settings
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

		room := &vibe.Room{
			ID:                slug,
			Name:              req.Name,
			Mode:              mode,
			HostID:            session.UserID,
			AdminPasswordHash: passwordHash,
			HasPassword:       passwordHash != "",
			Settings:          settings,
			CreatedAt:         time.Now(),
			ActiveSources:     []string{},
		}

		created, err := db.CreateRoom(ctx, room)
		if err != nil {
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

// SuggestRoomName handles GET /api/v1/rooms/suggestions.
//
//	@Summary		Suggest an available, memorable room name
//	@Tags			rooms
//	@Produce		json
//	@Success		200	{object}	vibe.RoomNameSuggestion
//	@Failure		500	{object}	map[string]string
//	@Router			/api/v1/rooms/suggestions [get]
func SuggestRoomName(db vibe.RoomNameSuggester) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		candidates, err := vibe.GenerateRoomNameCandidates()
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error generating room name candidates: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		suggestion, err := db.SuggestRoomName(ctx, candidates)
		if err != nil {
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
//	@Failure		500		{object}	map[string]string
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
//	@Failure	404	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
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

// UpdateRoomSettings handles PATCH /rooms/{id}/settings
//
//	@Summary	Update room settings
//	@Tags		rooms
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Room ID"
//	@Param		request	body		vibe.UpdateRoomRequest	true	"Room update payload"
//	@Success	200		{object}	vibe.Room
//	@Failure	400		{object}	map[string]string
//	@Failure	404		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/settings [patch]
func UpdateRoomSettings(
	db vibe.RoomSettingsUpdater,
	ips vibe.RoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.UpdateRoomRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error invalid request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		session, _ := helper.GetSessionFromContext(ctx)
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

		err = ips.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
			Type:    vibe.SettingsUpdate,
			Payload: body,
		})
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
//	@Failure	401		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/sessions [post]
func CreateSession(
	db vibe.SessionCreatorGetter,
	ips vibe.RoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.CreateSessionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
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
					err = ips.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
						Type:    vibe.SettingsUpdate,
						Payload: body,
						UserID:  session.UserID, // Include the user who set the password
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

		body, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
