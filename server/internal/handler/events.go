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
	events vibe.RoomEventReplayNotifier,
	participants vibe.RoomEventParticipantFetcherUpdater,
	snapshot vibe.RoomEventSnapshotFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		// Get UserID from session/context.
		userID := ""
		isCastReceiver := false
		isRemoteController := false
		castOwnerID := ""
		isNewSession := false
		session, ok := helper.GetSessionFromContext(ctx)
		if ok {
			userID = session.UserID
			isCastReceiver = session.AuthType == "cast"
			isRemoteController = session.AuthType == "remote"
			isNewSession = session.IsNew
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

		lastUsersCount := -1
		notifyUsers := func(ctx context.Context) {
			counts, err := participants.GetActiveListenerCounts(ctx, roomID, 15*time.Second)
			if err != nil {
				log.Printf("failed to fetch active participants: %v", err)
				return
			}

			count := counts.ActiveListeners
			if counts.ActiveListeners == 0 && counts.ActiveCastReceivers > 0 {
				count = 1
			}
			if count == lastUsersCount {
				return
			}

			payload, err := json.Marshal(count)
			if err != nil {
				log.Printf("failed to marshal active participants count: %v", err)
				return
			}

			err = events.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
				Type:    vibe.UsersUpdate,
				Payload: payload,
			})
			if err != nil {
				log.Printf("failed to notify room update: %v", err)
				return
			}
			lastUsersCount = count

			// Admin room updates are handled by the app event job to avoid
			// amplifying updates on every listener heartbeat/connect.
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		topicName := fmt.Sprintf("room:%s", roomID)
		replay, err := events.PrepareReplay(ctx, topicName, lastEventIDFromRequest(r))
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error preparing room event replay: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		container, err := events.SubscribeFrom(ctx, topicName, replay.AfterID)
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

		connection := vibe.RoomEventConnection{Time: time.Now().UnixMilli()}
		data, err := json.Marshal(connection)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling connection event in RoomEvents: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		err = writeRoomEvent(w, flusher, vibe.Connected, data, "")
		if err != nil {
			log.Printf("error writing connection event in RoomEvents: %v", err)
			return
		}

		if replay.RequiresSnapshot {
			room, err := snapshot.GetRoom(ctx, roomID, userID)
			if err != nil {
				log.Printf("error fetching room snapshot in RoomEvents: %v", err)
				return
			}
			if room == nil || room.IsEmpty() {
				log.Printf("error fetching room snapshot in RoomEvents: room is empty")
				return
			}
			roomData, err := json.Marshal(room)
			if err != nil {
				log.Printf("error marshaling room snapshot in RoomEvents: %v", err)
				return
			}

			songs, err := snapshot.GetSongs(ctx, roomID)
			if err != nil {
				log.Printf("error fetching songs snapshot in RoomEvents: %v", err)
				return
			}
			if songs == nil {
				songs = []vibe.Song{}
			}
			songsData, err := json.Marshal(songs)
			if err != nil {
				log.Printf("error marshaling songs snapshot in RoomEvents: %v", err)
				return
			}

			state, err := snapshot.GetPlaybackState(ctx, roomID)
			if err != nil {
				log.Printf("error fetching playback snapshot in RoomEvents: %v", err)
				return
			}
			if state == nil {
				log.Printf("error fetching playback snapshot in RoomEvents: playback is nil")
				return
			}
			if state.IsPlaying && state.UpdatedAt.Before(time.Now()) {
				state.PositionMs += int(time.Since(state.UpdatedAt).Milliseconds())
				state.UpdatedAt = time.Now()
			}
			state.ServerTimeMs = int(time.Now().UnixMilli())
			playbackData, err := json.Marshal(state)
			if err != nil {
				log.Printf("error marshaling playback snapshot in RoomEvents: %v", err)
				return
			}

			err = writeRoomEvent(w, flusher, vibe.SettingsUpdate, roomData, replay.AfterID)
			if err != nil {
				log.Printf("error writing room snapshot in RoomEvents: %v", err)
				return
			}
			err = writeRoomEvent(w, flusher, vibe.QueueReordered, songsData, replay.AfterID)
			if err != nil {
				log.Printf("error writing songs snapshot in RoomEvents: %v", err)
				return
			}
			err = writeRoomEvent(w, flusher, vibe.PlaybackUpdate, playbackData, replay.AfterID)
			if err != nil {
				log.Printf("error writing playback snapshot in RoomEvents: %v", err)
				return
			}
		}

		participantRegistered := false

		// Reconnects reuse the participant row from the signed session cookie.
		// New cookie-less connections must survive one heartbeat before they
		// become listeners so short probes cannot manufacture participant rows.
		if userID != "" && !isRemoteController {
			if !isNewSession {
				err = participants.UpdateParticipant(ctx, roomID, userID, !isCastReceiver, isCastReceiver, castOwnerID)
				if err != nil {
					log.Printf("failed to update participant on connect: %v", err)
				}
				if err == nil {
					participantRegistered = true
					notifyUsers(ctx)
				}
			}

			defer func() {
				if participantRegistered {
					notifyUsers(context.Background())
				}
			}()
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
				if userID != "" && !isRemoteController {
					err = participants.UpdateParticipant(ctx, roomID, userID, !isCastReceiver, isCastReceiver, castOwnerID)
					if err != nil {
						log.Printf("failed to update participant on heartbeat: %v", err)
					}
					if err == nil {
						if !participantRegistered {
							participantRegistered = true
						}
						notifyUsers(ctx)
					}
				}

				_, err = fmt.Fprint(w, ": heartbeat\n\n")
				if err != nil {
					log.Printf("error writing heartbeat in RoomEvents: %v", err)
					return
				}
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
				if event.Type == vibe.UsersUpdate {
					err = json.Unmarshal(event.Payload, &lastUsersCount)
					if err != nil {
						log.Printf("failed to unmarshal users update: %v", err)
						continue
					}
				}

				filterID := userID
				if isCastReceiver && castOwnerID != "" {
					filterID = castOwnerID
				}
				if event.UserID != "" &&
					event.UserID == filterID &&
					event.Origin != vibe.RoomEventOriginRemote {
					err = writeRoomEventCursor(w, flusher, event.ID)
					if err != nil {
						log.Printf("error writing filtered event cursor in RoomEvents: %v", err)
						return
					}
					continue
				}

				// payloadData is already []byte (JSON), so we can just use it.
				// However, if we print it as %s, it works.
				err = writeRoomEvent(w, flusher, event.Type, event.Payload, event.ID)
				if err != nil {
					log.Printf("error writing event in RoomEvents: %v", err)
					return
				}
			}
		}
	}
}

func writeRoomEvent(
	w http.ResponseWriter,
	flusher http.Flusher,
	eventType string,
	payload []byte,
	eventID string,
) error {
	if eventID != "" {
		_, err := fmt.Fprintf(w, "id: %s\n", eventID)
		if err != nil {
			return fmt.Errorf("error writing SSE event id in writeRoomEvent: %w", err)
		}
	}

	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, payload)
	if err != nil {
		return fmt.Errorf("error writing SSE event in writeRoomEvent: %w", err)
	}
	if eventID != "" {
		err = writeRoomEventCursor(w, flusher, eventID)
		if err != nil {
			return fmt.Errorf("error writing cursor in writeRoomEvent: %w", err)
		}
	}
	flusher.Flush()

	return nil
}

func writeRoomEventCursor(
	w http.ResponseWriter,
	flusher http.Flusher,
	eventID string,
) error {
	if eventID == "" {
		return nil
	}

	cursorData, err := json.Marshal(vibe.RoomEventCursor{ID: eventID})
	if err != nil {
		return fmt.Errorf("error marshaling SSE cursor in writeRoomEventCursor: %w", err)
	}
	_, err = fmt.Fprintf(
		w,
		"id: %s\nevent: %s\ndata: %s\n\n",
		eventID,
		vibe.EventCursor,
		cursorData,
	)
	if err != nil {
		return fmt.Errorf("error writing SSE cursor in writeRoomEventCursor: %w", err)
	}
	flusher.Flush()

	return nil
}

func lastEventIDFromRequest(r *http.Request) string {
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		return lastEventID
	}

	lastEventID = r.URL.Query().Get("lastEventId")

	return lastEventID
}
