package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/vibe"
)

// SearchMusic handles GET /api/v1/search (originally youtube/search)
func SearchMusic(
	ms vibe.MusicSearcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.URL.Query().Get("q")

		if query == "" {
			handleError(w, fmt.Errorf("missing query parameter 'q'"), http.StatusBadRequest, true)
			return
		}

		tracks, err := ms.Search(ctx, query)
		if err != nil {
			handleError(w, fmt.Errorf("search failed: %w", err), http.StatusInternalServerError, true)
			return
		}

		body, err := json.Marshal(tracks)
		if err != nil {
			handleError(w, fmt.Errorf("search music: marshal response: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// GetMusicTrack handles GET /api/v1/tracks/:id
func GetMusicTrack(
	ms vibe.MusicSearcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		id := vars["id"]

		if id == "" {
			handleError(w, fmt.Errorf("missing track id"), http.StatusBadRequest, true)
			return
		}

		track, err := ms.GetTrack(ctx, id)
		if err != nil {
			handleError(w, fmt.Errorf("failed to get track: %w", err), http.StatusInternalServerError, true)
			return
		}

		body, err := json.Marshal(track)
		if err != nil {
			handleError(w, fmt.Errorf("get music track: marshal response: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
