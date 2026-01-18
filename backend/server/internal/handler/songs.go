package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/vibe"
)

// GetSongs handles GET /api/v1/rooms/:id/songs
func GetSongs(
	sf vibe.SongsFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		songs, err := sf.GetSongs(ctx, roomID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to fetch songs: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(songs)
	}
}

// AddSong handles POST /api/v1/rooms/:id/songs
func AddSong(
	sm vibe.SongsModifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.AddSongRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest, true)
			return
		}

		// Get the max position to append new song at end of queue
		maxPosition, err := sm.GetMaxPosition(ctx, roomID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to get max position: %w", err), http.StatusInternalServerError, true)
			return
		}

		artist := req.Artist
		song := &vibe.Song{
			ID:           uuid.New().String(),
			RoomID:       roomID,
			SourceType:   req.SourceType,
			SourceID:     req.SourceID,
			Title:        req.Title,
			Artist:       &artist,
			ThumbnailURL: req.Thumbnail,
			Duration:     req.Duration,
			AddedBy:      req.AddedBy,
			AddedAt:      time.Now(),
			Position:     maxPosition + 1,
		}

		created, err := sm.AddSong(ctx, song)
		if err != nil {
			handleError(w, fmt.Errorf("failed to add song: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created)
	}
}

// RemoveSong handles DELETE /api/v1/rooms/:id/songs/:songId
func RemoveSong(
	sm vibe.SongsModifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		songID := vars["songId"]

		err := sm.RemoveSong(ctx, roomID, songID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to remove song: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ReorderSongs handles PATCH /api/v1/rooms/:id/songs/reorder/:songId
func ReorderSongs(
	sm vibe.SongsModifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		songID := vars["songId"]

		var req vibe.ReorderSongsRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest, true)
			return
		}

		err = sm.ReorderSongs(ctx, roomID, songID, req.NewPosition)
		if err != nil {
			handleError(w, fmt.Errorf("failed to reorder songs: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
