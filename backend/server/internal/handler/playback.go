package handler

import (
	"context"
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
	db vibe.RoomGetterPlaybackUpdater,
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

		switch req.Action {
		case vibe.RoomActionPlay, vibe.RoomActionPause, vibe.RoomActionSeek:
			// Allowed actions
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

		state, err := db.UpdatePlayback(ctx, roomID, userID, req.Action, req.PositionMs)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("action %s failed: %w", req.Action, err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		state.ServerTimeMs = int(time.Now().UnixMilli())

		// Broadcast if in Host mode
		if room.Mode == vibe.RoomModeHost {
			// Notify room
			statePayload, err := json.Marshal(state)
			if err != nil {
				log.Printf("failed to marshal playback state payload: %v", err)
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
				log.Printf("failed to notify room: %v", err)
			}
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

// ReviewRoomPlayback handles playback monitoring for server-mode rooms
type ReviewRoomPlayback struct {
	DB  vibe.ExpiredPlaybackProcessor
	IPS vibe.RoomBatchEventNotifier
}

// Handle checks for rooms that need to auto-advance
func (h *ReviewRoomPlayback) Handle(ctx context.Context, data []byte) error {
	state, err := h.DB.ProcessNextExpiredPlayback(ctx)
	if err != nil {
		return fmt.Errorf("error processing next expired playback: %w", err)
	}

	statePayload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("error marshaling playback state payload: %w", err)
	}

	err = h.IPS.NotifyRoomUpdates(ctx, state.RoomID, []vibe.RoomEvent{
		{
			Type:    vibe.PlaybackUpdate,
			Payload: statePayload,
		},
	})
	if err != nil {
		return fmt.Errorf("error notifying room %s update: %w", state.RoomID, err)
	}

	return nil
}

// ReviewHostHealth handles host health checks
type ReviewHostHealth struct {
	DB  vibe.AbandonnedHostProcessor
	IPS vibe.RoomEventNotifier
}

// Handle checks for rooms that need a new host
func (h *ReviewHostHealth) Handle(ctx context.Context, data []byte) error {
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
