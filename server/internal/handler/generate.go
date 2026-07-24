package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
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
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error active room generation already exists"),
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
				fmt.Errorf("error queueing room generation: %w", err),
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
							Message:   "Playlists can only be generated when the room has 5 songs or fewer.",
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

type GenerateRoomPlaylist struct {
	AI       vibe.PlaylistGenerator
	Cache    vibe.CachedYouTubeSearchFetcherCreator
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
			Error:  vibe.RoomGenerationFailure,
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

	queries := make([]string, 0, len(*playlist))
	for _, track := range *playlist {
		if track.Artist == "" || track.Title == "" {
			continue
		}
		queries = append(queries, track.Artist+" "+track.Title)
	}
	cachedSearches, err := h.Cache.GetCachedYouTubeSearches(
		generationContext,
		queries,
	)
	if err != nil {
		log.Printf("error getting cached youtube searches for room generation: %v", err)
		cachedSearches = []vibe.CachedYouTubeSearch{}
	}

	searchResult, err := h.Searcher.SearchGeneratedPlaylist(
		generationContext,
		*playlist,
		cachedSearches,
	)
	if err != nil {
		var quotaError internalerror.ErrProviderQuotaExceeded
		if errors.As(err, &quotaError) {
			err = h.DB.FailRoomGeneration(
				generationContext,
				generation.RoomID,
				vibe.RoomGenerationYouTubeQuotaFailure,
			)
			if err != nil {
				return fmt.Errorf("error failing room generation after youtube quota exhaustion: %w", err)
			}

			update := vibe.RoomGenerationUpdate{
				Status: vibe.RoomGenerationFailed,
				Error:  vibe.RoomGenerationYouTubeQuotaFailure,
			}
			payload, err := json.Marshal(update)
			if err != nil {
				return fmt.Errorf("error marshaling youtube quota room generation update: %w", err)
			}
			err = h.IPS.NotifyRoomUpdate(generationContext, generation.RoomID, vibe.RoomEvent{
				Type:    vibe.GenerationUpdate,
				Payload: payload,
			})
			if err != nil {
				return fmt.Errorf("error notifying youtube quota room generation failure: %w", err)
			}

			return internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf("error youtube search quota exhausted: %w", quotaError),
				},
			}
		}

		return fmt.Errorf("error searching generated playlist: %w", err)
	}
	err = h.Cache.CacheYouTubeSearches(
		generationContext,
		searchResult.CachedSearches,
	)
	if err != nil {
		log.Printf("error caching youtube searches for room generation: %v", err)
	}
	playlist = &searchResult.Playlist

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

		addedSong, err := h.DB.AddGeneratedSong(generationContext, song)
		if err != nil {
			return fmt.Errorf("error adding generated song: %w", err)
		}
		if addedSong.IsEmpty() {
			continue
		}

		songPayload, err := json.Marshal(addedSong)
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
				CurrentSong:  addedSong,
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

	err = h.DB.CompleteRoomGeneration(ctx, generation.RoomID)
	if err != nil {
		return fmt.Errorf("error completing room generation: %w", err)
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
