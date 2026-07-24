package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.URL.Query().Get("q")

		if query == "" {
			handleError(
				w,
				fmt.Errorf("error missing query parameter 'q'"),
				http.StatusBadRequest,
				true,
			)
			return
		}

		tracks, err := ms.Search(ctx, query)
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
				fmt.Errorf("error search failed: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(tracks)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
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
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.URL.Query().Get("q")

		if query == "" {
			handleError(
				w,
				fmt.Errorf("error missing query parameter 'q'"),
				http.StatusBadRequest,
				true,
			)
			return
		}

		tracks, err := ms.Search(ctx, query)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error search failed: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(tracks)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
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
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.URL.Query().Get("q")

		if query == "" {
			handleError(
				w,
				fmt.Errorf("error missing query parameter 'q'"),
				http.StatusBadRequest,
				true,
			)
			return
		}

		tracks, err := ms.Search(ctx, query)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error search failed: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(tracks)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
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
