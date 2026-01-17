package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/zoff-music/vibes/vibe"
)

// RoomEvents handles GET /api/v1/rooms/:id/events (SSE)
// RoomEvents handles GET /api/v1/rooms/:id/events (SSE)
func RoomEvents(
	sb vibe.Subscriber,
	pf vibe.PlaybackFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		topicName := fmt.Sprintf("room:%s", roomID)
		container, err := sb.Subscribe(topicName)
		if err != nil {
			handleError(w, fmt.Errorf("failed to subscribe to room events: %w", err), http.StatusInternalServerError, true)
			return
		}
		defer container.Subscription.Destroy()

		flusher, ok := w.(http.Flusher)
		if !ok {
			handleError(w, fmt.Errorf("streaming not supported"), http.StatusInternalServerError, true)
			return
		}

		// Send initial heartbeat or connection established event
		fmt.Fprintf(w, "event: connected\ndata: {\"time\": %d}\n\n", time.Now().UnixMilli())
		flusher.Flush()

		// Send initial playback state
		state, err := pf.GetPlaybackState(ctx, roomID)
		if err == nil {
			// Project position to current time if playing
			if state.IsPlaying && state.UpdatedAt.Before(time.Now()) {
				state.PositionMs += time.Since(state.UpdatedAt).Milliseconds()
				state.UpdatedAt = time.Now()
			}
			state.ServerTimeMs = time.Now().UnixMilli()

			data, _ := json.Marshal(state)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", vibe.EventTypePlaybackUpdate, data)
			flusher.Flush()
		}

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		messages := container.Subscription.Listen()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Keep-alive heartbeat
				fmt.Fprintf(w, ": heartbeat\n\n")
				flusher.Flush()
			case data, ok := <-messages:
				if !ok {
					return
				}

				var event vibe.RoomEvent
				err := json.Unmarshal(data, &event)
				if err != nil {
					log.Errorf("failed to unmarshal room event: %v", err)
					continue
				}

				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
				flusher.Flush()
			}
		}
	}
}
