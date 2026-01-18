package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/server/internal/helper"
	"github.com/zoff-music/vibes/vibe"
)

// GetSongs handles GET /api/v1/rooms/:id/songs
func GetSongs(
	db vibe.SongsFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		songs, err := db.GetSongs(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to fetch songs: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(songs)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("marshal response: %w", err),
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

// AddSong handles the addition of a new song to the room
func AddSong(
	db vibe.SongController,
	ips vibe.RoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.AddSongRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("invalid request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		// Get the max position to append new song at end of queue
		maxPosition, err := db.GetMaxPosition(ctx, roomID)
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

		created, err := db.AddSong(ctx, song)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to add song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		// Broadcast queue update
		songs, err := db.GetSongs(ctx, roomID)
		if err == nil {
			_ = ips.NotifyRoom(ctx, roomID, &vibe.RoomEvent{
				Type:    vibe.EventTypeQueueReordered,
				Payload: songs,
			})
		}

		// Auto-play if this is the first song
		if created.Position == 1 {
			playbackState := &vibe.PlaybackState{
				RoomID:        roomID,
				CurrentSongID: &created.ID,
				CurrentSong:   created,
				IsPlaying:     true,
				PositionMs:    0,
				UpdatedAt:     time.Now(),
				ServerTimeMs:  time.Now().UnixMilli(),
			}

			if err := db.UpsertPlaybackState(ctx, playbackState); err != nil {
				// Log error but don't fail the request completely
				fmt.Printf("failed to auto-play first song: %v\n", err)
			} else {
				// Notify room about playback update
				_ = ips.NotifyRoom(ctx, roomID, &vibe.RoomEvent{
					Type:    vibe.EventTypePlaybackUpdate,
					Payload: playbackState,
				})
			}
		}

		body, err := json.Marshal(created)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("marshal response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	}
}

// RemoveSong handles the removal of a song from the room
func RemoveSong(
	db vibe.SongsModifier,
	ips vibe.RoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		songID := vars["songId"]

		err := db.RemoveSong(ctx, roomID, songID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to remove song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		// Broadcast queue update
		songs, err := db.GetSongs(ctx, roomID)
		if err == nil {
			_ = ips.NotifyRoom(ctx, roomID, &vibe.RoomEvent{
				Type:    vibe.EventTypeQueueReordered,
				Payload: songs,
			})
		}

		w.WriteHeader(http.StatusOK)
	}
}

// ReorderSongs handles the reordering of songs in the room
func ReorderSongs(
	db vibe.SongsModifier,
	ips vibe.RoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		songID := vars["songId"]

		var req vibe.ReorderSongsRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("invalid request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		err = db.ReorderSongs(ctx, roomID, songID, req.NewPosition)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to reorder songs: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		// Broadcast queue update
		songs, err := db.GetSongs(ctx, roomID)
		if err == nil {
			_ = ips.NotifyRoom(ctx, roomID, &vibe.RoomEvent{
				Type:    vibe.EventTypeQueueReordered,
				Payload: songs,
			})
		}

		w.WriteHeader(http.StatusOK)
	}
}

// VoteSong handles voting for a song
func VoteSong(
	db vibe.SongsModifier,
	ips vibe.RoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		songID := vars["songId"]

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}
		userID := session.UserID

		err := db.VoteSong(ctx, roomID, songID, userID)
		if err != nil {
			if errors.Is(err, vibe.ErrAlreadyVoted) {
				handleError(
					w,
					fmt.Errorf("already voted"),
					http.StatusConflict,
					false,
				)
				return
			}
			handleError(
				w,
				fmt.Errorf("failed to vote for song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		// Broadcast queue update
		songs, err := db.GetSongs(ctx, roomID)
		if err == nil {
			_ = ips.NotifyRoom(ctx, roomID, &vibe.RoomEvent{
				Type:    vibe.EventTypeQueueReordered,
				Payload: songs,
			})
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
