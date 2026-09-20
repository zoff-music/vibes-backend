package handler

import (
	"encoding/json/v2"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// CreateMessages accepts a room chat message.
// @Summary Send a room message
// @Tags messages
// @Accept json
// @Produce json
// @Param id path string true "Room ID"
// @Param request body vibe.CreateMessageRequest true "Message"
// @Success 201 {object} vibe.RoomMessage
// @Failure 400,401,404,500 {object} vibe.ErrorResponse
// @Router /api/v1/rooms/{id}/messages [post]
func CreateMessages(db vibe.MessageAuthorFetcherUsageCreator, events vibe.RoomEventNotifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["id"]
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" || session.AuthType == "cast" {
			handleError(w, fmt.Errorf("error sending message: unauthorized"), http.StatusUnauthorized, false)
			return
		}

		var request vibe.CreateMessageRequest
		err := json.UnmarshalRead(http.MaxBytesReader(w, r.Body, 4096), &request)
		if err != nil {
			handleError(w, fmt.Errorf("error decoding chat message: %w", err), http.StatusBadRequest, false)
			return
		}

		if !request.Validate() {
			handleError(w, client.ErrorCodeWrapper{
				Err: fmt.Errorf("error validating chat message"),
				ResponseBody: client.ErrorCodeResponseBody{
					Namespace: "vibes-backend",
					Error:     "chat_message_invalid",
					Message:   fmt.Sprintf("Use between 1 and %d characters for your message.", vibe.MessageMaxLength),
					Propagate: true,
				},
				StatusCode: http.StatusBadRequest,
			}, http.StatusBadRequest, false)
			return
		}

		room, err := db.GetRoom(ctx, roomID, session.UserID)
		if err != nil {
			handleError(w, fmt.Errorf("error fetching message room: %w", err), http.StatusInternalServerError, true)
			return
		}

		if room.IsEmpty() {
			handleError(w, fmt.Errorf("error sending message: room not found"), http.StatusNotFound, false)
			return
		}

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			handleError(w, fmt.Errorf("error fetching message author: %w", err), http.StatusInternalServerError, true)
			return
		}

		message := vibe.RoomMessage{
			ID:        uuid.NewString(),
			UserID:    session.UserID,
			Name:      profile.Name,
			IsAdmin:   room.IsAdmin,
			Kind:      "chat",
			Text:      strings.TrimSpace(request.Text),
			CreatedAt: time.Now().UnixMilli(),
		}

		payload, err := json.Marshal(message)
		if err != nil {
			handleError(w, fmt.Errorf("error marshaling message: %w", err), http.StatusInternalServerError, true)
			return
		}

		err = events.NotifyRoomUpdate(ctx, roomID, vibe.RoomEvent{
			Type:    vibe.MessageEvent,
			Payload: payload,
		})
		if err != nil {
			handleError(w, fmt.Errorf("error retaining message: %w", err), http.StatusServiceUnavailable, true)
			return
		}

		err = db.CreateMessageUsage(ctx, roomID, time.UnixMilli(message.CreatedAt))
		if err != nil {
			log.Printf("error recording sent chat message usage: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(payload)
	}
}

// Messages streams retained chat without registering a playback listener.
// @Summary Subscribe to room messages and activity
// @Tags messages
// @Produce text/event-stream
// @Param id path string true "Room ID"
// @Param lastEventId query string false "Last received stream cursor"
// @Success 200 {string} string
// @Failure 401,404,500 {object} vibe.ErrorResponse
// @Router /api/v1/rooms/{id}/messages [get]
func Messages(db vibe.RoomFetcher, events vibe.ReplaySubscriber) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" || session.AuthType == "cast" {
			handleError(w, fmt.Errorf("error reading messages: unauthorized"), http.StatusUnauthorized, false)
			return
		}

		roomID := mux.Vars(r)["id"]
		room, err := db.GetRoom(ctx, roomID, session.UserID)
		if err != nil {
			handleError(w, fmt.Errorf("error fetching messages room: %w", err), http.StatusInternalServerError, true)
			return
		}

		if room.IsEmpty() {
			handleError(w, fmt.Errorf("error reading messages: room not found"), http.StatusNotFound, false)
			return
		}

		cursor := r.Header.Get("Last-Event-ID")
		if cursor == "" {
			cursor = r.URL.Query().Get("lastEventId")
		}

		topic := "chat:" + roomID
		replay, err := events.PrepareReplay(ctx, topic, cursor)
		if err != nil {
			handleError(w, fmt.Errorf("error preparing messages replay: %w", err), http.StatusInternalServerError, true)
			return
		}

		afterID := replay.AfterID
		if replay.RequiresSnapshot {
			afterID = "0-0"
		}

		container, err := events.SubscribeFrom(ctx, topic, afterID)
		if err != nil {
			handleError(w, fmt.Errorf("error subscribing to messages: %w", err), http.StatusInternalServerError, true)
			return
		}

		defer container.Subscription.Destroy()

		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")
		_, err = fmt.Fprint(w, ": connected\n\n")
		if err != nil {
			return
		}

		flusher.Flush()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		messages := container.Subscription.Listen()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err = fmt.Fprint(w, ": heartbeat\n\n")
				if err != nil {
					return
				}

				flusher.Flush()
			case data, open := <-messages:
				if !open {
					return
				}

				var event vibe.RoomEvent
				err = json.Unmarshal(data, &event)
				if err != nil {
					log.Printf("error decoding chat event: %v", err)
					continue
				}

				cursorData, err := json.Marshal(vibe.RoomEventCursor{ID: event.ID})
				if err != nil {
					return
				}

				_, err = fmt.Fprintf(w, "id: %s\nevent: message\ndata: %s\n\nid: %s\nevent: event_cursor\ndata: %s\n\n", event.ID, event.Payload, event.ID, cursorData)
				if err != nil {
					return
				}

				flusher.Flush()
			}
		}
	}
}

// AdminMessageUsage returns sent message counts.
// @Summary Get chat usage overall or for one room
// @Tags admin
// @Produce json
// @Param roomId query string false "Room ID; omitted for all rooms"
// @Success 200 {object} vibe.AdminMessageUsage
// @Failure 400,401,403,500 {object} vibe.ErrorResponse
// @Router /api/v1/admin/messages/usage [get]
func AdminMessageUsage(db vibe.AdminMessageUsageLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID := strings.TrimSpace(r.URL.Query().Get("roomId"))
		if len(roomID) > 200 {
			handleError(w, fmt.Errorf("error reading message usage: room ID too long"), http.StatusBadRequest, false)
			return
		}

		usage, err := db.ListAdminMessageUsage(r.Context(), roomID)
		if err != nil {
			handleError(w, fmt.Errorf("error reading message usage: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.MarshalWrite(w, usage)
		if err != nil {
			log.Printf("error writing message usage: %v", err)
		}
	}
}
