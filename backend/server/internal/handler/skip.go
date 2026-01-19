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

// SkipSong handles POST /rooms/{id}/skips
func SkipSong(
	db vibe.RoomSkipper,
	ips vibe.RoomBatchEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized, false)
			return
		}
		userID := session.UserID

		// Check if user is admin - currently session doesn't have IsAdmin
		// Logic moved to DB, passing false for now unless we can verify admin status
		isAdmin := false

		state, err := db.SkipSong(ctx, roomID, userID, isAdmin)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("skip failed: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		songs, err := db.GetSongs(ctx, roomID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to fetch songs: %w", err), http.StatusInternalServerError, true)
			return
		}

		state.ServerTimeMs = time.Now().UnixMilli()

		// Notify room updates
		// We have to marshal payloads manually now
		songsPayload, err := json.Marshal(songs)
		if err != nil {
			log.Printf("failed to marshal songs payload: %v", err)
			songsPayload = []byte("{}")
		}

		statePayload, err := json.Marshal(state)
		if err != nil {
			log.Printf("failed to marshal playback state payload: %v", err)
			statePayload = []byte("{}")
		}

		err = ips.NotifyRoomUpdates(ctx, roomID, []vibe.RoomEvent{
			{
				Type:    vibe.QueueReordered,
				Payload: songsPayload,
			},
			{
				Type:    vibe.PlaybackUpdate,
				Payload: statePayload,
			},
		})
		if err != nil {
			log.Printf("failed to notify room updates: %v", err)
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
