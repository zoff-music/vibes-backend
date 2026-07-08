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
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetSongs handles GET /api/v1/rooms/:id/songs
//
//	@Summary	List room songs
//	@Tags		songs
//	@Produce	json
//	@Param		id	path		string	true	"Room ID"
//	@Success	200	{array}		vibe.Song
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/songs [get]
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
				fmt.Errorf("error fetching songs: %w", err),
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
//
//	@Summary	Add a song
//	@Tags		songs
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Room ID"
//	@Param		request	body		vibe.AddSongRequest	true	"Song payload"
//	@Success	201		{object}	vibe.Song
//	@Failure	400		{object}	map[string]string
//	@Failure	401		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	404		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/songs [post]
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
				fmt.Errorf("error decoding request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

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

		room, err := db.GetRoom(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error room not found"),
				http.StatusNotFound,
				false,
			)
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
				fmt.Errorf("error adding song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		songs, err := db.GetSongs(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching songs: %w", err),
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
					fmt.Errorf("error auto-playing first song: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

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
//
//	@Summary	Remove a song
//	@Tags		songs
//	@Param		id		path	string	true	"Room ID"
//	@Param		songId	path	string	true	"Song ID"
//	@Success	204
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/songs/{songId} [delete]
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
			handleError(
				w,
				fmt.Errorf("unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		room, err := db.GetRoom(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}
		if !room.IsAdmin {
			handleError(
				w,
				fmt.Errorf("forbidden"),
				http.StatusForbidden,
				false,
			)
			return
		}

		err = db.RemoveSong(ctx, roomID, songID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error removing song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

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
//
//	@Summary	Vote for a song
//	@Tags		songs
//	@Param		id		path	string	true	"Room ID"
//	@Param		songId	path	string	true	"Song ID"
//	@Success	204
//	@Failure	401	{object}	map[string]string
//	@Failure	409	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/songs/{songId} [post]
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
				fmt.Errorf("error voting for song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

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
				fmt.Errorf("error marshaling songs payload: %w", err),
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
