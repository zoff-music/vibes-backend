package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// AdminEventsV3 streams a fresh snapshot followed by individual room changes.
//
// @Summary Subscribe to room administration changes
// @Description Each connection receives admin_rooms_update with the complete snapshot, then admin_room_update with one summary or admin_room_removed with an id. Reconnects replace the snapshot. Heartbeats are comments, not room polling.
// @Tags admin
// @Produce text/event-stream
// @Success 200 {string} string "SSE frames"
// @Router /api/v3/admin/events [get]
func AdminEventsV3(
	subscriber vibe.Subscriber,
	db vibe.AdminRoomSummaryListerUserFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		admin, ok := helper.GetAdminUserFromContext(ctx)
		if !ok || admin == nil {
			handleError(
				w,
				fmt.Errorf("error missing admin session"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

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

		// Capture the stream position before reading the snapshot. Changes during
		// that read are consumed afterwards and resolved against current DB state.
		container, err := subscriber.Subscribe(ctx, vibe.AdminRoomChangesTopic)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error subscribing to admin room changes: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		defer container.Subscription.Destroy()

		rooms, err := db.ListAdminRoomsV2(ctx)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching admin room snapshot: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		payload, err := json.Marshal(rooms)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin room snapshot: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		currentAdmin, err := db.GetAdminUser(ctx, admin.ID)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("error revalidating admin session in AdminEventsV3: %v", err)
			return
		}
		if currentAdmin.IsEmpty() || currentAdmin.SessionVersion != admin.SessionVersion {
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", vibe.AdminRoomsUpdate, payload)
		if err != nil {
			log.Printf("error writing admin room snapshot: %v", err)
			return
		}
		flusher.Flush()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		// Presence expires after the last room heartbeat. These are one-shot
		// checks for snapshot listeners and disconnect events, not a polling job.
		departures := make(map[string]time.Time)
		for _, room := range rooms {
			if room.UserCount > 0 {
				departures[room.ID] = time.Now().Add(16 * time.Second)
			}
		}
		expiry := time.NewTimer(16 * time.Second)
		defer expiry.Stop()

		messages := container.Subscription.Listen()
		for {
			var data []byte
			var open bool
			heartbeat := false
			var change vibe.AdminRoomChange

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				heartbeat = true
			case <-expiry.C:
				for roomID, deadline := range departures {
					if !deadline.After(time.Now()) {
						change.ID = roomID
						delete(departures, roomID)
						break
					}
				}
			case data, open = <-messages:
				if !open {
					return
				}

				err = json.Unmarshal(data, &change)
				if err != nil || change.ID == "" {
					log.Printf("error decoding admin room change: %v", err)
					return
				}
			}

			if !change.RecheckAt.IsZero() {
				departures[change.ID] = change.RecheckAt
			}

			var nextDeparture time.Time
			for _, deadline := range departures {
				if nextDeparture.IsZero() || deadline.Before(nextDeparture) {
					nextDeparture = deadline
				}
			}
			if !nextDeparture.IsZero() {
				expiry.Reset(time.Until(nextDeparture))
			}

			currentAdmin, err = db.GetAdminUser(ctx, admin.ID)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				log.Printf("error revalidating admin session in AdminEventsV3: %v", err)
				return
			}
			if currentAdmin.IsEmpty() || currentAdmin.SessionVersion != admin.SessionVersion {
				return
			}

			if heartbeat {
				_, err = fmt.Fprint(w, ": heartbeat\n\n")
				if err != nil {
					log.Printf("error writing admin heartbeat: %v", err)
					return
				}
				flusher.Flush()
				continue
			}

			if change.ID == "" || !change.RecheckAt.IsZero() {
				continue
			}

			room, err := db.GetAdminRoomSummary(ctx, change.ID)
			if err != nil {
				log.Printf("error fetching changed admin room: %v", err)
				return
			}

			eventType := vibe.AdminRoomUpdate
			payload, err = json.Marshal(room)
			if room.IsEmpty() {
				eventType = vibe.AdminRoomRemoved
				payload, err = json.Marshal(change)
			}
			if err != nil {
				log.Printf("error marshaling admin room change: %v", err)
				return
			}

			currentAdmin, err = db.GetAdminUser(ctx, admin.ID)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				log.Printf("error revalidating admin session in AdminEventsV3: %v", err)
				return
			}
			if currentAdmin.IsEmpty() || currentAdmin.SessionVersion != admin.SessionVersion {
				return
			}

			_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, payload)
			if err != nil {
				log.Printf("error writing admin room change: %v", err)
				return
			}
			flusher.Flush()
		}
	}
}
