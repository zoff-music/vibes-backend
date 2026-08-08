package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

// SearchMusic handles GET /api/v1/youtube/search
//
//	@Summary	Search YouTube tracks
//	@Tags		providers
//	@Produce	json
//	@Param		q	query		string	true	"Search query"
//	@Success	200	{array}		vibe.MusicTrack
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Failure	503	{object}	map[string]string
//	@Router		/api/v1/youtube/search [get]
func SearchMusic(
	ms vibe.MusicSearcher,
	cache vibe.MusicSearchCache,
	usageCreator vibe.SearchUsageCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))

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

		cachedSearches, err := cache.GetCachedSearches(
			ctx,
			vibe.SourceTypeYouTube,
			[]string{query},
		)
		if err != nil {
			log.Printf("error getting cached youtube search: %v", err)
			cachedSearches = []vibe.CachedSearch{}
		}

		tracks := make([]vibe.MusicTrack, 0)
		cacheHit := len(cachedSearches) > 0
		if cacheHit {
			tracks = cachedSearches[0].GetMusicTracks()
		}
		usage := vibe.GenerateSearchUsage(
			vibe.SourceTypeYouTube,
			query,
			cacheHit,
		)
		err = usageCreator.CreateSearchUsages(ctx, []vibe.SearchUsage{usage})
		if err != nil {
			log.Printf("error creating youtube search usage: %v", err)
		}
		if !cacheHit {
			tracks, err = ms.Search(ctx, query)
		}
		if err != nil {
			var quotaError internalerror.ErrProviderQuotaExceeded
			if errors.As(err, &quotaError) {
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
			search := vibe.GenerateCachedSearch(query, tracks)
			err = cache.CacheSearches(
				ctx,
				vibe.SourceTypeYouTube,
				[]vibe.CachedSearch{
					search,
				},
			)
			if err != nil {
				log.Printf("error caching youtube search: %v", err)
			}
		}

		err = cache.CacheMusicTracks(ctx, vibe.SourceTypeYouTube, tracks)
		if err != nil {
			log.Printf("error caching youtube track metadata: %v", err)
		}

		body, err := json.Marshal(tracks)
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
//	@Summary	Search SoundCloud tracks
//	@Tags		providers
//	@Produce	json
//	@Param		q	query		string	true	"Search query"
//	@Success	200	{array}		vibe.MusicTrack
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/soundcloud/search [get]
func SearchSoundCloud(
	ms vibe.MusicSearcher,
	cache vibe.MusicSearchCache,
	usageCreator vibe.SearchUsageCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))

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

		cachedSearches, err := cache.GetCachedSearches(
			ctx,
			vibe.SourceTypeSoundCloud,
			[]string{query},
		)
		if err != nil {
			log.Printf("error getting cached soundcloud search: %v", err)
			cachedSearches = []vibe.CachedSearch{}
		}

		tracks := make([]vibe.MusicTrack, 0)
		cacheHit := len(cachedSearches) > 0
		if cacheHit {
			tracks = cachedSearches[0].GetMusicTracks()
		}
		usage := vibe.GenerateSearchUsage(
			vibe.SourceTypeSoundCloud,
			query,
			cacheHit,
		)
		err = usageCreator.CreateSearchUsages(ctx, []vibe.SearchUsage{usage})
		if err != nil {
			log.Printf("error creating soundcloud search usage: %v", err)
		}
		if !cacheHit {
			tracks, err = ms.Search(ctx, query)
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
			search := vibe.GenerateCachedSearch(query, tracks)
			err = cache.CacheSearches(
				ctx,
				vibe.SourceTypeSoundCloud,
				[]vibe.CachedSearch{
					search,
				},
			)
			if err != nil {
				log.Printf("error caching soundcloud search: %v", err)
			}
		}

		err = cache.CacheMusicTracks(ctx, vibe.SourceTypeSoundCloud, tracks)
		if err != nil {
			log.Printf("error caching soundcloud track metadata: %v", err)
		}

		body, err := json.Marshal(tracks)
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

// SearchSpotify handles GET /api/v1/spotify/search
//
//	@Summary	Search Spotify tracks
//	@Tags		providers
//	@Produce	json
//	@Param		q	query		string	true	"Search query"
//	@Success	200	{array}		vibe.MusicTrack
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/spotify/search [get]
func SearchSpotify(
	ms vibe.MusicSearcher,
	cache vibe.MusicSearchCache,
	usageCreator vibe.SearchUsageCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))

		if utf8.RuneCountInString(query) < minimumSearchQueryLength {
			handleError(
				w,
				fmt.Errorf(
					"error validating query in SearchSpotify handler: parameter 'q' must contain at least %d characters",
					minimumSearchQueryLength,
				),
				http.StatusBadRequest,
				true,
			)
			return
		}

		cachedSearches, err := cache.GetCachedSearches(
			ctx,
			vibe.SourceTypeSpotify,
			[]string{query},
		)
		if err != nil {
			log.Printf("error getting cached spotify search: %v", err)
			cachedSearches = []vibe.CachedSearch{}
		}

		tracks := make([]vibe.MusicTrack, 0)
		cacheHit := len(cachedSearches) > 0
		if cacheHit {
			tracks = cachedSearches[0].GetMusicTracks()
		}
		usage := vibe.GenerateSearchUsage(
			vibe.SourceTypeSpotify,
			query,
			cacheHit,
		)
		err = usageCreator.CreateSearchUsages(ctx, []vibe.SearchUsage{usage})
		if err != nil {
			log.Printf("error creating spotify search usage: %v", err)
		}
		if !cacheHit {
			tracks, err = ms.Search(ctx, query)
		}
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error searching music in SearchSpotify handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if !cacheHit {
			search := vibe.GenerateCachedSearch(query, tracks)
			err = cache.CacheSearches(
				ctx,
				vibe.SourceTypeSpotify,
				[]vibe.CachedSearch{
					search,
				},
			)
			if err != nil {
				log.Printf("error caching spotify search: %v", err)
			}
		}

		err = cache.CacheMusicTracks(ctx, vibe.SourceTypeSpotify, tracks)
		if err != nil {
			log.Printf("error caching spotify track metadata: %v", err)
		}

		body, err := json.Marshal(tracks)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling response in SearchSpotify handler: %w", err),
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
