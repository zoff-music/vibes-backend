package handler

import (
	"context"
	"encoding/json/v2"
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
//	@Failure	500	{object}	vibe.ErrorResponse
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

// AddSong handles the addition of a new song to the room
//
//	@Summary	Add a song
//	@Tags		songs
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Room ID"
//	@Param		request	body		vibe.AddSongRequest	true	"Song payload"
//	@Success	200		{object}	vibe.AddSongResult
//	@Success	201		{object}	vibe.AddSongResult
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	403		{object}	vibe.ErrorResponse
//	@Failure	404		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v1/rooms/{id}/songs [post]
func AddSong(
	db vibe.SongQueueAdder,
	events vibe.CachedMusicTrackRoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.AddSongRequest
		err := json.UnmarshalRead(r.Body, &req)
		if err != nil {
			handleError(
				w,
				vibe.PublicError{
					Err:        fmt.Errorf("error decoding request body: %w", err),
					Kind:       vibe.PublicSongRequestInvalid,
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				true,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				vibe.PublicError{
					Err:        fmt.Errorf("error unauthorized"),
					Kind:       vibe.PublicSongSessionRequired,
					StatusCode: http.StatusUnauthorized,
				},
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
				vibe.PublicError{
					Err:        fmt.Errorf("error room not found"),
					Kind:       vibe.PublicSongRoomNotFound,
					StatusCode: http.StatusNotFound,
				},
				http.StatusNotFound,
				false,
			)
			return
		}

		if room.Settings.OnlyAdminAddSongs && !room.IsAdmin {
			handleError(
				w,
				vibe.PublicError{
					Err:        fmt.Errorf("error only admins can add songs in this room"),
					Kind:       vibe.PublicSongRoomAdminRequired,
					StatusCode: http.StatusForbidden,
				},
				http.StatusForbidden,
				false,
			)
			return
		}

		sourceEnabled := false
		for _, source := range room.Settings.EnabledSources {
			if req.SourceType == source {
				sourceEnabled = true
				break
			}
		}

		if !sourceEnabled {
			handleError(
				w,
				vibe.PublicError{
					Err:        fmt.Errorf("error source type %s is not enabled for this room", req.SourceType),
					Kind:       vibe.PublicSongProviderDisabled,
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		if vibe.IsLiveVideo(req.SourceType, req.Duration) {
			handleError(
				w,
				vibe.PublicError{
					Err:        fmt.Errorf("error live videos cannot be added to rooms"),
					Kind:       vibe.PublicYouTubeLiveVideoNotSupported,
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		providerURL, err := req.CanonicalProviderURL()
		if err != nil {
			handleError(
				w,
				vibe.PublicError{
					Err:        fmt.Errorf("error getting canonical provider URL: %w", err),
					Kind:       vibe.PublicSongProviderURLInvalid,
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		artist := req.Artist
		playbackRestriction := ""
		cachedTrack, err := events.GetCachedMusicTrack(ctx, req.SourceType, req.SourceID)
		if err != nil {
			log.Printf("error getting cached provider track metadata: %v", err)
		}

		if err == nil && !cachedTrack.IsEmpty() {
			if vibe.IsLiveVideo(req.SourceType, cachedTrack.DurationSeconds) {
				handleError(
					w,
					vibe.PublicError{
						Err:        fmt.Errorf("error live videos cannot be added to rooms"),
						Kind:       vibe.PublicYouTubeLiveVideoNotSupported,
						StatusCode: http.StatusBadRequest,
					},
					http.StatusBadRequest,
					false,
				)
				return
			}

			playbackRestriction = cachedTrack.PlaybackRestriction
		}

		song := &vibe.Song{
			ID:                  uuid.New().String(),
			RoomID:              roomID,
			SourceType:          req.SourceType,
			SourceID:            req.SourceID,
			ProviderURL:         providerURL,
			PlaybackRestriction: playbackRestriction,
			Title:               req.Title,
			Artist:              artist,
			ThumbnailURL:        req.Thumbnail,
			Duration:            req.Duration,
			AddedBySessionID:    session.UserID,
			AddedAt:             time.Now(),
		}

		result, err := db.AddSong(ctx, song)
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

		var v2Event *vibe.RoomEventV2Payload
		if result.Outcome == vibe.AddSongOutcomeAdded || result.Outcome == vibe.AddSongOutcomeDuplicateVoted {
			for position, song := range songs {
				if song.ID != result.Song.ID {
					continue
				}

				v2Payload, marshalErr := json.Marshal(vibe.SongPositionUpdate{
					Song:     song,
					Position: position,
				})
				if marshalErr != nil {
					handleError(
						w,
						fmt.Errorf("error marshaling compact positioned song event in add song: %w", marshalErr),
						http.StatusInternalServerError,
						true,
					)
					return
				}

				v2Event = &vibe.RoomEventV2Payload{Type: vibe.SongUpdated, Payload: v2Payload}
				break
			}

			if v2Event == nil {
				v2Payload, marshalErr := json.Marshal(vibe.SongIDUpdate{ID: result.Song.ID})
				if marshalErr != nil {
					handleError(
						w,
						fmt.Errorf("error marshaling compact song removal fallback in add song: %w", marshalErr),
						http.StatusInternalServerError,
						true,
					)
					return
				}

				v2Event = &vibe.RoomEventV2Payload{Type: vibe.SongRemoved, Payload: v2Payload}
			}
		}

		err = events.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
			Type:    vibe.QueueReordered,
			Payload: songsPayload,
			Origin:  session.EventOrigin,
			V2:      v2Event,
		})
		if err != nil {
			log.Printf("failed to notify room: %v", err)
		}

		if result.Outcome == vibe.AddSongOutcomeAdded && len(songs) == 1 {
			playbackState := &vibe.PlaybackState{
				RoomID:       roomID,
				CurrentSong:  &result.Song,
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

			err = events.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
				Type:    vibe.PlaybackUpdate,
				Payload: playbackPayload,
				Origin:  session.EventOrigin,
			})
			if err != nil {
				log.Printf("failed to notify room: %v", err)
			}

		}

		body, err := json.Marshal(result)
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
		status := http.StatusOK
		if result.Outcome == vibe.AddSongOutcomeAdded {
			status = http.StatusCreated
		}

		w.WriteHeader(status)
		_, _ = w.Write(body)

		// Activity is best-effort after the song and playback changes succeed.
		if result.Outcome == vibe.AddSongOutcomeDuplicateAlreadyVoted {
			return
		}

		kind := vibe.MessageKindAdded
		if result.Outcome == vibe.AddSongOutcomeDuplicateVoted {
			kind = vibe.MessageKindVoted
		}

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			log.Printf("error fetching AddSong chat author: %v", err)
			return
		}

		message := vibe.RoomMessage{
			ID:        uuid.NewString(),
			UserID:    session.UserID,
			Name:      profile.Name,
			IsAdmin:   room.IsAdmin,
			Kind:      kind,
			Text:      result.Song.Title,
			CreatedAt: time.Now().UnixMilli(),
		}

		chatPayload, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling AddSong chat activity: %v", err)
			return
		}

		err = events.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
			Type:    vibe.MessageEvent,
			Payload: chatPayload,
		})
		if err != nil {
			log.Printf("error publishing AddSong chat activity: %v", err)
		}
	}
}

// RemoveSong handles the removal of a song from the room
//
//	@Summary	Remove a song
//	@Tags		songs
//	@Param		id		path	string	true	"Room ID"
//	@Param		songId	path	string	true	"Song ID"
//	@Success	204
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/rooms/{id}/songs/{songId} [delete]
func RemoveSong(
	db vibe.SongQueueRemover,
	notifier vibe.RoomEventNotifier,
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
				fmt.Errorf("error unauthorized"),
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
				fmt.Errorf("error forbidden"),
				http.StatusForbidden,
				false,
			)
			return
		}

		removedSong, err := db.GetSong(ctx, roomID, songID)
		if err != nil {
			handleError(w, fmt.Errorf("error finding song to remove: %w", err), http.StatusInternalServerError, true)
			return
		}

		if removedSong.IsEmpty() {
			handleError(w, fmt.Errorf("error finding song to remove"), http.StatusNotFound, false)
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

		v2Payload, err := json.Marshal(vibe.SongIDUpdate{ID: songID})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling compact remove song event: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		v2Event := &vibe.RoomEventV2Payload{Type: vibe.SongRemoved, Payload: v2Payload}

		err = notifier.NotifyRoomUpdate(ctx, roomID, vibe.RoomEvent{
			Type:    vibe.QueueReordered,
			Payload: songsPayload,
			Origin:  session.EventOrigin,
			V2:      v2Event,
		})
		if err != nil {
			log.Printf("failed to notify room in remove song: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			log.Printf("error fetching RemoveSong chat author: %v", err)
			return
		}

		message := vibe.RoomMessage{
			ID:        uuid.NewString(),
			UserID:    session.UserID,
			Name:      profile.Name,
			IsAdmin:   room.IsAdmin,
			Kind:      vibe.MessageKindDeleted,
			Text:      removedSong.Title,
			CreatedAt: time.Now().UnixMilli(),
		}

		chatPayload, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling RemoveSong chat activity: %v", err)
			return
		}

		err = notifier.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
			Type:    vibe.MessageEvent,
			Payload: chatPayload,
		})
		if err != nil {
			log.Printf("error publishing RemoveSong chat activity: %v", err)
		}
	}
}

// VoteSong handles voting for a song
//
//	@Summary	Vote for a song
//	@Tags		songs
//	@Param		id		path	string	true	"Room ID"
//	@Param		songId	path	string	true	"Song ID"
//	@Success	204
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	409	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/rooms/{id}/songs/{songId} [post]
func VoteSong(
	db vibe.SongQueueVoter,
	notifier vibe.RoomEventNotifier,
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
				fmt.Errorf("error unauthorized"),
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
					vibe.PublicError{
						Err:        alreadyVotedError,
						Kind:       vibe.PublicSongVoteAlreadyExists,
						StatusCode: http.StatusConflict,
					},
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

		var v2Event *vibe.RoomEventV2Payload
		votedTitle := "song"
		for position, song := range songs {
			if song.ID != songID {
				continue
			}

			v2Payload, marshalErr := json.Marshal(vibe.SongPositionUpdate{
				Song:     song,
				Position: position,
			})
			if marshalErr != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling compact vote song event: %w", marshalErr),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			v2Event = &vibe.RoomEventV2Payload{Type: vibe.SongUpdated, Payload: v2Payload}
			votedTitle = song.Title
			break
		}

		if v2Event == nil {
			v2Payload, marshalErr := json.Marshal(vibe.SongIDUpdate{ID: songID})
			if marshalErr != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling compact vote song removal fallback: %w", marshalErr),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			v2Event = &vibe.RoomEventV2Payload{Type: vibe.SongRemoved, Payload: v2Payload}
		}

		err = notifier.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
			Type:    vibe.QueueReordered,
			Payload: songsPayload,
			Origin:  session.EventOrigin,
			V2:      v2Event,
		})
		if err != nil {
			log.Printf("failed to notify room in vote song: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)

		chatRoom, err := db.GetRoom(ctx, roomID, userID)
		if err != nil {
			log.Printf("error fetching VoteSong chat room: %v", err)
			return
		}

		if chatRoom.IsEmpty() {
			return
		}

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			log.Printf("error fetching VoteSong chat author: %v", err)
			return
		}

		message := vibe.RoomMessage{
			ID:        uuid.NewString(),
			UserID:    session.UserID,
			Name:      profile.Name,
			IsAdmin:   chatRoom.IsAdmin,
			Kind:      vibe.MessageKindVoted,
			Text:      votedTitle,
			CreatedAt: time.Now().UnixMilli(),
		}

		chatPayload, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling VoteSong chat activity: %v", err)
			return
		}

		err = notifier.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
			Type:    vibe.MessageEvent,
			Payload: chatPayload,
		})
		if err != nil {
			log.Printf("error publishing VoteSong chat activity: %v", err)
		}
	}
}
