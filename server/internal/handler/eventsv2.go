package handler

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// RoomEventsV2 handles GET /api/v2/rooms/:id/events (SSE).
//
//	@Summary	Subscribe to compact room events
//	@Tags		rooms
//	@Produce	text/event-stream
//	@Param		id	path	string	true	"Room ID"
//	@Success	200	{string}	string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v2/rooms/{id}/events [get]
func RoomEventsV2(
	events vibe.RoomEventReplayNotifier,
	state vibe.RoomEventStateFetcherUpdater,
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
			counts, err := state.GetActiveListenerCounts(ctx, roomID, 15*time.Second)
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
				fmt.Errorf("error marshaling connection event in RoomEventsV2: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		err = writeRoomEvent(w, flusher, vibe.Connected, data, "")
		if err != nil {
			log.Printf("error writing connection event in RoomEventsV2: %v", err)
			return
		}

		if replay.RequiresSnapshot {
			room, err := state.GetRoom(ctx, roomID, userID)
			if err != nil {
				log.Printf("error fetching room snapshot in RoomEventsV2: %v", err)
				return
			}
			if room == nil || room.IsEmpty() {
				log.Printf("error fetching room snapshot in RoomEventsV2: room is empty")
				return
			}
			roomData, err := json.Marshal(room)
			if err != nil {
				log.Printf("error marshaling room snapshot in RoomEventsV2: %v", err)
				return
			}

			songs, err := state.GetSongs(ctx, roomID)
			if err != nil {
				log.Printf("error fetching songs snapshot in RoomEventsV2: %v", err)
				return
			}
			if songs == nil {
				songs = []vibe.Song{}
			}
			songsData, err := json.Marshal(songs)
			if err != nil {
				log.Printf("error marshaling songs snapshot in RoomEventsV2: %v", err)
				return
			}

			playback, err := state.GetPlaybackState(ctx, roomID)
			if err != nil {
				log.Printf("error fetching playback snapshot in RoomEventsV2: %v", err)
				return
			}
			if playback == nil {
				log.Printf("error fetching playback snapshot in RoomEventsV2: playback is nil")
				return
			}
			if playback.IsPlaying && playback.UpdatedAt.Before(time.Now()) {
				playback.PositionMs += int(time.Since(playback.UpdatedAt).Milliseconds())
				playback.UpdatedAt = time.Now()
			}
			playback.ServerTimeMs = int(time.Now().UnixMilli())
			playbackData, err := json.Marshal(playback)
			if err != nil {
				log.Printf("error marshaling playback snapshot in RoomEventsV2: %v", err)
				return
			}

			err = writeRoomEvent(w, flusher, vibe.SettingsUpdate, roomData, replay.AfterID)
			if err != nil {
				log.Printf("error writing room snapshot in RoomEventsV2: %v", err)
				return
			}
			err = writeRoomEvent(w, flusher, vibe.QueueSnapshot, songsData, replay.AfterID)
			if err != nil {
				log.Printf("error writing songs snapshot in RoomEventsV2: %v", err)
				return
			}
			err = writeRoomEvent(w, flusher, vibe.PlaybackUpdate, playbackData, replay.AfterID)
			if err != nil {
				log.Printf("error writing playback snapshot in RoomEventsV2: %v", err)
				return
			}
		}

		participantRegistered := false

		// Reconnects reuse the participant row from the signed session cookie.
		// New cookie-less connections must survive one heartbeat before they
		// become listeners so short probes cannot manufacture participant rows.
		if userID != "" && !isRemoteController {
			if !isNewSession {
				err = state.UpdateParticipant(ctx, roomID, userID, !isCastReceiver, isCastReceiver, castOwnerID)
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

		presenceTicker := time.NewTicker(5 * time.Second)
		defer presenceTicker.Stop()
		keepAliveTicker := time.NewTicker(15 * time.Second)
		defer keepAliveTicker.Stop()

		messages := container.Subscription.Listen()

		for {
			select {
			case <-ctx.Done():
				return
			case <-presenceTicker.C:
				// Update participant status independently from the wire keep-alive.
				if userID != "" && !isRemoteController {
					err = state.UpdateParticipant(ctx, roomID, userID, !isCastReceiver, isCastReceiver, castOwnerID)
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
			case <-keepAliveTicker.C:
				_, err = fmt.Fprint(w, ": heartbeat\n\n")
				if err != nil {
					log.Printf("error writing heartbeat in RoomEventsV2: %v", err)
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
						log.Printf("error writing filtered event cursor in RoomEventsV2: %v", err)
						return
					}
					continue
				}

				eventType := event.Type
				payload := event.Payload
				if event.V2 != nil {
					eventType = event.V2.Type
					payload = event.V2.Payload
				}
				if event.Type == vibe.QueueReordered && event.V2 == nil {
					err = writeRoomEventCursor(w, flusher, event.ID)
					if err != nil {
						log.Printf("error writing v2 filtered event cursor in RoomEventsV2: %v", err)
						return
					}
					continue
				}

				err = writeRoomEvent(w, flusher, eventType, payload, event.ID)
				if err != nil {
					log.Printf("error writing event in RoomEventsV2: %v", err)
					return
				}
			}
		}
	}
}

func songAddedV2(song vibe.Song) (*vibe.RoomEventV2Payload, error) {
	payload, err := json.Marshal(song)
	if err != nil {
		return nil, fmt.Errorf("error marshaling v2 added song: %w", err)
	}

	return &vibe.RoomEventV2Payload{Type: vibe.SongAdded, Payload: payload}, nil
}

func songRemovedV2(songID string) (*vibe.RoomEventV2Payload, error) {
	payload, err := json.Marshal(vibe.SongIDUpdate{ID: songID})
	if err != nil {
		return nil, fmt.Errorf("error marshaling v2 removed song: %w", err)
	}

	return &vibe.RoomEventV2Payload{Type: vibe.SongRemoved, Payload: payload}, nil
}

func songPositionV2(songs []vibe.Song, songID string) (*vibe.RoomEventV2Payload, error) {
	for position, song := range songs {
		if song.ID != songID {
			continue
		}

		payload, err := json.Marshal(vibe.SongPositionUpdate{
			Song:     song,
			Position: position,
		})
		if err != nil {
			return nil, fmt.Errorf("error marshaling v2 positioned song: %w", err)
		}

		return &vibe.RoomEventV2Payload{Type: vibe.SongUpdated, Payload: payload}, nil
	}

	payload, err := songRemovedV2(songID)
	if err != nil {
		return nil, fmt.Errorf("error creating v2 removed song fallback: %w", err)
	}

	return payload, nil
}
