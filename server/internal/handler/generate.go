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
				fmt.Errorf("error reading playlist request: %w", err),
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
				fmt.Errorf("error decoding playlist request: %w", err),
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
		if utf8.RuneCountInString(request.Prompt) > generatedPlaylistPromptMaxLength {
			handleError(
				w,
				fmt.Errorf(
					"error playlist prompt exceeds %d characters",
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
			busyError := internalerror.ErrRoomGenerationBusy{
				Err: fmt.Errorf("error active room generation already exists"),
			}
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

			handleError(
				w,
				fmt.Errorf("error queueing room generation: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		createdRoom.IsGenerating = true

		body, err = json.Marshal(createdRoom)
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
		update := vibe.RoomGenerationUpdate{
			Status: vibe.RoomGenerationFailed,
			Error:  "Could not finish generating this playlist.",
		}
		payload, err := json.Marshal(update)
		if err != nil {
			return fmt.Errorf("error marshaling failed room generation update: %w", err)
		}
		err = h.IPS.NotifyRoomUpdate(ctx, generation.RoomID, vibe.RoomEvent{
			Type:    vibe.GenerationUpdate,
			Payload: payload,
		})
		if err != nil {
			return fmt.Errorf("error notifying exhausted room generation: %w", err)
		}
		return nil
	}

	generationContext, cancel := context.WithTimeout(ctx, roomGenerationTimeout)
	defer cancel()

	room, err := h.DB.GetRoom(generationContext, generation.RoomID, "")
	if err != nil {
		return fmt.Errorf("error getting room for generation: %w", err)
	}
	if room.IsEmpty() {
		return fmt.Errorf("error room for generation is empty")
	}

	playlist, err := h.AI.GeneratePlaylist(generationContext, generation.Prompt)
	if err != nil {
		return fmt.Errorf("error generating playlist: %w", err)
	}

	playlist, err = h.Searcher.SearchGeneratedPlaylist(generationContext, *playlist)
	if err != nil {
		return fmt.Errorf("error searching generated playlist: %w", err)
	}

	playbackState, err := h.DB.GetPlaybackState(generationContext, room.ID)
	if err != nil {
		return fmt.Errorf("error getting generated room playback: %w", err)
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

		result, err := h.DB.AddSong(generationContext, song)
		if err != nil {
			return fmt.Errorf("error adding generated song: %w", err)
		}
		if result.Outcome != vibe.AddSongOutcomeAdded {
			continue
		}

		songPayload, err := json.Marshal(result.Song)
		if err != nil {
			return fmt.Errorf("error marshaling generated song: %w", err)
		}

		err = h.IPS.NotifyRoomUpdate(generationContext, room.ID, vibe.RoomEvent{
			Type:    vibe.SongAdded,
			Payload: songPayload,
		})
		if err != nil {
			return fmt.Errorf("error notifying generated song: %w", err)
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
				return fmt.Errorf("error starting generated room playback: %w", err)
			}

			playbackPayload, err := json.Marshal(playbackState)
			if err != nil {
				return fmt.Errorf("error marshaling generated room playback: %w", err)
			}
			err = h.IPS.NotifyRoomUpdate(generationContext, room.ID, vibe.RoomEvent{
				Type:    vibe.PlaybackUpdate,
				Payload: playbackPayload,
			})
			if err != nil {
				return fmt.Errorf("error notifying generated room playback: %w", err)
			}
			shouldStartPlayback = false
		}
	}

	err = h.DB.DeleteRoomGeneration(ctx, generation.RoomID)
	if err != nil {
		return fmt.Errorf("error deleting completed room generation: %w", err)
	}

	update := vibe.RoomGenerationUpdate{
		Status: vibe.RoomGenerationCompleted,
	}
	payload, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("error marshaling completed room generation update: %w", err)
	}
	err = h.IPS.NotifyRoomUpdate(ctx, generation.RoomID, vibe.RoomEvent{
		Type:    vibe.GenerationUpdate,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("error notifying completed room generation: %w", err)
	}

	return nil
}

const roomGenerationTimeout = 4 * time.Minute

const roomGenerationBusyRetryAfterSeconds = "60"

const generatedPlaylistPromptMaxLength = 300
