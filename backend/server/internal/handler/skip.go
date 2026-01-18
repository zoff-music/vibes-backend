package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/zoff-music/vibes/server/internal/helper"
	"github.com/zoff-music/vibes/vibe"
)

// SkipSong handles POST /rooms/{id}/skips
func SkipSong(
	db vibe.RoomSkipper,
	ips vibe.RoomEventNotifier,
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

		// Fetch room to check permissions and mode
		room, err := db.GetRoom(ctx, roomID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to fetch room: %w", err), http.StatusInternalServerError, true)
			return
		}

		log.Infof("Room %s: User %s attempting skip/vote", roomID, userID)

		var state *vibe.PlaybackState

		isHost := room.HostID == userID
		// If we need to check session.IsAdmin, we might need a way to pass it or check user role.
		// Current middleware.SessionPayload does not contain IsAdmin.
		// However, if the room has a password, and the user joined with it, they might be admin.
		// But in stateless session, we rely on token. If token doesn't have it, we check DB or assume HostID match.
		// For now, assuming HostID match is sufficient for acting as host.

		shouldForce := isHost
		if !room.Settings.DemocraticSkip {
			// If not democratic, only force skip exists?
			// Or maybe anyone can skip if settings allow?
			// RoomSettings has SkipAllowed.
			shouldForce = true
		}

		skipFunc := db.VoteToSkip
		if shouldForce {
			skipFunc = db.SkipTrack
		}

		state, err = skipFunc(ctx, roomID, userID)
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

		err = ips.NotifyRoomUpdates(ctx, roomID, []*vibe.RoomEvent{
			{
				Type:    vibe.EventTypeQueueReordered,
				Payload: songs,
			},
			{
				Type:    vibe.EventTypePlaybackUpdate,
				Payload: state,
			},
		})
		if err != nil {
			log.Errorf("failed to notify room updates: %v", err)
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
