package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

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
