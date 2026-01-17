package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/vibe"
)

// SearchYouTube handles GET /api/v1/youtube/search
func SearchYouTube(
	yf vibe.YouTubeFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.URL.Query().Get("q")

		if query == "" {
			handleError(w, fmt.Errorf("missing query parameter 'q'"), http.StatusBadRequest, true)
			return
		}

		videos, err := yf.Search(ctx, query)
		if err != nil {
			handleError(w, fmt.Errorf("search failed: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(videos)
	}
}

// GetYouTubeVideo handles GET /api/v1/youtube/videos/:id
func GetYouTubeVideo(
	yf vibe.YouTubeFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		id := vars["id"]

		if id == "" {
			handleError(w, fmt.Errorf("missing video id"), http.StatusBadRequest, true)
			return
		}

		video, err := yf.GetVideo(ctx, id)
		if err != nil {
			handleError(w, fmt.Errorf("failed to get video: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(video)
	}
}
