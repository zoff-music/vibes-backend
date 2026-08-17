package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// CreateGeneratedRoom handles POST /api/v1/rooms/generation.
//
//	@Summary	Create a room and queue playlist generation
//	@Tags		rooms
//	@Accept		json
//	@Produce	json
//	@Param		request	body		vibe.GeneratedPlaylistRequest	true	"Playlist prompt"
//	@Success	201	{object}	vibe.Room
//	@Failure	400	{object}	map[string]string
//	@Failure	401	{object}	map[string]string
//	@Failure	429	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/rooms/generation [post]
func CreateGeneratedRoom(
	db vibe.GeneratedRoomCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error reading playlist request in CreateGeneratedRoom handler: %w",
					err,
				),
				http.StatusBadRequest,
				false,
			)
			return
		}

		var request vibe.GeneratedPlaylistRequest
		err = json.Unmarshal(body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error decoding playlist request in CreateGeneratedRoom handler: %w",
					err,
				),
				http.StatusBadRequest,
				false,
			)
			return
		}

		request.Prompt = strings.TrimSpace(request.Prompt)
		if request.Prompt == "" {
			handleError(
				w,
				fmt.Errorf(
					"error validating playlist request in CreateGeneratedRoom handler: prompt is required",
				),
				http.StatusBadRequest,
				false,
			)
			return
		}
		if utf8.RuneCountInString(request.Prompt) > generatedPlaylistPromptMaxLength {
			handleError(
				w,
				fmt.Errorf(
					"error validating playlist request in CreateGeneratedRoom handler: prompt exceeds %d characters",
					generatedPlaylistPromptMaxLength,
				),
				http.StatusBadRequest,
				false,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf(
					"error getting session in CreateGeneratedRoom handler: session is missing",
				),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		hasActiveGeneration, err := db.HasActiveRoomGeneration(ctx)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error checking active room generation in CreateGeneratedRoom handler: %w",
					err,
				),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if hasActiveGeneration {
			w.Header().Set("Retry-After", roomGenerationBusyRetryAfterSeconds)
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf(
						"error validating room generation in CreateGeneratedRoom handler: active generation already exists",
					),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "room_generation_busy",
						Message:   "A playlist is already being generated. Please wait and try again.",
						Propagate: true,
					},
					StatusCode: http.StatusTooManyRequests,
				},
				http.StatusTooManyRequests,
				false,
			)
			return
		}

		reservation, err := db.ReserveSuggestedRoomName(ctx, session.UserID)
		if err != nil {
			if handleRoomNameUnavailable(w, err) {
				return
			}

			handleError(
				w,
				fmt.Errorf(
					"error reserving generated room name in CreateGeneratedRoom handler: %w",
					err,
				),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		settings := vibe.DefaultRoomSettings()
		room := vibe.Room{
			ID:            helper.Slugify(reservation.Name),
			Name:          reservation.Name,
			Mode:          vibe.RoomModeServer,
			HostID:        session.UserID,
			Settings:      settings,
			CreatedAt:     time.Now(),
			ActiveSources: settings.EnabledSources,
		}
		createdRoom, err := db.CreateRoom(ctx, &room, reservation.Token)
		if err != nil {
			if handleRoomNameUnavailable(w, err) {
				return
			}

			handleError(
				w,
				fmt.Errorf(
					"error creating generated room in CreateGeneratedRoom handler: %w",
					err,
				),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = db.CreateRoomGeneration(ctx, createdRoom.ID, request.Prompt)
		if err != nil {
			var busyError internalerror.ErrRoomGenerationBusy
			if errors.As(err, &busyError) {
				w.Header().Set("Retry-After", roomGenerationBusyRetryAfterSeconds)
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: busyError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_generation_busy",
							Message:   "A playlist is already being generated. Please wait and try again.",
							Propagate: true,
						},
						StatusCode: http.StatusTooManyRequests,
					},
					http.StatusTooManyRequests,
					false,
				)
				return
			}

			var dailyLimitError internalerror.ErrRoomGenerationDailyLimit
			if errors.As(err, &dailyLimitError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: dailyLimitError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_generation_daily_limit",
							Message:   "This room has reached its daily playlist generation limit.",
							Propagate: true,
						},
						StatusCode: http.StatusTooManyRequests,
					},
					http.StatusTooManyRequests,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf(
					"error queueing room generation in CreateGeneratedRoom handler: %w",
					err,
				),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		createdRoom.IsGenerating = true
		createdRoom.GenerationCount = 1

		body, err = json.Marshal(createdRoom)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error marshaling generated room in CreateGeneratedRoom handler: %w",
					err,
				),
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

// CreateRoomGeneration handles POST /api/v1/rooms/{id}/generations.
//
//	@Summary	Queue playlist generation for a room
//	@Tags		rooms
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string							true	"Room ID"
//	@Param		request	body		vibe.GeneratedPlaylistRequest	true	"Playlist prompt"
//	@Success	202		{object}	vibe.RoomGenerationUpdate
//	@Failure	400		{object}	map[string]string
//	@Failure	401		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	409		{object}	map[string]string
//	@Failure	429		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/generations [post]
func CreateRoomGeneration(
	creator vibe.RoomGenerationCreator,
	maxExistingSongs int,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		body, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error reading playlist request in CreateRoomGeneration handler: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		var request vibe.GeneratedPlaylistRequest
		err = json.Unmarshal(body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding playlist request in CreateRoomGeneration handler: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		request.Prompt = strings.TrimSpace(request.Prompt)
		if request.Prompt == "" {
			handleError(
				w,
				fmt.Errorf("error validating playlist request in CreateRoomGeneration handler: prompt is required"),
				http.StatusBadRequest,
				false,
			)
			return
		}
		if utf8.RuneCountInString(request.Prompt) > generatedPlaylistPromptMaxLength {
			handleError(
				w,
				fmt.Errorf(
					"error validating playlist request in CreateRoomGeneration handler: prompt exceeds %d characters",
					generatedPlaylistPromptMaxLength,
				),
				http.StatusBadRequest,
				false,
			)
			return
		}

		err = creator.CreateRoomGeneration(ctx, roomID, request.Prompt)
		if err != nil {
			var busyError internalerror.ErrRoomGenerationBusy
			if errors.As(err, &busyError) {
				w.Header().Set("Retry-After", roomGenerationBusyRetryAfterSeconds)
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: busyError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_generation_busy",
							Message:   "A playlist is already being generated. Please wait and try again.",
							Propagate: true,
						},
						StatusCode: http.StatusTooManyRequests,
					},
					http.StatusTooManyRequests,
					false,
				)
				return
			}

			var songLimitError internalerror.ErrRoomGenerationSongLimit
			if errors.As(err, &songLimitError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: songLimitError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_generation_song_limit",
							Message: fmt.Sprintf(
								"Playlists can only be generated when the room has %d songs or fewer.",
								maxExistingSongs,
							),
							Propagate: true,
						},
						StatusCode: http.StatusConflict,
					},
					http.StatusConflict,
					false,
				)
				return
			}

			var dailyLimitError internalerror.ErrRoomGenerationDailyLimit
			if errors.As(err, &dailyLimitError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: dailyLimitError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_generation_daily_limit",
							Message:   "This room has reached its daily playlist generation limit.",
							Propagate: true,
						},
						StatusCode: http.StatusTooManyRequests,
					},
					http.StatusTooManyRequests,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf("error queueing room generation in CreateRoomGeneration handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		update := vibe.RoomGenerationUpdate{
			Status: vibe.RoomGenerationGenerating,
		}
		body, err = json.Marshal(&update)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshalling response in CreateRoomGeneration handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(body)
	}
}

const roomGenerationBusyRetryAfterSeconds = "60"
const generatedPlaylistPromptMaxLength = 300
