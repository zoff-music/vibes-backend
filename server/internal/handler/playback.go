package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zoff-music/vibes-backend/monitoring/opentracing"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetPlaybackState handles GET /rooms/{id}/states
func GetPlaybackState(
	db vibe.PlaybackFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		state, err := db.GetPlaybackState(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to fetch playback state: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		state.ServerTimeMs = int(time.Now().UnixMilli())

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

// UpdatePlaybackState handles PUT /rooms/{id}/states
func UpdatePlaybackState(
	db vibe.RoomGetterPlaybackUpdater,
	ips vibe.RoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["id"]

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

		switch req.Action {
		case vibe.RoomActionPlay, vibe.RoomActionPause, vibe.RoomActionSeek:
			// Allowed
		default:
			handleError(
				w,
				fmt.Errorf("invalid action for state update: %s", req.Action),
				http.StatusBadRequest,
				false,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
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

		room, err := db.GetRoom(ctx, roomID, userID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to fetch room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		log.Printf("Room %s: User %s attempting %s", roomID, userID, req.Action)

		var state *vibe.PlaybackState // adjust if your type name differs

		// ----------------------------
		// Server mode: do NOT broadcast.
		// ----------------------------
		if room.Mode == vibe.RoomModeServer {
			state, err = db.GetPlaybackState(ctx, roomID)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("failed to get playback state: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			// Cases where we actually update DB in server mode:
			// 1) play when no current song (kick off first song from queue)
			// 2) seek (allowed)
			shouldPersist := req.Action == vibe.RoomActionSeek ||
				(req.Action == vibe.RoomActionPlay && state.CurrentSong == nil)

			if shouldPersist {
				state, err = db.UpdatePlayback(ctx, roomID, userID, req.Action, req.PositionMs)
				if err != nil {
					handleError(
						w,
						fmt.Errorf("action %s failed: %w", req.Action, err),
						http.StatusInternalServerError,
						true,
					)
					return
				}

				// fallthrough to response writing
			}

			switch req.Action {
			case vibe.RoomActionPause:
				state.IsPlaying = false
			case vibe.RoomActionPlay:
				state.IsPlaying = true
			}

			state.ServerTimeMs = int(time.Now().UnixMilli())

			body, err := json.Marshal(state)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshalling response in update playback state handler: %w", err),
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

		// ----------------------------
		// Non-server modes: update DB.
		// ----------------------------
		state, err = db.UpdatePlayback(ctx, roomID, userID, req.Action, req.PositionMs)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error updating playback: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		state.ServerTimeMs = int(time.Now().UnixMilli())

		// Host mode broadcast (best effort; only marshal once)
		if room.Mode == vibe.RoomModeHost {
			statePayload, err := json.Marshal(state)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshalling playback state payload in update playback state handler: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			err = ips.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
				Type:    vibe.PlaybackUpdate,
				Payload: statePayload,
			})
			if err != nil {
				log.Printf("error notifying room %s of playback update: %v", roomID, err)
			}
		}

		body, err := json.Marshal(state)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshalling response in update playback state handler: %w", err),
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

// ReviewRoomPlayback handles playback monitoring for server-mode rooms
type ReviewRoomPlayback struct {
	DB  vibe.ExpiredPlaybackSongFetcher
	IPS vibe.RoomBatchEventNotifier
}

// Handle checks for rooms that need to auto-advance
func (h *ReviewRoomPlayback) Handle(ctx context.Context, data []byte) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Handle")
	defer span.Finish()

	state, err := h.DB.ProcessNextExpiredPlayback(ctx)
	if err != nil {
		return fmt.Errorf("error processing next expired playback: %w", err)
	}

	statePayload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("error marshaling playback state payload: %w", err)
	}

	songs, err := h.DB.GetSongs(ctx, state.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching songs for room %s: %w", state.RoomID, err)
	}

	songsPayload, err := json.Marshal(songs)
	if err != nil {
		return fmt.Errorf("error marshaling songs payload: %w", err)
	}

	err = h.IPS.NotifyRoomUpdates(ctx, state.RoomID, []vibe.RoomEvent{
		{
			Type:    vibe.PlaybackUpdate,
			Payload: statePayload,
		},
		{
			Type:    vibe.QueueReordered,
			Payload: songsPayload,
		},
	})
	if err != nil {
		return fmt.Errorf("error notifying room %s update: %w", state.RoomID, err)
	}

	return nil
}

// ReviewHostHealth handles host health checks
type ReviewHostHealth struct {
	DB  vibe.AbandonedHostProcessor
	IPS vibe.RoomEventNotifier
}

// Handle checks for rooms that need a new host
func (h *ReviewHostHealth) Handle(ctx context.Context, data []byte) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Handle")
	defer span.Finish()

	info, err := h.DB.ProcessNextAbandonedHost(ctx)
	if err != nil {
		return fmt.Errorf("error processing next abandoned host: %w", err)
	}

	payloadMap := map[string]string{"userId": info.NewHostID, "message": "You are now the host"}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return fmt.Errorf("error marshaling new host payload: %w", err)
	}

	err = h.IPS.NotifyRoomUpdate(ctx, info.RoomID, vibe.RoomEvent{
		Type:    vibe.NewHost,
		Payload: payloadBytes,
	})
	if err != nil {
		return fmt.Errorf("error notifying room %s host update: %w", info.RoomID, err)
	}

	return nil
}
