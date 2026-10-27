package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
	"log"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"
)

// SearchYouTubeV2 handles GET /api/v2/rooms/{id}/search/youtube
//
//	@Summary	Search YouTube items
//	@Description	Requires an active room connection. Searches wait briefly for room presence to register.
//	@Tags		providers
//	@Produce	json
//	@Param		q	query		string	true	"Search query"
//	@Param		id	path	string	true	"Room ID"
//	@Success	200	{array}		vibe.ProviderItem
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Failure	503	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/search/youtube [get]
//	@Failure	403	{object}	vibe.ErrorResponse
func SearchYouTubeV2(
	ms vibe.ProviderItemsSearcher,
	cache vibe.ProviderSearchFetcherCreator,
	db vibe.RoomSearchUsageCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		vars := mux.Vars(r)
		roomID := vars["id"]
		if utf8.RuneCountInString(roomID) > 200 {
			handleError(
				w,
				fmt.Errorf("error validating room attribution: room ID is too long"),
				http.StatusBadRequest,
				true,
			)
			return
		}

		if utf8.RuneCountInString(query) < minimumSearchQueryLength || utf8.RuneCountInString(query) > 200 {
			handleError(
				w,
				fmt.Errorf(
					"error validating query in SearchYouTubeV2 handler: parameter 'q' must contain between %d and 200 characters",
					minimumSearchQueryLength,
				),
				http.StatusBadRequest,
				true,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error searching room: missing session"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching search room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error searching room: room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}
		if !room.RoomType.AllowsSource(vibe.SourceTypeYouTube) || !slices.Contains(room.Settings.EnabledSources, vibe.SourceTypeYouTube) {
			handleError(
				w,
				fmt.Errorf("error searching room: provider is disabled"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		active, err := db.WaitForActiveParticipant(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error checking search presence: %w", err),
				http.StatusServiceUnavailable,
				true,
			)
			return
		}
		if !active {
			handleError(
				w,
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error searching room: active presence required"),
					ResponseBody: internalerror.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "room_presence_required",
						Message:   "You need an active connection to this room to search. Reconnect and try again.",
						Propagate: true,
					},
					StatusCode: http.StatusForbidden,
				},
				http.StatusForbidden,
				false,
			)
			return
		}

		result, err := cache.GetOrClaimProviderSearch(ctx, vibe.SourceTypeYouTube, query, room.RoomType)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error reading youtube search cache: %w", err),
				http.StatusServiceUnavailable,
				true,
			)
			return
		}

		defer func() {
			releaseErr := cache.ReleaseSearch(ctx, result.Claim)
			if releaseErr != nil {
				log.Printf("error releasing youtube search claim: %v", releaseErr)
			}
		}()

		cachedSearches := result.Searches

		cacheHit := len(cachedSearches) > 0
		items := make([]vibe.ProviderItem, 0)
		usage, err := vibe.GenerateSearchUsage(vibe.SourceTypeYouTube, query, cacheHit)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error generating youtube search usage: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		usage.RoomID = roomID
		if !cacheHit {
			items, err = ms.SearchProviderItems(ctx, query, room.RoomType)
			if err != nil {
				var quotaError internalerror.ErrProviderQuotaExceeded
				if errors.As(err, &quotaError) {
					log.Printf("event=search_quota_exhausted provider=youtube purpose=interactive")
					handleError(
						w,
						internalerror.ErrorCodeWrapper{
							Err: quotaError,
							ResponseBody: internalerror.ErrorCodeResponseBody{
								Namespace: "vibes-backend",
								Error:     "youtube_search_quota_exhausted",
								Message:   vibe.RoomGenerationYouTubeQuotaFailure,
								Propagate: true,
							},
							StatusCode: http.StatusServiceUnavailable,
						},
						http.StatusServiceUnavailable,
						false,
					)
					return
				}
				handleError(
					w,
					fmt.Errorf("error searching youtube: %w", err),
					http.StatusServiceUnavailable,
					false,
				)
				return
			}

			search := vibe.CachedProviderSearch{Query: query, Items: items}
			err = cache.CacheProviderSearches(ctx, vibe.SourceTypeYouTube, []vibe.CachedProviderSearch{search}, room.RoomType)
			if err != nil {
				log.Printf("error caching youtube search: %v", err)
			}
		}
		if cacheHit {
			for _, item := range cachedSearches[0].GetProviderItems() {
				if item.AllowedInRoom(room.RoomType) {
					items = append(items, item)
				}
			}
		}

		err = db.CreateSearchUsages(ctx, []vibe.SearchUsage{*usage})
		if err != nil {
			log.Printf("error recording youtube search usage: %v", err)
		}

		err = cache.CacheProviderItems(ctx, vibe.SourceTypeYouTube, items)
		if err != nil {
			log.Printf("error caching youtube item metadata: %v", err)
		}

		if items == nil {
			items = make([]vibe.ProviderItem, 0)
		}

		body, err := json.Marshal(items)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling response in SearchYouTubeV2 handler: %w", err),
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

// SearchSoundCloudV2 handles GET /api/v2/rooms/{id}/search/soundcloud
//
//	@Summary	Search SoundCloud items
//	@Description	Requires an active room connection. Searches wait briefly for room presence to register.
//	@Tags		providers
//	@Produce	json
//	@Param		q	query		string	true	"Search query"
//	@Param		id	path	string	true	"Room ID"
//	@Success	200	{array}		vibe.ProviderItem
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/search/soundcloud [get]
//	@Failure	403	{object}	vibe.ErrorResponse
func SearchSoundCloudV2(
	ms vibe.ProviderItemsSearcher,
	cache vibe.ProviderSearchFetcherCreator,
	db vibe.RoomSearchUsageCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		vars := mux.Vars(r)
		roomID := vars["id"]
		if utf8.RuneCountInString(roomID) > 200 {
			handleError(
				w,
				fmt.Errorf("error validating room attribution: room ID is too long"),
				http.StatusBadRequest,
				true,
			)
			return
		}

		if utf8.RuneCountInString(query) < minimumSearchQueryLength {
			handleError(
				w,
				fmt.Errorf(
					"error validating query in SearchSoundCloudV2 handler: parameter 'q' must contain at least %d characters",
					minimumSearchQueryLength,
				),
				http.StatusBadRequest,
				true,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error searching room: missing session"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching search room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error searching room: room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}
		if !room.RoomType.AllowsSource(vibe.SourceTypeSoundCloud) || !slices.Contains(room.Settings.EnabledSources, vibe.SourceTypeSoundCloud) {
			handleError(
				w,
				fmt.Errorf("error searching room: provider is disabled"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		active, err := db.WaitForActiveParticipant(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error checking search presence: %w", err),
				http.StatusServiceUnavailable,
				true,
			)
			return
		}
		if !active {
			handleError(
				w,
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error searching room: active presence required"),
					ResponseBody: internalerror.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "room_presence_required",
						Message:   "You need an active connection to this room to search. Reconnect and try again.",
						Propagate: true,
					},
					StatusCode: http.StatusForbidden,
				},
				http.StatusForbidden,
				false,
			)
			return
		}

		result, err := cache.GetOrClaimProviderSearch(ctx, vibe.SourceTypeSoundCloud, query, room.RoomType)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error reading soundcloud search cache: %w", err),
				http.StatusServiceUnavailable,
				true,
			)
			return
		}

		defer func() {
			releaseErr := cache.ReleaseSearch(ctx, result.Claim)
			if releaseErr != nil {
				log.Printf("error releasing soundcloud search claim: %v", releaseErr)
			}
		}()

		cachedSearches := result.Searches

		items := make([]vibe.ProviderItem, 0)
		cacheHit := len(cachedSearches) > 0
		usage, err := vibe.GenerateSearchUsage(
			vibe.SourceTypeSoundCloud,
			query,
			cacheHit,
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error generating soundcloud search usage: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		usage.RoomID = roomID
		if !cacheHit {
			items, err = ms.SearchProviderItems(ctx, query, room.RoomType)
			if err != nil {
				var quotaError internalerror.ErrProviderQuotaExceeded
				if errors.As(err, &quotaError) {
					log.Printf("event=search_quota_exhausted provider=soundcloud purpose=interactive")
					handleError(
						w,
						internalerror.ErrorCodeWrapper{
							Err: quotaError,
							ResponseBody: internalerror.ErrorCodeResponseBody{
								Namespace: "vibes-backend",
								Error:     "soundcloud_search_quota_exhausted",
								Message:   "SoundCloud search is temporarily unavailable because its quota has been reached. Please try again later.",
								Propagate: true,
							},
							StatusCode: http.StatusServiceUnavailable,
						},
						http.StatusServiceUnavailable,
						false,
					)
					return
				}
				handleError(
					w,
					fmt.Errorf("error searching soundcloud: %w", err),
					http.StatusServiceUnavailable,
					false,
				)
				return
			}

			search := vibe.CachedProviderSearch{Query: query, Items: items}
			err = cache.CacheProviderSearches(ctx, vibe.SourceTypeSoundCloud, []vibe.CachedProviderSearch{search}, room.RoomType)
			if err != nil {
				log.Printf("error caching soundcloud search: %v", err)
			}
		}
		if cacheHit {
			for _, item := range cachedSearches[0].GetProviderItems() {
				if item.AllowedInRoom(room.RoomType) {
					items = append(items, item)
				}
			}
		}

		err = db.CreateSearchUsages(ctx, []vibe.SearchUsage{*usage})
		if err != nil {
			log.Printf("error recording soundcloud search usage: %v", err)
		}

		err = cache.CacheProviderItems(ctx, vibe.SourceTypeSoundCloud, items)
		if err != nil {
			log.Printf("error caching soundcloud item metadata: %v", err)
		}

		if items == nil {
			items = make([]vibe.ProviderItem, 0)
		}

		body, err := json.Marshal(items)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error marshaling response in SearchSoundCloudV2 handler: %w",
					err,
				),
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

const minimumSearchQueryLength = 3
