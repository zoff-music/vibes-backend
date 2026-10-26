package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// CreateGeneration queues a new room without exposing a temporary room name.
//
// @Summary Generate a named room and playlist
// @Tags generations
// @Accept json
// @Produce json
// @Param request body vibe.GeneratedRoomRequestV2 true "Playlist prompt"
// @Success 202 {object} vibe.GenerationStatus
// @Failure 400 {object} vibe.ErrorResponse
// @Failure 401 {object} vibe.ErrorResponse
// @Failure 429 {object} vibe.ErrorResponse
// @Failure 500 {object} vibe.ErrorResponse
// @Router /api/v1/generations [post]
func CreateGeneration(creator vibe.GenerationCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error creating generation: missing session"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		var request vibe.GeneratedRoomRequestV2
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding generation request: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		request.Prompt = strings.TrimSpace(request.Prompt)
		if request.RoomType == "" {
			request.RoomType = vibe.RoomTypeMusic
		}
		if request.Prompt == "" || utf8.RuneCountInString(request.Prompt) > generatedPlaylistPromptMaxLength || !request.RoomType.IsValid() {
			handleError(
				w,
				fmt.Errorf("error validating generation: prompt must contain 1–300 characters and roomType must be MUSIC or WATCH"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		status, err := creator.CreateGeneration(ctx, session.UserID, request)
		if err != nil {
			var busyError internalerror.ErrRoomGenerationBusy
			if errors.As(err, &busyError) {
				w.Header().Set("Retry-After", roomGenerationBusyRetryAfterSeconds)
				handleError(
					w,
					internalerror.ErrorCodeWrapper{
						Err: busyError,
						ResponseBody: internalerror.ErrorCodeResponseBody{
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
				fmt.Errorf("error creating generation: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(status)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error encoding generation: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(body)
	}
}

// GetGeneration returns only a generation belonging to the current session.
//
// @Summary Get a room generation status
// @Tags generations
// @Produce json
// @Param id path string true "Generation ID"
// @Success 200 {object} vibe.GenerationStatus
// @Failure 400 {object} vibe.ErrorResponse
// @Failure 401 {object} vibe.ErrorResponse
// @Failure 404 {object} vibe.ErrorResponse
// @Failure 500 {object} vibe.ErrorResponse
// @Router /api/v1/generations/{id} [get]
func GetGeneration(fetcher vibe.GenerationFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error getting generation: missing session"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil || id <= 0 {
			handleError(
				w,
				fmt.Errorf("error validating generation ID: expected a positive integer"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		status, err := fetcher.GetGeneration(ctx, id, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting generation: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if status.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error getting generation: not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		body, err := json.Marshal(status)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error encoding generation: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
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
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	403		{object}	vibe.ErrorResponse
//	@Failure	409		{object}	vibe.ErrorResponse
//	@Failure	429		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v1/rooms/{id}/generations [post]
func CreateRoomGeneration(
	creator vibe.RoomGenerationCreator,
	maxExistingPlaylistItems int,
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
					internalerror.ErrorCodeWrapper{
						Err: busyError,
						ResponseBody: internalerror.ErrorCodeResponseBody{
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

			var playlistItemLimitError internalerror.ErrRoomGenerationPlaylistItemLimit
			if errors.As(err, &playlistItemLimitError) {
				handleError(
					w,
					internalerror.ErrorCodeWrapper{
						Err: playlistItemLimitError,
						ResponseBody: internalerror.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "room_generation_song_limit",
							Message: fmt.Sprintf(
								"Playlists can only be generated when the room has %d songs or fewer.",
								maxExistingPlaylistItems,
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
					internalerror.ErrorCodeWrapper{
						Err: dailyLimitError,
						ResponseBody: internalerror.ErrorCodeResponseBody{
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
	Cache    vibe.GeneratedPlaylistCacheNotifier
	DB       vibe.RoomGenerationWorker
	Searcher vibe.GeneratedPlaylistSearcher
}

func (h *GenerateRoomPlaylist) Handle(ctx context.Context, _ []byte) error {
	generation, err := h.DB.ProcessNextRoomGeneration(ctx)
	if err != nil {
		return fmt.Errorf("error processing next room generation in Handle: %w", err)
	}

	if generation.Exhausted {
		if generation.RoomID == "" {
			return nil
		}

		update := vibe.RoomGenerationUpdate{
			Status: vibe.RoomGenerationFailed,
			Error:  vibe.RoomGenerationFailure,
		}

		payload, err := json.Marshal(update)
		if err != nil {
			return fmt.Errorf("error marshaling failed room generation update in Handle: %w", err)
		}

		err = h.Cache.NotifyRoomUpdateV3(ctx, generation.RoomID, vibe.RoomEventV3{
			Type:    vibe.GenerationUpdate,
			Payload: payload,
		})
		if err != nil {
			return fmt.Errorf("error notifying exhausted room generation in Handle: %w", err)
		}

		return nil
	}

	room := generation.Room
	if generation.RoomID == "" {
		room.RoomType = generation.RoomType
	}
	playbackState := generation.PlaybackState
	currentPlaylistItem := playbackState.CurrentPlaylistItem

	prompt, err := vibe.GeneratePlaylistPrompt(
		generation.Prompt,
		currentPlaylistItem,
		generation.PlaylistItems,
	)
	if err != nil {
		return fmt.Errorf("error generating playlist prompt in Handle: %w", err)
	}

	generated, err := h.AI.GeneratePlaylist(ctx, prompt, room.RoomType)
	if err != nil {
		return fmt.Errorf("error generating playlist in Handle: %w", err)
	}

	playlist := &generated.Items
	if generation.RoomID == "" {
		createdRoom, err := h.DB.CreateGenerationRoom(ctx, generation, generated.Names)
		if err != nil {
			return fmt.Errorf("error creating named generation room in Handle: %w", err)
		}

		room = *createdRoom
		generation.RoomID = createdRoom.ID

		err = h.Cache.NotifyAdminRoomChange(ctx, createdRoom.ID)
		if err != nil {
			log.Printf("error notifying generated admin room: %v", err)
		}
	}

	queries := make([]string, 0, len(*playlist))
	for _, playlistItem := range *playlist {
		if playlistItem.Publisher == "" || playlistItem.Title == "" {
			continue
		}

		queries = append(queries, playlistItem.Publisher+" "+playlistItem.Title)
	}

	cachedSearches, err := h.Cache.GetCachedProviderResolutions(ctx, vibe.SourceTypeYouTube, queries, room.RoomType)
	if err != nil {
		return fmt.Errorf("error reading youtube resolution cache for generation: %w", err)
	}
	fullSearches, err := h.Cache.GetCachedProviderSearches(ctx, vibe.SourceTypeYouTube, queries, room.RoomType)
	if err != nil {
		return fmt.Errorf("error reading youtube search cache for generation: %w", err)
	}
	cachedSearches = append(cachedSearches, fullSearches...)

	searchQuotaReset, err := h.Cache.GetProviderQuotaReset(
		ctx,
		vibe.SourceTypeYouTube,
		vibe.ProviderQuotaOperationSearch,
	)
	if err != nil {
		return fmt.Errorf("error reading youtube quota guard for generation: %w", err)
	}

	searchResult, err := h.Searcher.SearchGeneratedPlaylist(
		ctx,
		*playlist,
		cachedSearches,
		room.RoomType,
	)

	if err == nil && len(searchResult.Unresolved) > 0 && time.Now().Before(searchQuotaReset) {
		log.Printf("event=search_quota_blocked provider=youtube purpose=generation upstream_attempted=false reset_at=%s", searchQuotaReset.UTC().Format(time.RFC3339))
	}

	if err == nil && len(searchResult.Unresolved) > 0 && !time.Now().Before(searchQuotaReset) {
		freshResolutions := searchResult.CachedSearches
		resolvedSearches := append(cachedSearches, searchResult.CachedSearches...)
		for _, candidate := range searchResult.Unresolved {
			query := candidate.Publisher + " " + candidate.Title
			usage := vibe.GenerateSearchUsage(vibe.SourceTypeYouTube, query, false)
			usage.RoomID = room.ID
			token := uuid.NewString()
			claimKey := "youtube:" + string(room.RoomType) + ":" + usage.QueryHash
			claim, claimErr := h.DB.ClaimSearch(ctx, claimKey, token)
			if claimErr != nil {
				return fmt.Errorf("error claiming generated search: %w", claimErr)
			}
			if claim.IsEmpty() {
				continue
			}
			defer func() {
				releaseErr := h.DB.ReleaseSearch(ctx, *claim)
				if releaseErr != nil {
					log.Printf("error releasing generated search claim: %v", releaseErr)
				}
			}()

			existing, cacheErr := h.Cache.GetCachedProviderSearches(ctx, vibe.SourceTypeYouTube, []string{query}, room.RoomType)
			if cacheErr != nil {
				return fmt.Errorf("error checking generated search cache: %w", cacheErr)
			}
			if len(existing) > 0 {
				resolvedSearches = append(resolvedSearches, existing...)
				continue
			}

			// Space generation fallbacks so interactive searches can share the burst budget.
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return fmt.Errorf("error waiting for generated search: %w", ctx.Err())
			case <-timer.C:
			}
			attempt := vibe.SearchAttempt{
				ID: token, QueryHash: usage.QueryHash, RoomID: room.ID,
				UserID: room.HostID, Purpose: vibe.SearchPurposeGeneration,
			}
			allowance, reserveErr := h.DB.ReserveSearch(ctx, attempt)
			if reserveErr != nil {
				return fmt.Errorf("error reserving generated search: %w", reserveErr)
			}
			if !allowance.Allowed {
				log.Printf("event=search_allowance_denied provider=youtube purpose=generation reason=budget_or_burst_guard upstream_attempted=false daily_reset_at=%s", allowance.ResetAt.UTC().Format(time.RFC3339))

				break
			}
			items, searchErr := h.Searcher.SearchProviderItems(ctx, query, room.RoomType)
			outcome := "success"
			if searchErr != nil {
				outcome = "failure"
			}
			var quotaError internalerror.ErrProviderQuotaExceeded
			if errors.As(searchErr, &quotaError) {
				log.Printf("event=search_quota_exhausted provider=youtube purpose=generation upstream_attempted=true reset_at=%s", quotaError.ResetAt.UTC().Format(time.RFC3339))

				outcome = "quota"
				searchQuotaReset = quotaError.ResetAt
				cacheErr = h.Cache.CacheProviderQuotaReset(ctx, vibe.SourceTypeYouTube, vibe.ProviderQuotaOperationSearch, quotaError.ResetAt)
				if cacheErr != nil {
					log.Printf("error caching generation quota reset: %v", cacheErr)
				}
			}
			recordErr := h.DB.UpdateSearchAttempt(ctx, token, outcome)
			if recordErr != nil {
				log.Printf("error recording generated search outcome: %v", recordErr)
			}
			recordErr = h.DB.CreateSearchUsages(ctx, []vibe.SearchUsage{usage})
			if recordErr != nil {
				log.Printf("error recording generated search usage: %v", recordErr)
			}
			if searchErr != nil {
				break
			}
			search := vibe.CachedProviderSearch{Query: query, Items: items, FetchedAt: time.Now().UTC()}
			resolvedSearches = append(resolvedSearches, search)
			cacheErr = h.Cache.CacheProviderSearches(ctx, vibe.SourceTypeYouTube, []vibe.CachedProviderSearch{search}, room.RoomType)
			if cacheErr != nil {
				log.Printf("error caching generated fallback search: %v", cacheErr)
			}
		}
		searchResult, err = h.Searcher.SearchGeneratedPlaylist(ctx, *playlist, resolvedSearches, room.RoomType)
		if err == nil {
			searchResult.CachedSearches = append(searchResult.CachedSearches, freshResolutions...)
		}
	}
	if err == nil && len(searchResult.Playlist) == 0 {
		err = fmt.Errorf("error resolving generated playlist: no playable matches within search allowance")
		if time.Now().Before(searchQuotaReset) {
			err = internalerror.ErrProviderQuotaExceeded{
				Err: err, Provider: vibe.SourceTypeYouTube, ResetAt: searchQuotaReset,
			}
		}
	}
	if err != nil {
		var quotaError internalerror.ErrProviderQuotaExceeded
		if errors.As(err, &quotaError) {
			err = h.Cache.CacheProviderQuotaReset(
				ctx,
				vibe.SourceTypeYouTube,
				vibe.ProviderQuotaOperationSearch,
				quotaError.ResetAt,
			)
			if err != nil {
				log.Printf(
					"error caching youtube quota reset for room generation: %v",
					err,
				)
			}
			err = h.DB.FailRoomGeneration(
				ctx,
				generation.RoomID,
				vibe.RoomGenerationYouTubeQuotaFailure,
			)
			if err != nil {
				return fmt.Errorf(
					"error failing room generation after youtube quota exhaustion in Handle: %w",
					err,
				)
			}

			update := vibe.RoomGenerationUpdate{
				Status: vibe.RoomGenerationFailed,
				Error:  vibe.RoomGenerationYouTubeQuotaFailure,
			}
			payload, err := json.Marshal(update)
			if err != nil {
				return fmt.Errorf(
					"error marshaling youtube quota room generation update in Handle: %w",
					err,
				)
			}
			err = h.Cache.NotifyRoomUpdateV3(ctx, generation.RoomID, vibe.RoomEventV3{
				Type:    vibe.GenerationUpdate,
				Payload: payload,
			})
			if err != nil {
				return fmt.Errorf(
					"error notifying youtube quota room generation failure in Handle: %w",
					err,
				)
			}

			return internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf("error checking youtube search quota in Handle: %w", quotaError),
				},
			}
		}

		return fmt.Errorf("error searching generated playlist in Handle: %w", err)
	}
	searchUsages := make([]vibe.SearchUsage, 0, len(searchResult.SearchUsages))
	for _, usage := range searchResult.SearchUsages {
		usage.RoomID = generation.RoomID
		searchUsages = append(searchUsages, usage)
	}

	err = h.DB.CreateSearchUsages(ctx, searchUsages)
	if err != nil {
		log.Printf("error creating generated playlist search usage: %v", err)
	}
	err = h.Cache.CacheProviderSearches(ctx, vibe.SourceTypeYouTube, searchResult.CachedSearches, room.RoomType)
	if err != nil {
		log.Printf("error caching youtube searches for room generation: %v", err)
	}
	playlist = &searchResult.Playlist

	shouldStartPlayback := playbackState.CurrentPlaylistItem == nil
	for _, generatedItem := range *playlist {
		playlistItem := &vibe.PlaylistItem{
			ID:                  uuid.NewString(),
			RoomID:              room.ID,
			SourceType:          vibe.SourceTypeYouTube,
			SourceID:            generatedItem.YouTubeID,
			ProviderURL:         fmt.Sprintf("https://www.youtube.com/watch?v=%s", generatedItem.YouTubeID),
			PlaybackRestriction: generatedItem.PlaybackRestriction,
			Title:               generatedItem.Title,
			Publisher:           generatedItem.Publisher,
			ThumbnailURL:        generatedItem.ThumbnailURL,
			Duration:            generatedItem.Duration,
			AddedBySessionID:    room.HostID,
			AddedAt:             time.Now(),
		}

		addedPlaylistItem, err := h.DB.AddGeneratedPlaylistItem(ctx, playlistItem)
		if err != nil {
			return fmt.Errorf("error adding generated playlist item in Handle: %w", err)
		}
		if addedPlaylistItem.IsEmpty() {
			continue
		}

		playlistItemPayload, err := json.Marshal(addedPlaylistItem)
		if err != nil {
			return fmt.Errorf("error marshaling generated playlist item in Handle: %w", err)
		}

		err = h.Cache.NotifyRoomUpdateV3(ctx, room.ID, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemAdded,
			Payload: playlistItemPayload,
		})
		if err != nil {
			return fmt.Errorf("error notifying generated playlist item in Handle: %w", err)
		}

		if shouldStartPlayback {
			playbackState := &vibe.PlaybackStateV2{
				RoomID:              room.ID,
				CurrentPlaylistItem: addedPlaylistItem,
				IsPlaying:           true,
				PositionMs:          0,
				UpdatedAt:           time.Now(),
				ServerTimeMs:        int(time.Now().UnixMilli()),
			}
			err = h.DB.UpsertPlaybackStateV2(ctx, playbackState)
			if err != nil {
				return fmt.Errorf("error starting generated room playback in Handle: %w", err)
			}

			playbackPayload, err := json.Marshal(playbackState)
			if err != nil {
				return fmt.Errorf("error marshaling generated room playback in Handle: %w", err)
			}
			err = h.Cache.NotifyRoomUpdateV3(ctx, room.ID, vibe.RoomEventV3{
				Type:    vibe.PlaybackUpdate,
				Payload: playbackPayload,
			})
			if err != nil {
				return fmt.Errorf("error notifying generated room playback in Handle: %w", err)
			}
			shouldStartPlayback = false
		}
	}

	err = h.DB.CompleteRoomGeneration(ctx, generation.RoomID)
	if err != nil {
		return fmt.Errorf("error completing room generation in Handle: %w", err)
	}

	update := vibe.RoomGenerationUpdate{Status: vibe.RoomGenerationCompleted}
	payload, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("error marshaling completed room generation update in Handle: %w", err)
	}
	err = h.Cache.NotifyRoomUpdateV3(ctx, generation.RoomID, vibe.RoomEventV3{
		Type:    vibe.GenerationUpdate,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("error notifying completed room generation in Handle: %w", err)
	}

	return nil
}

type CleanupRoomGenerations struct {
	DB vibe.RoomGenerationCleaner
}

func (h *CleanupRoomGenerations) Handle(ctx context.Context, _ []byte) error {
	deleted, err := h.DB.DeleteExpiredRoomGenerations(
		ctx,
		vibe.RoomGenerationRetention,
	)
	if err != nil {
		return fmt.Errorf(
			"error deleting expired room generations in CleanupRoomGenerations.Handle: %w",
			err,
		)
	}

	if deleted > 0 {
		log.Printf("Cleaned up %d expired room generations", deleted)
	}

	return nil
}

const roomGenerationBusyRetryAfterSeconds = "60"

const generatedPlaylistPromptMaxLength = 300
