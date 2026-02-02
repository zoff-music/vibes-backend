package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/internalerror"
	"github.com/zoff-music/vibes/server/internal/helper"
	"github.com/zoff-music/vibes/vibe"
	"golang.org/x/crypto/bcrypt"
)

// CreateRoom handles POST /api/v1/rooms
func CreateRoom(
	db vibe.RoomCreatorAdminRoomLister,
	ips vibe.AdminEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req vibe.CreateRoomRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest, true)
			return
		}

		if req.Name == "" {
			handleError(
				w,
				fmt.Errorf("room name is required"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		// Get session from context
		session, _ := helper.GetSessionFromContext(ctx)

		// Check if room already exists
		existingRoom, err := db.GetRoomByName(ctx, req.Name, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to check existing room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if !existingRoom.IsEmpty() {
			// Room exists, return it with 200 OK
			body, err := json.Marshal(existingRoom)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("marshal response: %w", err),
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

		var passwordHash string
		if req.Password != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("failed to hash password: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			passwordHash = string(hash)
		}

		// Generate slug from name
		slug := helper.Slugify(req.Name)
		if slug == "" {
			// Fallback if slugify results in empty string (e.g. "   ")
			handleError(
				w,
				fmt.Errorf("invalid room name"),
				http.StatusBadRequest,
				false,
			)
			return
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
				fmt.Errorf("admin password is required when enabling 'only admin add songs'"),
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
				fmt.Errorf("failed to create room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(created)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("marshal response: %w", err),
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

// GetRoom handles GET /api/v1/rooms/{id}
func GetRoom(
	db vibe.RoomFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		// Get session from context (guaranteed by middleware)
		session, _ := helper.GetSessionFromContext(ctx)

		room, err := db.GetRoom(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to fetch room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		body, err := json.Marshal(room)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("marshal response: %w", err),
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

		// Only update settings if they are provided and not empty
		if req.Settings != nil && !req.Settings.IsEmpty() {
			room.Settings = *req.Settings
		}
		if req.Mode != "" {
			room.Mode = req.Mode
		}

		// Validation: If OnlyAdminAddSongs is enabled, room must have a password
		if room.Settings.OnlyAdminAddSongs && !room.HasPassword {
			handleError(
				w,
				internalerror.ErrMissingAdminPassword{Err: fmt.Errorf("room must have a password to enable 'only admin add songs'")},
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

		// Broadcast room update to all connected clients
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
				fmt.Errorf("invalid request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		// Get session from context (guaranteed by middleware)
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("unauthorized: missing session"),
				http.StatusUnauthorized,
				true,
			)
			return
		}

		if req.Password == "" {
			handleError(
				w,
				fmt.Errorf("password required"),
				http.StatusForbidden,
				false,
			)
			return
		}

		// Authenticate and elevate in the DB
		authResult, err := db.AuthenticateAdmin(ctx, roomID, session.UserID, req.Password)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("authentication failed: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if !authResult.IsAdmin {
			handleError(
				w,
				fmt.Errorf("incorrect password"),
				http.StatusForbidden,
				false,
			)
			return
		}

		isFirstTimeSetup := authResult.IsFirstTimeSetup

		// Fetch updated room to return
		room, err := db.GetRoom(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to fetch room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		// If this was a first-time password setup, notify all clients
		if isFirstTimeSetup {
			// Fetch room without user-specific data for broadcast
			neutralRoom, err := db.GetRoom(ctx, roomID, "")
			if err != nil {
				log.Printf("CreateSession: failed to fetch neutral room for notification: %v", err)
			} else {
				body, err := json.Marshal(neutralRoom)
				if err != nil {
					log.Printf("CreateSession: failed to marshal room for notification: %v", err)
				} else {
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
