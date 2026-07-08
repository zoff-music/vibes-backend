package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// SkipSong handles POST /rooms/{id}/skips
func SkipSong(
	db vibe.RoomSkipper,
	ips vibe.RoomBatchEventAdminNotifier,
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

		result, err := db.SkipSong(ctx, roomID, userID, isAdmin)
		if err != nil {
			// Check if it's a host mode restriction error
			var errHostMode internalerror.ErrHostModeSkipOnly
			if errors.As(err, &errHostMode) {
				handleError(
					w,
					fmt.Errorf("only hosts can skip in host mode"),
					http.StatusForbidden,
					false,
				)
				return
			}

			// Check if skipping is disabled
			var errDisabled internalerror.ErrSkipDisabled
			if errors.As(err, &errDisabled) {
				handleError(
					w,
					fmt.Errorf("skipping is disabled in this room"),
					http.StatusForbidden,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf("skip failed: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if result.Playback != nil {
			result.Playback.ServerTimeMs = int(time.Now().UnixMilli())
		}

		if !result.Skipped {
			payload := vibe.SkipVoteUpdate{
				UserID:        userID,
				CurrentVotes:  result.CurrentVotes,
				RequiredVotes: result.RequiredVotes,
			}
			if result.Playback != nil && result.Playback.CurrentSong != nil {
				payload.SongID = result.Playback.CurrentSong.ID
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

			err = ips.NotifyRoomUpdates(context.WithoutCancel(ctx), roomID, []vibe.RoomEvent{
				{
					Type:    vibe.SkipVoteEvent,
					Payload: votePayload,
					UserID:  userID,
				},
			})
			if err != nil {
				log.Printf("failed to notify skip vote: %v", err)
			}
		}

		if result.Skipped {
			songs, err := db.GetSongs(ctx, roomID)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error fetching songs: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			songsPayload, err := json.Marshal(songs)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling songs payload: %w", err),
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

			err = ips.NotifyRoomUpdates(context.WithoutCancel(ctx), roomID, []vibe.RoomEvent{
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
		}

		body, err := json.Marshal(result)
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
