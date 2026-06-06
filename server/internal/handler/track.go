package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetMusicTrack handles GET /api/v1/youtube/videos/{id}
func GetMusicTrack(
	ms vibe.MusicSearcher,
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

// GetSoundCloudTrack handles GET /api/v1/soundcloud/tracks/{id}
func GetSoundCloudTrack(
	ms vibe.MusicSearcher,
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

// GetSpotifyTrack handles GET /api/v1/spotify/tracks/{id}
func GetSpotifyTrack(
	ms vibe.MusicSearcher,
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
