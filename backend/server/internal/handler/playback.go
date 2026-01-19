package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/server/internal/helper"
	"github.com/zoff-music/vibes/vibe"
)

// UpdatePlaybackState handles PUT /rooms/{id}/states
func UpdatePlaybackState(
	db vibe.PlaybackStateUpdater,
	ips vibe.RoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.RoomActionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest, true)
			return
		}

		switch req.Action {
		case vibe.RoomActionPlay, vibe.RoomActionPause, vibe.RoomActionSeek:
			// Allowed actions
		default:
			handleError(w, fmt.Errorf("invalid action for state update: %s", req.Action), http.StatusBadRequest, false)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized, false)
			return
		}
		userID := session.UserID

		room, err := db.GetRoom(ctx, roomID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to fetch room: %w", err), http.StatusInternalServerError, true)
			return
		}

		log.Printf("Room %s: User %s attempting %s", roomID, userID, req.Action)

		state, err := db.UpdatePlayback(ctx, roomID, userID, req.Action, req.PositionMs)
		if err != nil {
			handleError(w, fmt.Errorf("action %s failed: %w", req.Action, err), http.StatusInternalServerError, true)
			return
		}

		state.ServerTimeMs = time.Now().UnixMilli()

		// Broadcast if in Host mode
		if room.Mode == vibe.RoomModeHost {
			err = ips.NotifyRoomUpdate(ctx, roomID, vibe.RoomEvent{
				Type:    vibe.PlaybackUpdate,
				Payload: state,
			})
			if err != nil {
				log.Printf("failed to notify room: %v", err)
			}
		}

		body, err := json.Marshal(state)
		if err != nil {
			handleError(w, fmt.Errorf("marshal response: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
