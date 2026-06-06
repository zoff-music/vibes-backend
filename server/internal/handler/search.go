package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zoff-music/vibes-backend/vibe"
)

// SearchMusic handles GET /api/v1/youtube/search
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
