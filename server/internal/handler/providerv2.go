package handler

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetYouTubeItemV2 handles GET /api/v2/youtube/videos/{id}
//
//	@Summary	Get a YouTube item
//	@Tags		providers
//	@Produce	json
//	@Param		id	path		string	true	"Video ID"
//	@Success	200	{object}	vibe.ProviderItem
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/youtube/videos/{id} [get]
func GetYouTubeItemV2(
	ms vibe.ProviderItemFetcher,
	cache vibe.CachedProviderItemsCreator,
	provider string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		id := vars["id"]

		if id == "" {
			handleError(
				w,
				fmt.Errorf("error missing item id"),
				http.StatusBadRequest,
				true,
			)
			return
		}

		item, err := ms.GetProviderItem(ctx, id)
		if err != nil {
			var madeForKidsError internalerror.ErrMadeForKids
			if errors.As(err, &madeForKidsError) {
				handleError(w, client.ErrorCodeWrapper{
					Err: fmt.Errorf("error getting youtube item: %w", err),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "youtube_made_for_kids_not_supported",
						Message:   "This video is marked as made for kids on YouTube and cannot be added to Zoff. Try another version of the song.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				}, http.StatusBadRequest, false)
				return
			}

			var liveVideoError internalerror.ErrLiveVideo
			if errors.As(err, &liveVideoError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: liveVideoError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "youtube_live_video_not_supported",
							Message:   "Live videos cannot be added to rooms.",
							Propagate: true,
						},
						StatusCode: http.StatusBadRequest,
					},
					http.StatusBadRequest,
					false,
				)
				return
			}
			handleError(
				w,
				fmt.Errorf("error failed to get item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if provider == vibe.SourceTypeYouTube &&
			item.PlaybackRestriction == vibe.PlaybackRestrictionEmbedding {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error getting youtube item in GetYouTubeItemV2 handler: embedding is disabled"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "youtube_embedding_not_allowed",
						Message:   "This video cannot play outside YouTube. Try another version of the song.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		err = cache.CacheProviderItems(ctx, provider, []vibe.ProviderItem{*item})
		if err != nil {
			log.Printf("error caching provider item metadata: %v", err)
		}

		body, err := json.Marshal(item)
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

// GetSoundCloudItemV2 handles GET /api/v2/soundcloud/items/{id}
//
//	@Summary	Get a SoundCloud item
//	@Tags		providers
//	@Produce	json
//	@Param		id	path		string	true	"Track ID"
//	@Success	200	{object}	vibe.ProviderItem
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/soundcloud/items/{id} [get]
func GetSoundCloudItemV2(
	ms vibe.ProviderItemFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		id := vars["id"]

		if id == "" {
			handleError(
				w,
				fmt.Errorf("error missing item id"),
				http.StatusBadRequest,
				true,
			)
			return
		}

		item, err := ms.GetProviderItem(ctx, id)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error failed to get item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(item)
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

// ResolveSoundCloudItemV2 handles GET /api/v2/soundcloud/items
//
//	@Summary	Resolve a SoundCloud item URL
//	@Tags		providers
//	@Produce	json
//	@Param		url	query		string	true	"SoundCloud item URL"
//	@Success	200	{object}	vibe.ProviderItem
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/soundcloud/items [get]
func ResolveSoundCloudItemV2(
	resolver vibe.ProviderItemResolver,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		providerURL, err := vibe.ResolveSoundCloudTrackURL(
			r.URL.Query().Get("url"),
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error validating soundcloud URL in ResolveSoundCloudItemV2 handler: %w",
					err,
				),
				http.StatusBadRequest,
				true,
			)
			return
		}

		item, err := resolver.ResolveProviderItem(ctx, providerURL)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error resolving soundcloud item in ResolveSoundCloudItemV2 handler: %w",
					err,
				),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(item)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error marshaling response in ResolveSoundCloudItemV2 handler: %w",
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

// GetYouTubePlaylistV2 handles provider playlist lookups by ID.
//
//	@Summary	Get a provider playlist
//	@Tags		providers
//	@Produce	json
//	@Param		id	path		string	true	"Playlist ID"
//	@Success	200	{object}	vibe.ProviderPlaylist
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/youtube/playlists/{id} [get]
func GetYouTubePlaylistV2(
	fetcher vibe.ProviderPlaylistFetcher,
	cache vibe.CachedProviderItemsCreator,
	provider string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		id := vars["id"]
		validID := id != ""
		for _, character := range id {
			isLetter := character >= 'A' && character <= 'Z' ||
				character >= 'a' && character <= 'z'
			isNumber := character >= '0' && character <= '9'
			if !isLetter && !isNumber && character != '-' && character != '_' {
				validID = false
				break
			}
		}
		if !validID {
			handleError(
				w,
				fmt.Errorf("error invalid playlist id"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		playlist, err := fetcher.GetProviderPlaylist(ctx, id)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting provider playlist: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = cache.CacheProviderItems(ctx, provider, playlist.Items)
		if err != nil {
			log.Printf("error caching provider playlist item metadata: %v", err)
		}

		body, err := json.Marshal(playlist)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling provider playlist: %w", err),
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

// ResolveSoundCloudPlaylistV2 handles SoundCloud playlist URL lookups.
//
//	@Summary	Resolve a SoundCloud playlist URL
//	@Tags		providers
//	@Produce	json
//	@Param		url	query		string	true	"SoundCloud playlist URL"
//	@Success	200	{object}	vibe.ProviderPlaylist
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/soundcloud/playlists [get]
func ResolveSoundCloudPlaylistV2(
	resolver vibe.ProviderPlaylistResolver,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		providerURL, err := vibe.ResolveSoundCloudPlaylistURL(r.URL.Query().Get("url"))
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating soundcloud playlist URL: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		playlist, err := resolver.ResolveProviderPlaylist(ctx, providerURL)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error resolving soundcloud playlist: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(playlist)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling soundcloud playlist: %w", err),
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
