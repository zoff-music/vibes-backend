package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/internalerror"
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
	ips vibe.RoomEventAdminNotifier,
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

		// Get the user from the session
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

		// Get room to check enabled sources
		room, err := db.GetRoom(ctx, roomID, session.UserID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to fetch room: %w", err), http.StatusInternalServerError, true)
			return
		}

		if room.IsEmpty() {
			handleError(w, fmt.Errorf("room not found"), http.StatusNotFound, false)
			return
		}

		if room.Settings.OnlyAdminAddSongs && !room.IsAdmin {
			handleError(
				w,
				fmt.Errorf("only admins can add songs in this room"),
				http.StatusForbidden,
				false,
			)
			return
		}

		// Validate source type
		sourceEnabled := false
		for _, source := range room.Settings.EnabledSources {
			if string(req.SourceType) == source {
				sourceEnabled = true
				break
			}
		}

		if !sourceEnabled {
			handleError(
				w,
				fmt.Errorf("source type %s is not enabled for this room", req.SourceType),
				http.StatusBadRequest,
				false,
			)
			return
		}

		artist := req.Artist
		song := &vibe.Song{
			ID:           uuid.New().String(),
			RoomID:       roomID,
			SourceType:   req.SourceType,
			SourceID:     req.SourceID,
			Title:        req.Title,
			Artist:       artist,
			ThumbnailURL: req.Thumbnail,
			Duration:     req.Duration,
			AddedBy:      session.UserID,
			AddedAt:      time.Now(),
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
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to fetch songs: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		songsPayload, err := json.Marshal(songs)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling songs payload in add song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = ips.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
			Type:    vibe.QueueReordered,
			Payload: songsPayload,
		})
		if err != nil {
			log.Printf("failed to notify room: %v", err)
		}

		// Auto-play if this is the first song in the queue
		allSongs, err := db.GetSongs(ctx, roomID)
		if err == nil && len(allSongs) == 1 {
			playbackState := &vibe.PlaybackState{
				RoomID:       roomID,
				CurrentSong:  created,
				IsPlaying:    true,
				PositionMs:   0,
				UpdatedAt:    time.Now(),
				ServerTimeMs: int(time.Now().UnixMilli()),
			}

			err := db.UpsertPlaybackState(ctx, playbackState)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("failed to auto-play first song: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			// Notify room about playback update
			playbackPayload, err := json.Marshal(playbackState)
			if err != nil {
				handleError(w,
					fmt.Errorf("error marshaling playback payload in add song: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			err = ips.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
				Type:    vibe.PlaybackUpdate,
				Payload: playbackPayload,
			})
			if err != nil {
				log.Printf("failed to notify room: %v", err)
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
	db vibe.SongController,
	ips vibe.RoomEventAdminNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		songID := vars["songId"]

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized, false)
			return
		}

		room, err := db.GetRoom(ctx, roomID, session.UserID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to fetch room: %w", err), http.StatusInternalServerError, true)
			return
		}
		if room.IsEmpty() {
			handleError(w, fmt.Errorf("room not found"), http.StatusNotFound, false)
			return
		}
		if !room.IsAdmin {
			handleError(w, fmt.Errorf("forbidden"), http.StatusForbidden, false)
			return
		}

		err = db.RemoveSong(ctx, roomID, songID)
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
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetch songs in remove song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		songsPayload, err := json.Marshal(songs)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling songs payload in remove song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = ips.NotifyRoomUpdate(ctx, roomID, vibe.RoomEvent{
			Type:    vibe.QueueReordered,
			Payload: songsPayload,
		})
		if err != nil {
			log.Printf("failed to notify room in remove song: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// VoteSong handles voting for a song
func VoteSong(
	db vibe.SongController,
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
			var alreadyVotedError internalerror.ErrAlreadyVoted
			if errors.As(err, &alreadyVotedError) {
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
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetch songs in vote song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		songsPayload, err := json.Marshal(songs)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to marshal songs payload: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = ips.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
			Type:    vibe.QueueReordered,
			Payload: songsPayload,
		})
		if err != nil {
			log.Printf("failed to notify room in vote song: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
