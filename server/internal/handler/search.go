package handler

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

// SearchMusic handles GET /api/v1/youtube/search
//
//	@Summary	Search YouTube items
//	@Tags		providers
//	@Produce	json
//	@Param		q	query		string	true	"Search query"
//	@Param		roomId	query	string	false	"Room attribution"
//	@Success	200	{array}		vibe.MusicTrack
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Failure	503	{object}	vibe.ErrorResponse
//	@Router		/api/v1/youtube/search [get]
//
// Deprecated: Use GET /api/v2/rooms/{id}/search/youtube. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use GET /api/v2/rooms/{id}/search/youtube for the playlist-item contract. This endpoint retains its existing payloads.
func SearchMusic(
	ms vibe.ProviderItemsSearcher,
	cache vibe.ProviderSearchCache,
	usageCreator vibe.SearchUsageCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		roomID := strings.TrimSpace(r.URL.Query().Get("roomId"))
		if utf8.RuneCountInString(roomID) > 200 {
			handleError(w, fmt.Errorf("error validating room attribution: room ID is too long"), http.StatusBadRequest, true)
			return
		}

		if utf8.RuneCountInString(query) < minimumSearchQueryLength {
			handleError(
				w,
				fmt.Errorf(
					"error validating query in SearchMusic handler: parameter 'q' must contain at least %d characters",
					minimumSearchQueryLength,
				),
				http.StatusBadRequest,
				true,
			)
			return
		}

		cachedSearches, err := cache.GetCachedProviderSearches(
			ctx,
			vibe.SourceTypeYouTube,
			[]string{query},
		)
		if err != nil {
			log.Printf("error getting cached youtube search: %v", err)
			cachedSearches = []vibe.CachedProviderSearch{}
		}

		items := make([]vibe.ProviderItem, 0)
		cacheHit := len(cachedSearches) > 0
		if cacheHit {
			cachedItems := cachedSearches[0].GetProviderItems()
			for _, item := range cachedItems {
				if vibe.IsLiveVideo(item.Source, item.DurationSeconds) ||
					item.PlaybackRestriction == vibe.PlaybackRestrictionEmbedding {
					continue
				}
				items = append(items, item)
			}
		}
		usage := vibe.GenerateSearchUsage(
			vibe.SourceTypeYouTube,
			query,
			cacheHit,
		)
		usage.RoomID = roomID

		err = usageCreator.CreateSearchUsages(ctx, []vibe.SearchUsage{usage})
		if err != nil {
			log.Printf("error creating youtube search usage: %v", err)
		}
		if !cacheHit {
			var quotaReset time.Time
			quotaReset, err = cache.GetProviderQuotaReset(
				ctx,
				vibe.SourceTypeYouTube,
				vibe.ProviderQuotaOperationSearch,
			)
			quotaCheckSucceeded := err == nil
			if !quotaCheckSucceeded {
				log.Printf("error getting cached youtube quota reset: %v", err)
			}
			quotaExceeded := time.Now().Before(quotaReset)
			if quotaCheckSucceeded && quotaExceeded {
				err = internalerror.ErrProviderQuotaExceeded{
					Err: fmt.Errorf(
						"error checking youtube search quota in SearchMusic handler: cached until %s",
						quotaReset.Format(time.RFC3339),
					),
					Provider: vibe.SourceTypeYouTube,
					ResetAt:  quotaReset,
				}
			}
			if !quotaCheckSucceeded || !quotaExceeded {
				items, err = ms.SearchProviderItems(ctx, query)
			}
		}
		if err != nil {
			var quotaError internalerror.ErrProviderQuotaExceeded
			if errors.As(err, &quotaError) {
				err = cache.CacheProviderQuotaReset(
					ctx,
					vibe.SourceTypeYouTube,
					vibe.ProviderQuotaOperationSearch,
					quotaError.ResetAt,
				)
				if err != nil {
					log.Printf("error caching youtube quota reset: %v", err)
				}

				w.Header().Set("Retry-After", quotaError.ResetAt.UTC().Format(http.TimeFormat))

				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: quotaError,
						ResponseBody: client.ErrorCodeResponseBody{
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
				fmt.Errorf("error searching music in SearchMusic handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if !cacheHit {
			search := vibe.CachedProviderSearch{Query: query, Items: items}
			err = cache.CacheProviderSearches(
				ctx,
				vibe.SourceTypeYouTube,
				[]vibe.CachedProviderSearch{
					search,
				},
			)
			if err != nil {
				log.Printf("error caching youtube search: %v", err)
			}
		}

		err = cache.CacheProviderItems(ctx, vibe.SourceTypeYouTube, items)
		if err != nil {
			log.Printf("error caching youtube item metadata: %v", err)
		}

		legacy := make([]vibe.MusicTrack, len(items))
		for index, item := range items {
			legacy[index] = *item.ToMusicTrack()
		}

		body, err := json.Marshal(legacy)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling response in SearchMusic handler: %w", err),
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

// SearchSoundCloud handles GET /api/v1/soundcloud/search
//
//	@Summary	Search SoundCloud items
//	@Tags		providers
//	@Produce	json
//	@Param		q	query		string	true	"Search query"
//	@Param		roomId	query	string	false	"Room attribution"
//	@Success	200	{array}		vibe.MusicTrack
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/soundcloud/search [get]
//
// Deprecated: Use GET /api/v2/rooms/{id}/search/soundcloud. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use GET /api/v2/rooms/{id}/search/soundcloud for the playlist-item contract. This endpoint retains its existing payloads.
func SearchSoundCloud(
	ms vibe.ProviderItemsSearcher,
	cache vibe.CachedProviderSearchItemFetcherCreator,
	usageCreator vibe.SearchUsageCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		roomID := strings.TrimSpace(r.URL.Query().Get("roomId"))
		if utf8.RuneCountInString(roomID) > 200 {
			handleError(w, fmt.Errorf("error validating room attribution: room ID is too long"), http.StatusBadRequest, true)
			return
		}

		if utf8.RuneCountInString(query) < minimumSearchQueryLength {
			handleError(
				w,
				fmt.Errorf(
					"error validating query in SearchSoundCloud handler: parameter 'q' must contain at least %d characters",
					minimumSearchQueryLength,
				),
				http.StatusBadRequest,
				true,
			)
			return
		}

		cachedSearches, err := cache.GetCachedProviderSearches(
			ctx,
			vibe.SourceTypeSoundCloud,
			[]string{query},
		)
		if err != nil {
			log.Printf("error getting cached soundcloud search: %v", err)
			cachedSearches = []vibe.CachedProviderSearch{}
		}

		items := make([]vibe.ProviderItem, 0)
		cacheHit := len(cachedSearches) > 0
		if cacheHit {
			items = cachedSearches[0].GetProviderItems()
		}
		usage := vibe.GenerateSearchUsage(
			vibe.SourceTypeSoundCloud,
			query,
			cacheHit,
		)
		usage.RoomID = roomID

		err = usageCreator.CreateSearchUsages(ctx, []vibe.SearchUsage{usage})
		if err != nil {
			log.Printf("error creating soundcloud search usage: %v", err)
		}
		if !cacheHit {
			items, err = ms.SearchProviderItems(ctx, query)
		}
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error searching music in SearchSoundCloud handler: %w",
					err,
				),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if !cacheHit {
			search := vibe.CachedProviderSearch{Query: query, Items: items}
			err = cache.CacheProviderSearches(
				ctx,
				vibe.SourceTypeSoundCloud,
				[]vibe.CachedProviderSearch{
					search,
				},
			)
			if err != nil {
				log.Printf("error caching soundcloud search: %v", err)
			}
		}

		err = cache.CacheProviderItems(ctx, vibe.SourceTypeSoundCloud, items)
		if err != nil {
			log.Printf("error caching soundcloud item metadata: %v", err)
		}

		legacy := make([]vibe.MusicTrack, len(items))
		for index, item := range items {
			legacy[index] = *item.ToMusicTrack()
		}

		body, err := json.Marshal(legacy)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error marshaling response in SearchSoundCloud handler: %w",
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
