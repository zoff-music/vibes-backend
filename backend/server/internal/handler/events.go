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

// RoomEvents handles GET /api/v1/rooms/:id/events (SSE)
func RoomEvents(
	ips vibe.Subscriber,
	db vibe.PlaybackStateUpdater,
	participants vibe.ParticipantStorage,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		// Get UserID from session/context. SSE usually auth via cookie or query param?
		// Vibes uses cookie session middleware?
		// Check action.go: middleware.SessionKey.
		userID := ""
		session, ok := helper.GetSessionFromContext(ctx)
		if ok {
			userID = session.UserID
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		topicName := fmt.Sprintf("room:%s", roomID)
		container, err := ips.Subscribe(topicName)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to subscribe to room events: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		defer container.Subscription.Destroy()

		flusher, ok := w.(http.Flusher)
		if !ok {
			handleError(
				w,
				fmt.Errorf("streaming not supported"),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		// Send initial heartbeat or connection established event
		fmt.Fprintf(w, "event: connected\ndata: {\"time\": %d}\n\n", time.Now().UnixMilli())
		flusher.Flush()

		// Update presence on connect and cleanup on disconnect
		if userID != "" {
			_ = participants.UpdateParticipant(ctx, roomID, userID, true)
			defer func() {
				_ = participants.RemoveParticipant(context.Background(), roomID, userID)
			}()
		}

		// Send initial playback state
		state, err := db.GetPlaybackState(ctx, roomID)
		if err != nil {
			log.Printf("failed to fetch initial playback state: %v", err)
		}

		if state != nil {
			// Project position to current time if playing
			if state.IsPlaying && state.UpdatedAt.Before(time.Now()) {
				state.PositionMs += time.Since(state.UpdatedAt).Milliseconds()
				state.UpdatedAt = time.Now()
			}
			state.ServerTimeMs = time.Now().UnixMilli()

			data, _ := json.Marshal(state)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", vibe.PlaybackUpdate, data)
			flusher.Flush()
		}

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		messages := container.Subscription.Listen()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Keep-alive heartbeat AND update participant status
				if userID != "" {
					_ = participants.UpdateParticipant(ctx, roomID, userID, true)
				}

				fmt.Fprintf(w, ": heartbeat\n\n")
				flusher.Flush()
			case data, ok := <-messages:
				if !ok {
					return
				}

				var event vibe.RoomEvent
				err := json.Unmarshal(data, &event)
				if err != nil {
					log.Printf("failed to unmarshal room event: %v", err)
					continue
				}

				// payloadData is already []byte (JSON), so we can just use it.
				// However, if we print it as %s, it works.
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Payload)
				flusher.Flush()
			}
		}
	}
}
