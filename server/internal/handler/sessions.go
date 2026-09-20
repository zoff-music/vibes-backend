package handler

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetSessionProfile handles GET /api/v1/sessions.
//
//	@Summary	Get the current session profile
//	@Tags		sessions
//	@Produce	json
//	@Success	200	{object}	vibe.SessionProfile
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/sessions [get]
func GetSessionProfile(db vibe.SessionProfileFetcherCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error getting session profile: unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting session profile: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(profile)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling session profile: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// UpdateSessionProfile handles PATCH /api/v1/sessions.
//
//	@Summary	Update the current session profile
//	@Tags		sessions
//	@Accept		json
//	@Produce	json
//	@Param		request	body		vibe.UpdateSessionProfileRequest	true	"Session profile"
//	@Success	200		{object}	vibe.SessionProfile
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v1/sessions [patch]
func UpdateSessionProfile(db vibe.SessionProfileRoomUpdater, events vibe.RoomEventNotifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error updating session profile: unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		var request vibe.UpdateSessionProfileRequest
		err := json.UnmarshalRead(http.MaxBytesReader(w, r.Body, 4096), &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding session profile: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		if !request.Validate() {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error validating session profile name"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "profile_name_invalid",
						Message:   fmt.Sprintf("Use between 1 and %d characters for your name.", vibe.SessionNameMaxLength),
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		previous, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			handleError(w, fmt.Errorf("error reading previous profile: %w", err), http.StatusInternalServerError, true)
			return
		}

		profile, err := db.UpdateSessionProfile(ctx, session.UserID, strings.TrimSpace(request.Name))
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error updating session profile: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if previous.Name != profile.Name {
			rooms, roomsErr := db.GetSessionRooms(ctx, session.UserID)
			if roomsErr != nil {
				log.Printf("error fetching rooms for name change: %v", roomsErr)
			}

			for _, roomID := range rooms {
				room, roomErr := db.GetRoom(ctx, roomID, session.UserID)
				if roomErr != nil {
					log.Printf("error fetching room for name change: %v", roomErr)
					continue
				}

				if room.IsEmpty() {
					continue
				}

				message := vibe.RoomMessage{
					ID:        uuid.NewString(),
					UserID:    session.UserID,
					Name:      previous.Name,
					IsAdmin:   room.IsAdmin,
					Kind:      "renamed",
					Text:      profile.Name,
					CreatedAt: time.Now().UnixMilli(),
				}

				payload, chatErr := json.Marshal(message)
				if chatErr != nil {
					log.Printf("error marshaling name change: %v", chatErr)
					continue
				}

				chatErr = events.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
					Type:    vibe.MessageEvent,
					Payload: payload,
				})
				if chatErr != nil {
					log.Printf("error publishing name change: %v", chatErr)
				}
			}
		}

		body, err := json.Marshal(profile)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling session profile: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
