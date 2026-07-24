package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/zoff-music/vibes-backend/client"
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
	maxPromptLength int,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var request vibe.GeneratedPlaylistRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, playlistRequestMaxBytes))
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding playlist request: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}
		var trailingJSON json.RawMessage
		err = decoder.Decode(&trailingJSON)
		if !errors.Is(err, io.EOF) {
			handleError(
				w,
				fmt.Errorf("error playlist request must contain one JSON object"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		request.Prompt = strings.TrimSpace(request.Prompt)
		if request.Prompt == "" {
			handleError(
				w,
				fmt.Errorf("error playlist prompt is required"),
				http.StatusBadRequest,
				false,
			)
			return
		}
		if utf8.RuneCountInString(request.Prompt) > maxPromptLength {
			handleError(
				w,
				fmt.Errorf("error playlist prompt exceeds %d characters", maxPromptLength),
				http.StatusBadRequest,
				false,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error generated room session is missing"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		hasActiveGeneration, err := db.HasActiveRoomGeneration(ctx)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error checking active room generation: %w", err),
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
					Err: vibe.RoomGenerationBusyError{},
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

		candidates, err := vibe.GenerateRoomNameCandidates()
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error generating room name candidates: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		suggestion, err := db.SuggestRoomName(ctx, candidates)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error suggesting generated room name: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		settings := vibe.DefaultRoomSettings()
		room := vibe.Room{
			ID:            helper.Slugify(suggestion.Name),
			Name:          suggestion.Name,
			Mode:          vibe.RoomModeServer,
			HostID:        session.UserID,
			Settings:      settings,
			CreatedAt:     time.Now(),
			ActiveSources: settings.EnabledSources,
		}
		createdRoom, err := db.CreateRoom(ctx, &room)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error creating generated room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = db.CreateRoomGeneration(ctx, createdRoom.ID, request.Prompt)
		if err != nil {
			var busyError vibe.RoomGenerationBusyError
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

			handleError(
				w,
				fmt.Errorf("error queueing room generation: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		createdRoom.IsGenerating = true

		body, err := json.Marshal(createdRoom)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling generated room: %w", err),
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

type GenerateRoomPlaylist struct {
	AI       vibe.PlaylistGenerator
	DB       vibe.RoomGenerationWorker
	IPS      vibe.RoomEventNotifier
	Searcher vibe.GeneratedPlaylistSearcher
}

func (h *GenerateRoomPlaylist) Handle(ctx context.Context, data []byte) error {
	generation, err := h.DB.ProcessNextRoomGeneration(ctx)
	if err != nil {
		return fmt.Errorf("error processing next room generation: %w", err)
	}
	if generation.Exhausted {
		err = h.notifyGenerationUpdate(ctx, generation.RoomID, vibe.RoomGenerationFailed)
		if err != nil {
			return fmt.Errorf("error notifying exhausted room generation: %w", err)
		}
		return nil
	}

	generationContext, cancel := context.WithTimeout(ctx, roomGenerationTimeout)
	defer cancel()

	room, err := h.DB.GetRoom(generationContext, generation.RoomID, "")
	if err != nil {
		return h.handleGenerationError(ctx, *generation, fmt.Errorf("error getting room for generation: %w", err))
	}
	if room.IsEmpty() {
		return h.handleGenerationError(ctx, *generation, fmt.Errorf("error room for generation is empty"))
	}

	playlist, err := h.AI.GeneratePlaylist(generationContext, generation.Prompt)
	if err != nil {
		return h.handleGenerationError(ctx, *generation, fmt.Errorf("error generating playlist: %w", err))
	}

	playlist, err = h.Searcher.SearchGeneratedPlaylist(generationContext, *playlist)
	if err != nil {
		return h.handleGenerationError(ctx, *generation, fmt.Errorf("error searching generated playlist: %w", err))
	}

	playbackState, err := h.DB.GetPlaybackState(generationContext, room.ID)
	if err != nil {
		return h.handleGenerationError(ctx, *generation, fmt.Errorf("error getting generated room playback: %w", err))
	}
	shouldStartPlayback := playbackState.CurrentSong == nil

	for _, track := range *playlist {
		song := &vibe.Song{
			ID:           uuid.NewString(),
			RoomID:       room.ID,
			SourceType:   vibe.SourceTypeYouTube,
			SourceID:     track.YouTubeID,
			Title:        track.Title,
			Artist:       track.Artist,
			ThumbnailURL: track.ThumbnailURL,
			Duration:     track.Duration,
			AddedBy:      room.HostID,
			AddedAt:      time.Now(),
		}

		result, addErr := h.DB.AddSong(generationContext, song)
		if addErr != nil {
			return h.handleGenerationError(ctx, *generation, fmt.Errorf("error adding generated song: %w", addErr))
		}
		if result.Outcome != vibe.AddSongOutcomeAdded {
			continue
		}

		songPayload, marshalErr := json.Marshal(result.Song)
		if marshalErr != nil {
			return h.handleGenerationError(ctx, *generation, fmt.Errorf("error marshaling generated song: %w", marshalErr))
		}

		notifyErr := h.IPS.NotifyRoomUpdate(generationContext, room.ID, vibe.RoomEvent{
			Type:    vibe.SongAdded,
			Payload: songPayload,
		})
		if notifyErr != nil {
			return h.handleGenerationError(ctx, *generation, fmt.Errorf("error notifying generated song: %w", notifyErr))
		}

		if shouldStartPlayback {
			playbackState := &vibe.PlaybackState{
				RoomID:       room.ID,
				CurrentSong:  &result.Song,
				IsPlaying:    true,
				PositionMs:   0,
				UpdatedAt:    time.Now(),
				ServerTimeMs: int(time.Now().UnixMilli()),
			}
			err = h.DB.UpsertPlaybackState(generationContext, playbackState)
			if err != nil {
				return h.handleGenerationError(ctx, *generation, fmt.Errorf("error starting generated room playback: %w", err))
			}

			playbackPayload, marshalErr := json.Marshal(playbackState)
			if marshalErr != nil {
				return h.handleGenerationError(ctx, *generation, fmt.Errorf("error marshaling generated room playback: %w", marshalErr))
			}
			notifyErr = h.IPS.NotifyRoomUpdate(generationContext, room.ID, vibe.RoomEvent{
				Type:    vibe.PlaybackUpdate,
				Payload: playbackPayload,
			})
			if notifyErr != nil {
				return h.handleGenerationError(ctx, *generation, fmt.Errorf("error notifying generated room playback: %w", notifyErr))
			}
			shouldStartPlayback = false
		}
	}

	err = h.DB.DeleteRoomGeneration(ctx, generation.RoomID)
	if err != nil {
		return fmt.Errorf("error deleting completed room generation: %w", err)
	}

	err = h.notifyGenerationUpdate(ctx, generation.RoomID, vibe.RoomGenerationCompleted)
	if err != nil {
		return fmt.Errorf("error notifying completed room generation: %w", err)
	}

	return nil
}

func (h *GenerateRoomPlaylist) handleGenerationError(
	ctx context.Context,
	generation vibe.RoomGeneration,
	generationErr error,
) error {
	if generation.Attempt < vibe.RoomGenerationMaxAttempts {
		return fmt.Errorf("error processing room generation attempt %d: %w", generation.Attempt, generationErr)
	}

	err := h.DB.DeleteRoomGeneration(ctx, generation.RoomID)
	if err != nil {
		return fmt.Errorf("error deleting failed room generation: %w", err)
	}

	err = h.notifyGenerationUpdate(ctx, generation.RoomID, vibe.RoomGenerationFailed)
	if err != nil {
		return fmt.Errorf("error notifying failed room generation: %w", err)
	}

	return fmt.Errorf("error room generation failed after %d attempts: %w", generation.Attempt, generationErr)
}

func (h *GenerateRoomPlaylist) notifyGenerationUpdate(
	ctx context.Context,
	roomID string,
	status vibe.RoomGenerationStatus,
) error {
	update := vibe.RoomGenerationUpdate{
		Status: status,
	}
	if status == vibe.RoomGenerationFailed {
		update.Error = "Could not finish generating this playlist."
	}

	payload, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("error marshaling room generation update: %w", err)
	}

	err = h.IPS.NotifyRoomUpdate(ctx, roomID, vibe.RoomEvent{
		Type:    vibe.GenerationUpdate,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("error notifying room generation update: %w", err)
	}

	return nil
}

const playlistRequestMaxBytes = 4096

const roomGenerationTimeout = 4 * time.Minute

const roomGenerationBusyRetryAfterSeconds = "60"
