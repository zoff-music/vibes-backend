package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/server/internal/helper"
	"github.com/zoff-music/vibes/vibe"
	"golang.org/x/crypto/bcrypt"
)

// CreateRoom handles POST /api/v1/rooms
func CreateRoom(
	db vibe.RoomCreator,
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

		// Check if room already exists
		existingRoom, err := db.GetRoomByName(ctx, req.Name)
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

		room := &vibe.Room{
			ID:                slug,
			Name:              req.Name,
			Mode:              mode,
			AdminPasswordHash: passwordHash,
			HasPassword:       passwordHash != "",
			Settings:          vibe.DefaultRoomSettings(),
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

// GetRoom handles GET /api/v1/rooms/:id
func GetRoom(
	db vibe.RoomFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		room, err := db.GetRoom(ctx, roomID)
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

		room, err := db.GetRoom(ctx, roomID)
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

		if req.Settings != nil {
			room.Settings = *req.Settings
		}
		if req.Mode != "" {
			room.Mode = req.Mode
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
		// Note: 'body' is already the marshaled 'updated' from line 237
		err = ips.NotifyRoomUpdate(ctx, roomID, vibe.RoomEvent{
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
	db vibe.SessionCreator,
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

		// Check if user already has a valid session - don't create a new one
		session, hasSession := helper.GetSessionFromContext(ctx)
		if hasSession && session.UserID != "" {
			// User already has a session, just return existing info
			room, err := db.GetRoom(ctx, roomID)
			if err == nil && !room.IsEmpty() {
				resp := vibe.SessionResponse{
					UserID:   session.UserID,
					Nickname: nil,   // Frontend will use stored value
					IsAdmin:  false, // Will be updated if they re-auth
					Room:     room,
				}

				body, _ := json.Marshal(resp)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(body)
				return
			}
		}

		room, err := db.GetRoom(ctx, roomID)
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

		isAdmin := false
		if req.Password != "" && room.AdminPasswordHash != "" {
			err = bcrypt.CompareHashAndPassword([]byte(room.AdminPasswordHash), []byte(req.Password))
			isAdmin = err == nil
		}

		user := &vibe.User{
			// ID is generated by DB
			RoomID:     roomID,
			Nickname:   &req.Nickname,
			IsAdmin:    isAdmin,
			JoinedAt:   time.Now(),
			LastSeenAt: time.Now(),
		}

		createdUser, err := db.CreateUser(ctx, user)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to create session: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		resp := vibe.SessionResponse{
			UserID:   createdUser.ID,
			Nickname: createdUser.Nickname,
			IsAdmin:  createdUser.IsAdmin,
			Room:     room,
		}

		sessionPayload := helper.SessionPayload{
			UserID: createdUser.ID,
		}
		sessionJSON, _ := json.Marshal(sessionPayload)
		sessionEncoded := base64.StdEncoding.EncodeToString(sessionJSON)

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionEncoded,
			Path:     "/",
			HttpOnly: true,
			// Secure:   true, // TODO: Enable in production
			SameSite: http.SameSiteLaxMode,
		})

		body, err := json.Marshal(resp)
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
