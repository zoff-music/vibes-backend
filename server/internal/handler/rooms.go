package handler

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// ReserveRoomName handles POST /api/v1/rooms/reservations.
//
//	@Summary		Reserve a custom or generated room name
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			request	body		vibe.RoomNameReservationRequest	true	"Room name reservation request"
//	@Success		201		{object}	vibe.RoomNameReservation
//	@Failure		400		{object}	vibe.ErrorResponse
//	@Failure		409		{object}	vibe.ErrorResponse
//	@Failure		500		{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Router			/api/v1/rooms/reservations [post]
func ReserveRoomName(db vibe.RoomNameReserver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req vibe.RoomNameReservationRequest
		err := json.UnmarshalRead(http.MaxBytesReader(w, r.Body, 4096), &req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding room name reservation request: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		if !req.Validate() {
			handleError(w, client.ErrorCodeWrapper{
				Err: fmt.Errorf("error validating room name reservation"),
				ResponseBody: client.ErrorCodeResponseBody{
					Namespace: "vibes-backend",
					Error:     "room_name_invalid",
					Message:   fmt.Sprintf("Use between 1 and %d characters for the room name.", vibe.RoomNameMaxLength),
					Propagate: true,
				},
				StatusCode: http.StatusBadRequest,
			}, http.StatusBadRequest, false)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error getting room name reservation session"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		var reservation *vibe.RoomNameReservation
		if req.Name == "" {
			reservation, err = db.ReserveSuggestedRoomName(
				ctx,
				session.UserID,
			)
		} else {
			roomID := helper.Slugify(req.Name)
			if roomID == "" {
				handleError(
					w,
					fmt.Errorf("error invalid room name"),
					http.StatusBadRequest,
					false,
				)
				return
			}

			reservation, err = db.ReserveRoomName(
				ctx,
				roomID,
				session.UserID,
			)
		}
		if err != nil {
			var unavailableError internalerror.ErrRoomNameUnavailable
			if errors.As(err, &unavailableError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: unavailableError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_name_unavailable",
							Message:   "This room name is unavailable or its reservation expired.",
							Propagate: true,
						},
						StatusCode: http.StatusConflict,
					},
					http.StatusConflict,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf("error reserving room name: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(reservation)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling room name reservation: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	}
}

// SuggestRoomName handles GET /api/v1/rooms/suggestions.
//
//	@Summary		Suggest an available, memorable room name
//	@Tags			rooms
//	@Produce		json
//	@Success		200	{object}	vibe.RoomNameReservation
//	@Failure		500	{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	409	{object}	vibe.ErrorResponse
//	@Router			/api/v1/rooms/suggestions [get]
func SuggestRoomName(db vibe.RoomNameSuggester) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error getting room name suggestion session"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		suggestion, err := db.ReserveSuggestedRoomName(ctx, session.UserID)
		if err != nil {
			var unavailableError internalerror.ErrRoomNameUnavailable
			if errors.As(err, &unavailableError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: unavailableError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_name_unavailable",
							Message:   "This room name is unavailable or its reservation expired.",
							Propagate: true,
						},
						StatusCode: http.StatusConflict,
					},
					http.StatusConflict,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf("error suggesting room name: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(suggestion)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling room name suggestion: %w", err),
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

// RoomExists handles HEAD /api/v1/rooms/{id}.
//
//	@Summary		Check whether a room exists
//	@Tags			rooms
//	@Param			id	path	string	true	"Room ID"
//	@Success		200
//	@Failure		404
//	@Failure		500
//	@Router			/api/v1/rooms/{id} [head]
func RoomExists(db vibe.RoomExistenceChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		exists, err := db.RoomExists(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error checking room existence: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// CleanupRoomNameReservations cleans up expired room name reservations.
type CleanupRoomNameReservations struct {
	DB vibe.RoomNameReservationDeleter
}

func (h *CleanupRoomNameReservations) Handle(
	ctx context.Context,
	_ []byte,
) error {
	deleted, err := h.DB.DeleteExpiredRoomNameReservations(ctx)
	if err != nil {
		return fmt.Errorf(
			"error deleting expired room name reservations in CleanupRoomNameReservations.Handle: %w",
			err,
		)
	}

	if deleted > 0 {
		log.Printf("Cleaned up %d expired room name reservations", deleted)
	}

	return nil
}
