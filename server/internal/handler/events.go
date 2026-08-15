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

			err = ips.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
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
		err = writeRoomEvent(w, flusher, vibe.Connected, data)
		if err != nil {
			log.Printf("error writing connection event in RoomEvents: %v", err)
			return
		}

		// Room event delivery is ephemeral, so every connection receives a full
		// authoritative snapshot instead of relying on unavailable event replay.
		err = writeRoomEventSnapshot(ctx, w, flusher, snapshot, roomID, userID)
		if err != nil {
			log.Printf("error writing room snapshot in RoomEvents: %v", err)
			return
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
					continue
				}

				// payloadData is already []byte (JSON), so we can just use it.
				// However, if we print it as %s, it works.
				err = writeRoomEvent(w, flusher, event.Type, event.Payload)
				if err != nil {
					log.Printf("error writing event in RoomEvents: %v", err)
					return
				}
			}
		}
	}
}

func writeRoomEventSnapshot(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	snapshot vibe.RoomEventSnapshotFetcher,
	roomID string,
	userID string,
) error {
	room, err := snapshot.GetRoom(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("error fetching room in writeRoomEventSnapshot: %w", err)
	}
	if room == nil || room.IsEmpty() {
		return fmt.Errorf("error fetching room in writeRoomEventSnapshot: room is empty")
	}
	roomData, err := json.Marshal(room)
	if err != nil {
		return fmt.Errorf("error marshaling room in writeRoomEventSnapshot: %w", err)
	}

	songs, err := snapshot.GetSongs(ctx, roomID)
	if err != nil {
		return fmt.Errorf("error fetching songs in writeRoomEventSnapshot: %w", err)
	}
	if songs == nil {
		songs = []vibe.Song{}
	}
	songsData, err := json.Marshal(songs)
	if err != nil {
		return fmt.Errorf("error marshaling songs in writeRoomEventSnapshot: %w", err)
	}

	state, err := snapshot.GetPlaybackState(ctx, roomID)
	if err != nil {
		return fmt.Errorf("error fetching playback in writeRoomEventSnapshot: %w", err)
	}
	if state == nil {
		return fmt.Errorf("error fetching playback in writeRoomEventSnapshot: playback is nil")
	}
	if state.IsPlaying && state.UpdatedAt.Before(time.Now()) {
		state.PositionMs += int(time.Since(state.UpdatedAt).Milliseconds())
		state.UpdatedAt = time.Now()
	}
	state.ServerTimeMs = int(time.Now().UnixMilli())
	playbackData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("error marshaling playback in writeRoomEventSnapshot: %w", err)
	}

	err = writeRoomEvent(w, flusher, vibe.SettingsUpdate, roomData)
	if err != nil {
		return fmt.Errorf("error writing room in writeRoomEventSnapshot: %w", err)
	}
	err = writeRoomEvent(w, flusher, vibe.QueueReordered, songsData)
	if err != nil {
		return fmt.Errorf("error writing songs in writeRoomEventSnapshot: %w", err)
	}
	err = writeRoomEvent(w, flusher, vibe.PlaybackUpdate, playbackData)
	if err != nil {
		return fmt.Errorf("error writing playback in writeRoomEventSnapshot: %w", err)
	}

	return nil
}

func writeRoomEvent(
	w http.ResponseWriter,
	flusher http.Flusher,
	eventType string,
	payload []byte,
) error {
	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, payload)
	if err != nil {
		return fmt.Errorf("error writing SSE event in writeRoomEvent: %w", err)
	}
	flusher.Flush()

	return nil
}
