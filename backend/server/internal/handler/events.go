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
	ips vibe.SubscriberPublisher,
	db vibe.ParticipantGetterUpdaterPlaybackGetter,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		casterID := r.URL.Query().Get("casterId")
		isCastReceiver := r.URL.Query().Get("castReceiver") == "1" ||
			r.Header.Get("X-Cast-Receiver") == "1"

		// Get UserID from session/context. SSE usually auth via cookie or query param?
		// Vibes uses cookie session middleware?
		// Check action.go: middleware.SessionKey.
		userID := ""
		session, ok := helper.GetSessionFromContext(ctx)
		if ok {
			userID = session.UserID
		}

		if isCastReceiver {
			if casterID == "" {
				casterID = r.Header.Get("X-Cast-Caster-Id")
			}
			if casterID == "" && userID != "" {
				casterID = userID
			}
			if casterID != "" {
				userID = fmt.Sprintf("cast:%s:%s", roomID, casterID)
			}
		}

		notifyUsers := func(ctx context.Context) {
			counts, err := db.GetActiveListenerCounts(ctx, roomID, 15*time.Second)
			if err != nil {
				log.Printf("failed to fetch active participants: %v", err)
				return
			}

			count := counts.ActiveListeners
			if counts.ActiveListeners == 0 && counts.ActiveCastReceivers > 0 {
				count = 1
			}
			payload, err := json.Marshal(count)
			if err != nil {
				log.Printf("failed to marshal active participants count: %v", err)
				return
			}

			err = ips.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
				Type:    vibe.UsersUpdate,
				Payload: payload,
			})
			if err != nil {
				log.Printf("failed to notify room update: %v", err)
			}

			// Admin room updates are handled by the app event job to avoid
			// amplifying updates on every listener heartbeat/connect.
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
			_ = db.UpdateParticipant(ctx, roomID, userID, !isCastReceiver, isCastReceiver, casterID)
			notifyUsers(ctx)
			defer notifyUsers(context.Background())
		}

		// Send initial playback state
		state, err := db.GetPlaybackState(ctx, roomID)
		if err != nil {
			log.Printf("failed to fetch initial playback state: %v", err)
		}

		if state != nil {
			// Project position to current time if playing
			if state.IsPlaying && state.UpdatedAt.Before(time.Now()) {
				state.PositionMs += int(time.Since(state.UpdatedAt).Milliseconds())
				state.UpdatedAt = time.Now()
			}
			state.ServerTimeMs = int(time.Now().UnixMilli())

			data, err := json.Marshal(state)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshalling playback state: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

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
					_ = db.UpdateParticipant(ctx, roomID, userID, !isCastReceiver, isCastReceiver, casterID)
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

				// Filter out events triggered by the same user (e.g., password setup)
				if event.UserID != "" && event.UserID == userID {
					continue // Skip sending this event to the user who triggered it
				}

				// payloadData is already []byte (JSON), so we can just use it.
				// However, if we print it as %s, it works.
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Payload)
				flusher.Flush()
			}
		}
	}
}
