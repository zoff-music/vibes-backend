package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/vibe"
	"golang.org/x/crypto/bcrypt"
)

type CreateRoomRequest struct {
	Name     string `json:"name"`
	Password string `json:"password,omitempty"`
}

type UpdateRoomRequest struct {
	Settings vibe.RoomSettings `json:"settings"`
}

type CreateSessionRequest struct {
	Nickname string `json:"nickname,omitempty"`
	Password string `json:"password,omitempty"`
}

// CreateRoom handles POST /api/v1/rooms
func CreateRoom(
	rc vibe.RoomCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req CreateRoomRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest, true)
			return
		}

		if req.Name == "" {
			handleError(w, fmt.Errorf("room name is required"), http.StatusBadRequest, false)
			return
		}

		var passwordHash string
		if req.Password != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				handleError(w, fmt.Errorf("failed to hash password: %w", err), http.StatusInternalServerError, true)
				return
			}
			passwordHash = string(hash)
		}

		room := &vibe.Room{
			Name:              req.Name,
			AdminPasswordHash: passwordHash,
			HasPassword:       passwordHash != "",
			Settings:          vibe.DefaultRoomSettings(),
			CreatedAt:         time.Now(),
		}

		created, err := rc.CreateRoom(ctx, room)
		if err != nil {
			handleError(w, fmt.Errorf("failed to create room: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created)
	}
}

// GetRoom handles GET /api/v1/rooms/:id
func GetRoom(
	rf vibe.RoomFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		room, err := rf.GetRoom(ctx, roomID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to fetch room: %w", err), http.StatusInternalServerError, true)
			return
		}

		if room.IsEmpty() {
			handleError(w, fmt.Errorf("room not found"), http.StatusNotFound, false)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(room)
	}
}

// UpdateRoom handles PATCH /api/v1/rooms/:id
func UpdateRoom(
	rf vibe.RoomFetcher,
	ru vibe.RoomUpdater,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		// TODO: Implement admin authorization check
		// For now, we assume the user is authorized if they are an admin in their session

		var req UpdateRoomRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest, true)
			return
		}

		room, err := rf.GetRoom(ctx, roomID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to fetch room: %w", err), http.StatusInternalServerError, true)
			return
		}

		if room.IsEmpty() {
			handleError(w, fmt.Errorf("room not found"), http.StatusNotFound, false)
			return
		}

		room.Settings = req.Settings
		updated, err := ru.UpdateRoom(ctx, room)
		if err != nil {
			handleError(w, fmt.Errorf("failed to update room: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(updated)
	}
}

// CreateSession handles POST /api/v1/rooms/:id/sessions
func CreateSession(
	rf vibe.RoomFetcher,
	um vibe.UserManager,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req CreateSessionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest, true)
			return
		}

		room, err := rf.GetRoom(ctx, roomID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to fetch room: %w", err), http.StatusInternalServerError, true)
			return
		}

		if room.IsEmpty() {
			handleError(w, fmt.Errorf("room not found"), http.StatusNotFound, false)
			return
		}

		isAdmin := false
		if req.Password != "" && room.AdminPasswordHash != "" {
			err = bcrypt.CompareHashAndPassword([]byte(room.AdminPasswordHash), []byte(req.Password))
			if err == nil {
				isAdmin = true
			}
		}

		user := &vibe.User{
			RoomID:     roomID,
			Nickname:   &req.Nickname,
			IsAdmin:    isAdmin,
			JoinedAt:   time.Now(),
			LastSeenAt: time.Now(),
		}

		createdUser, err := um.CreateUser(ctx, user)
		if err != nil {
			handleError(w, fmt.Errorf("failed to create session: %w", err), http.StatusInternalServerError, true)
			return
		}

		resp := vibe.SessionResponse{
			UserID:   createdUser.ID,
			Nickname: createdUser.Nickname,
			IsAdmin:  createdUser.IsAdmin,
			Room:     room,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
