package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/zoff-music/vibes/server/internal/middleware"
	"github.com/zoff-music/vibes/vibe"
)

// RoomAction handles POST /api/v1/rooms/:id
func RoomAction(
	db vibe.RoomActioner,
	ips vibe.RoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.RoomActionRequest
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

		session, ok := ctx.Value(middleware.SessionKey).(middleware.SessionPayload)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}
		userID := session.UserID

		var state *vibe.PlaybackState

		switch req.Action {
		case vibe.RoomActionPlay, vibe.RoomActionPause, vibe.RoomActionSeek:
			state, err = db.UpdatePlayback(ctx, roomID, req.Action, req.PositionMs)
		case vibe.RoomActionSkip:
			state, err = db.SkipTrack(ctx, roomID)
		case vibe.RoomActionVote:
			state, err = db.VoteToSkip(ctx, roomID, userID)
		default:
			handleError(
				w,
				fmt.Errorf("invalid action: %s", req.Action),
				http.StatusBadRequest,
				false,
			)
			return
		}

		if err != nil {
			handleError(
				w,
				fmt.Errorf("action %s failed: %w", req.Action, err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		// Update server time for response
		state.ServerTimeMs = time.Now().UnixMilli()

		err = ips.NotifyRoom(ctx, roomID, &vibe.RoomEvent{
			Type:    vibe.EventTypePlaybackUpdate,
			Payload: state,
		})
		if err != nil {
			log.Errorf("failed to notify room: %v", err)
		}

		body, err := json.Marshal(state)
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
