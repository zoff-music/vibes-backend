package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// RoomEvents handles GET /api/v1/rooms/:id/events (SSE)
//
//	@Summary	Subscribe to room events
//	@Tags		events
//	@Produce	text/event-stream
//	@Param		id	path	string	true	"Room ID"
//	@Success	200	{string}	string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/events [get]
func RoomEvents(
	ips vibe.SubscriberPublisher,
	db vibe.ParticipantGetterUpdaterPlaybackGetter,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		// Get UserID from session/context.
		userID := ""
		isCastReceiver := false
		castOwnerID := ""
		session, ok := helper.GetSessionFromContext(ctx)
		if ok {
			userID = session.UserID
			isCastReceiver = session.AuthType == "cast"
			if isCastReceiver {
				castOwnerID = session.UserID
			}
		}

		if isCastReceiver {
			// Use a per-connection ID to avoid colliding with the real user session.
			// The underlying user identity is castOwnerID.
			if castOwnerID != "" {
				userID = fmt.Sprintf("cast:%s:%s", roomID, castOwnerID)
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
				fmt.Errorf("error subscribing to room events: %w", err),
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
				fmt.Errorf("error streaming not supported"),
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
			_ = db.UpdateParticipant(ctx, roomID, userID, !isCastReceiver, isCastReceiver, castOwnerID)
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
					_ = db.UpdateParticipant(ctx, roomID, userID, !isCastReceiver, isCastReceiver, castOwnerID)
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
				// For cast connections, compare against the underlying user id (castOwnerID).
				filterID := userID
				if isCastReceiver && castOwnerID != "" {
					filterID = castOwnerID
				}
				if event.UserID != "" && event.UserID == filterID {
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
