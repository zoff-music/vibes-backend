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

// GetMusicTrack handles GET /api/v1/youtube/videos/{id}
//
//	@Summary	Get a YouTube track
//	@Tags		providers
//	@Produce	json
//	@Param		id	path		string	true	"Video ID"
//	@Success	200	{object}	vibe.MusicTrack
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/youtube/videos/{id} [get]
func GetMusicTrack(
	ms vibe.MusicTrackFetcher,
	cache vibe.CachedMusicTrackCreator,
	provider string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		id := vars["id"]

		if id == "" {
			handleError(
				w,
				fmt.Errorf("error missing track id"),
				http.StatusBadRequest,
				true,
			)
			return
		}

		track, err := ms.GetTrack(ctx, id)
		if err != nil {
			var liveVideoError internalerror.ErrLiveVideo
			if errors.As(err, &liveVideoError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: liveVideoError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "youtube_live_video_not_supported",
							Message:   liveVideoErrorMessage,
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
				fmt.Errorf("error failed to get track: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = cache.CacheMusicTracks(ctx, provider, []vibe.MusicTrack{*track})
		if err != nil {
			log.Printf("error caching provider track metadata: %v", err)
		}

		body, err := json.Marshal(track)
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

// GetSoundCloudTrack handles GET /api/v1/soundcloud/tracks/{id}
//
//	@Summary	Get a SoundCloud track
//	@Tags		providers
//	@Produce	json
//	@Param		id	path		string	true	"Track ID"
//	@Success	200	{object}	vibe.MusicTrack
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/soundcloud/tracks/{id} [get]
func GetSoundCloudTrack(
	ms vibe.MusicTrackFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		id := vars["id"]

		if id == "" {
			handleError(
				w,
				fmt.Errorf("error missing track id"),
				http.StatusBadRequest,
				true,
			)
			return
		}

		track, err := ms.GetTrack(ctx, id)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error failed to get track: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(track)
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

// ResolveSoundCloudTrack handles GET /api/v1/soundcloud/tracks
//
//	@Summary	Resolve a SoundCloud track URL
//	@Tags		providers
//	@Produce	json
//	@Param		url	query		string	true	"SoundCloud track URL"
//	@Success	200	{object}	vibe.MusicTrack
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/soundcloud/tracks [get]
func ResolveSoundCloudTrack(
	resolver vibe.MusicTrackResolver,
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
					"error validating soundcloud URL in ResolveSoundCloudTrack handler: %w",
					err,
				),
				http.StatusBadRequest,
				true,
			)
			return
		}

		track, err := resolver.ResolveTrack(ctx, providerURL)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error resolving soundcloud track in ResolveSoundCloudTrack handler: %w",
					err,
				),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(track)
		if err != nil {
			handleError(
				w,
				fmt.Errorf(
					"error marshaling response in ResolveSoundCloudTrack handler: %w",
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

const liveVideoErrorMessage = "Live videos cannot be added to rooms."
