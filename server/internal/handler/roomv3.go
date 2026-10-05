package handler

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// RoomEventsV3 handles GET /api/v3/rooms/:id/events (SSE).
//
//	@Summary	Subscribe to compact room events
//
// @Description Returns an SSE stream, not a single JSON response. Frames contain an optional `id`, an `event` name, and a JSON `data` payload, followed by a blank line.
// @Description `connected` carries {"time": 1700000000000}; `settings_update` carries a RoomV2; `playlist_items_snapshot` carries a PlaylistItem array; `playback_update` carries a PlaybackStateV2. Compact queue updates use `playlist_item_added`, `playlist_item_updated`, and `playlist_item_removed`.
// @Description Reconnect with Last-Event-ID or lastEventId. See the [event contract](https://github.com/zoff-music/vibes-backend/blob/main/docs/flows/sessions.md) for event payloads and replay.
//
//	@Tags		rooms
//	@Produce	text/event-stream
//
// @Param lastEventId query string false "Last received stream cursor; used when Last-Event-ID is absent"
// @Param Last-Event-ID header string false "Last received stream cursor"
//
//	@Param		id	path	string	true	"Room ID"
//	@Success	200	{string}	string "SSE frames with event-specific JSON data; see the stream description"
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v3/rooms/{id}/events [get]
func RoomEventsV3(
	events vibe.RoomEventV3ReplayNotifier,
	state vibe.RoomEventV3StateFetcherUpdater,
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
			if err != nil && ctx.Err() != nil {
				return
			}
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

			err = events.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
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
		lastEventID := r.Header.Get("Last-Event-ID")
		if lastEventID == "" {
			lastEventID = r.URL.Query().Get("lastEventId")
		}
		replay, err := events.PrepareReplay(ctx, topicName, lastEventID)
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
				fmt.Errorf("error marshaling connection event in RoomEventsV3: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", vibe.Connected, data)
		if err != nil {
			log.Printf("error writing connection event in RoomEventsV3: %v", err)
			return
		}
		flusher.Flush()

		if replay.RequiresSnapshot {
			room, err := state.GetRoomV2(ctx, roomID, userID)
			if err != nil {
				log.Printf("error fetching room snapshot in RoomEventsV3: %v", err)
				return
			}
			if room == nil || room.IsEmpty() {
				log.Printf("error fetching room snapshot in RoomEventsV3: room is empty")
				return
			}
			roomData, err := json.Marshal(room)
			if err != nil {
				log.Printf("error marshaling room snapshot in RoomEventsV3: %v", err)
				return
			}

			playlistItems, err := state.GetPlaylistItems(ctx, roomID)
			if err != nil {
				log.Printf("error fetching playlist items snapshot in RoomEventsV3: %v", err)
				return
			}
			if playlistItems == nil {
				playlistItems = []vibe.PlaylistItem{}
			}
			playlistItemsData, err := json.Marshal(playlistItems)
			if err != nil {
				log.Printf("error marshaling playlist items snapshot in RoomEventsV3: %v", err)
				return
			}

			playback, err := state.GetPlaybackStateV2(ctx, roomID)
			if err != nil {
				log.Printf("error fetching playback snapshot in RoomEventsV3: %v", err)
				return
			}
			if playback == nil {
				log.Printf("error fetching playback snapshot in RoomEventsV3: playback is nil")
				return
			}
			if playback.IsPlaying && playback.UpdatedAt.Before(time.Now()) {
				playback.PositionMs += int(time.Since(playback.UpdatedAt).Milliseconds())
				playback.UpdatedAt = time.Now()
			}
			playback.ServerTimeMs = int(time.Now().UnixMilli())
			playbackData, err := json.Marshal(playback)
			if err != nil {
				log.Printf("error marshaling playback snapshot in RoomEventsV3: %v", err)
				return
			}

			if replay.AfterID != "" {
				_, err = fmt.Fprintf(w, "id: %s\n", replay.AfterID)
				if err != nil {
					log.Printf("error writing room snapshot event id in RoomEventsV3: %v", err)
					return
				}
			}
			_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", vibe.SettingsUpdate, roomData)
			if err != nil {
				log.Printf("error writing room snapshot in RoomEventsV3: %v", err)
				return
			}
			if replay.AfterID != "" {
				cursorData, marshalErr := json.Marshal(vibe.RoomEventCursor{ID: replay.AfterID})
				if marshalErr != nil {
					log.Printf("error marshaling room snapshot cursor in RoomEventsV3: %v", marshalErr)
					return
				}
				_, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", replay.AfterID, vibe.EventCursor, cursorData)
				if err != nil {
					log.Printf("error writing room snapshot cursor in RoomEventsV3: %v", err)
					return
				}
			}
			flusher.Flush()

			if replay.AfterID != "" {
				_, err = fmt.Fprintf(w, "id: %s\n", replay.AfterID)
				if err != nil {
					log.Printf("error writing playlist items snapshot event id in RoomEventsV3: %v", err)
					return
				}
			}
			_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", vibe.PlaylistItemsSnapshot, playlistItemsData)
			if err != nil {
				log.Printf("error writing playlist items snapshot in RoomEventsV3: %v", err)
				return
			}
			if replay.AfterID != "" {
				cursorData, marshalErr := json.Marshal(vibe.RoomEventCursor{ID: replay.AfterID})
				if marshalErr != nil {
					log.Printf("error marshaling playlist items snapshot cursor in RoomEventsV3: %v", marshalErr)
					return
				}
				_, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", replay.AfterID, vibe.EventCursor, cursorData)
				if err != nil {
					log.Printf("error writing playlist items snapshot cursor in RoomEventsV3: %v", err)
					return
				}
			}
			flusher.Flush()

			if replay.AfterID != "" {
				_, err = fmt.Fprintf(w, "id: %s\n", replay.AfterID)
				if err != nil {
					log.Printf("error writing playback snapshot event id in RoomEventsV3: %v", err)
					return
				}
			}
			_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", vibe.PlaybackUpdate, playbackData)
			if err != nil {
				log.Printf("error writing playback snapshot in RoomEventsV3: %v", err)
				return
			}
			if replay.AfterID != "" {
				cursorData, marshalErr := json.Marshal(vibe.RoomEventCursor{ID: replay.AfterID})
				if marshalErr != nil {
					log.Printf("error marshaling playback snapshot cursor in RoomEventsV3: %v", marshalErr)
					return
				}
				_, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", replay.AfterID, vibe.EventCursor, cursorData)
				if err != nil {
					log.Printf("error writing playback snapshot cursor in RoomEventsV3: %v", err)
					return
				}
			}
			flusher.Flush()
		}

		participantRegistered := false

		// Reconnects reuse the participant row from the signed session cookie.
		// New cookie-less connections must survive one heartbeat before they
		// become listeners so short probes cannot manufacture participant rows.
		if userID != "" && !isRemoteController {
			if !isNewSession {
				err = state.UpdateParticipant(ctx, roomID, userID, !isCastReceiver, isCastReceiver, castOwnerID)
				if err != nil && ctx.Err() != nil {
					return
				}
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
					if err != nil && ctx.Err() != nil {
						return
					}
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
					log.Printf("error writing heartbeat in RoomEventsV3: %v", err)
					return
				}
				flusher.Flush()
			case data, ok := <-messages:
				if !ok {
					return
				}

				event, err := vibe.ParseRoomEvent(data)
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
					if event.ID != "" {
						cursorData, marshalErr := json.Marshal(vibe.RoomEventCursor{ID: event.ID})
						if marshalErr != nil {
							log.Printf("error marshaling filtered event cursor in RoomEventsV3: %v", marshalErr)
							return
						}
						_, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, vibe.EventCursor, cursorData)
						if err != nil {
							log.Printf("error writing filtered event cursor in RoomEventsV3: %v", err)
							return
						}
						flusher.Flush()
					}
					continue
				}

				converted, err := event.ToRoomEventV3Payload()
				if err != nil {
					log.Printf("error converting room event in RoomEventsV3: %v", err)
					return
				}

				eventType := converted.Type
				payload := converted.Payload

				if event.ID != "" {
					_, err = fmt.Fprintf(w, "id: %s\n", event.ID)
					if err != nil {
						log.Printf("error writing event id in RoomEventsV3: %v", err)
						return
					}
				}
				_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, payload)
				if err != nil {
					log.Printf("error writing event in RoomEventsV3: %v", err)
					return
				}
				if event.ID != "" {
					cursorData, marshalErr := json.Marshal(vibe.RoomEventCursor{ID: event.ID})
					if marshalErr != nil {
						log.Printf("error marshaling event cursor in RoomEventsV3: %v", marshalErr)
						return
					}
					_, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, vibe.EventCursor, cursorData)
					if err != nil {
						log.Printf("error writing event cursor in RoomEventsV3: %v", err)
						return
					}
				}
				flusher.Flush()
			}
		}
	}
}

// GetPublicRoomsV3 handles GET /api/v3/rooms/public.
//
//	@Summary		Browse public rooms
//	@Description	Returns public rooms ordered by listener count, playlist item count, then ID, all descending. Ranges are zero-based and inclusive. Only rooms with protected admin controls are listed.
//	@Tags			rooms
//	@Produce		json
//	@Param			q		query		string	false	"Case-insensitive room name search (literal substring, up to 100 characters)"
//	@Param			live	query		boolean	false	"Only rooms with active listeners" default(false)
//	@Param			from	query		int		false	"First row, zero-based" minimum(0) default(0)
//	@Param			to		query		int		false	"Last row, inclusive. Defaults to from + 9; at most 100 rooms per page" minimum(0)
//	@Success		200		{object}	vibe.PublicRoomResultV3
//	@Failure		400		{object}	vibe.ErrorResponse
//	@Failure		500		{object}	vibe.ErrorResponse
//	@Router			/api/v3/rooms/public [get]
func GetPublicRoomsV3(db vibe.PublicRoomsV3Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if !utf8.ValidString(query) || strings.ContainsRune(query, '\x00') {
			handleError(w, fmt.Errorf("error invalid room search text"), http.StatusBadRequest, false)
			return
		}

		if utf8.RuneCountInString(query) > publicRoomMaximumQueryLength {
			handleError(w, fmt.Errorf("error room search is too long"), http.StatusBadRequest, false)
			return
		}

		live := r.URL.Query().Get("live")
		if live != "" && live != "true" && live != "false" {
			handleError(w, fmt.Errorf("error invalid live room filter"), http.StatusBadRequest, false)
			return
		}

		var from int64
		var err error
		fromValue := r.URL.Query().Get("from")
		if fromValue != "" {
			from, err = strconv.ParseInt(fromValue, 10, 32)
			if err != nil {
				handleError(w, fmt.Errorf("error parsing first room row: %w", err), http.StatusBadRequest, false)
				return
			}

			if from < 0 {
				handleError(w, fmt.Errorf("error invalid first room row"), http.StatusBadRequest, false)
				return
			}
		}

		to := from + publicRoomPageSize - 1
		toValue := r.URL.Query().Get("to")
		if toValue != "" {
			to, err = strconv.ParseInt(toValue, 10, 32)
			if err != nil {
				handleError(w, fmt.Errorf("error parsing last room row: %w", err), http.StatusBadRequest, false)
				return
			}

			if to < from {
				handleError(w, fmt.Errorf("error invalid last room row"), http.StatusBadRequest, false)
				return
			}
		}

		if to-from+1 > publicRoomMaximumPageSize {
			handleError(w, fmt.Errorf("error room page is too large"), http.StatusBadRequest, false)
			return
		}

		result, err := db.SearchPublicRoomsV3(ctx, vibe.PublicRoomSearch{
			Query: query,
			Live:  live == "true",
			From:  int(from),
			To:    int(to),
		})
		if err != nil {
			handleError(w, fmt.Errorf("error searching public rooms: %w", err), http.StatusInternalServerError, true)
			return
		}

		body, err := json.Marshal(result)
		if err != nil {
			handleError(w, fmt.Errorf("error marshaling public rooms: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
