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
	"github.com/zoff-music/vibes-backend/client"
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
//
// Deprecated: Use GET /api/v2/rooms/{id}/playlist-items. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use GET /api/v2/rooms/{id}/playlist-items for the playlist-item contract. This endpoint retains its existing payloads.
func GetSongs(
	db vibe.PlaylistItemsFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		items, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching songs: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		songs := make([]vibe.Song, len(items))
		for index, item := range items {
			songs[index] = *item.ToSong()
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
//
// Deprecated: Use POST /api/v2/rooms/{id}/playlist-items. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use POST /api/v2/rooms/{id}/playlist-items for the playlist-item contract. This endpoint retains its existing payloads.
func AddSong(
	db vibe.PlaylistItemQueueAdder,
	events vibe.ProviderItemRoomNotifier,
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
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error decoding request body: %w", err),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_request_invalid",
						Message:   "The song details could not be read. Search for the song again and retry.",
						Propagate: true,
					},
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
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error unauthorized"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_session_required",
						Message:   "Your room session is missing. Rejoin the room and try adding the song again.",
						Propagate: true,
					},
					StatusCode: http.StatusUnauthorized,
				},
				http.StatusUnauthorized,
				false,
			)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
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
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error room not found"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_room_not_found",
						Message:   "This room no longer exists. Join another room to add songs.",
						Propagate: true,
					},
					StatusCode: http.StatusNotFound,
				},
				http.StatusNotFound,
				false,
			)
			return
		}

		if room.Settings.OnlyAdminAddPlaylistItems && !room.IsAdmin {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error only admins can add songs in this room"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_room_admin_required",
						Message:   "Only room admins can add songs here. Log in as a room admin in room settings, or ask an admin to allow everyone to add songs.",
						Propagate: true,
					},
					StatusCode: http.StatusForbidden,
				},
				http.StatusForbidden,
				false,
			)
			return
		}

		sourceEnabled := false
		for _, source := range room.Settings.EnabledSources {
			if req.SourceType == source && room.RoomType.AllowsSource(source) {
				sourceEnabled = true
				break
			}
		}

		if !sourceEnabled {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error source type %s is not enabled for this room", req.SourceType),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_provider_disabled",
						Message:   "This music provider is not enabled for this room. Choose another provider or ask a room admin to enable it.",
						Propagate: true,
					},
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
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error live videos cannot be added to rooms"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "youtube_live_video_not_supported",
						Message:   "Live videos cannot be added to rooms.",
						Propagate: true,
					},
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
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error getting canonical provider URL: %w", err),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_provider_url_invalid",
						Message:   "The song link is not valid for this provider. Search for the song again or paste a valid track link.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		artist := req.Artist
		playbackRestriction := ""
		cachedItem, err := events.GetCachedProviderItem(ctx, req.SourceType, req.SourceID)
		if err != nil {
			log.Printf("error getting cached provider track metadata: %v", err)
		}

		if req.SourceType == vibe.SourceTypeYouTube && (err != nil || cachedItem.IsEmpty()) {
			handleError(w, client.ErrorCodeWrapper{
				Err: fmt.Errorf("error adding youtube track: verified metadata is unavailable"),
				ResponseBody: client.ErrorCodeResponseBody{
					Namespace: "vibes-backend",
					Error:     "youtube_track_verification_required",
					Message:   "This video's availability needs to be checked again. Search for it or paste its YouTube link again before adding it.",
					Propagate: true,
				},
				StatusCode: http.StatusBadRequest,
			}, http.StatusBadRequest, false)
			return
		}

		if err == nil && !cachedItem.IsEmpty() {
			if !cachedItem.AllowedInRoom(room.RoomType) {
				handleError(w, client.ErrorCodeWrapper{
					Err: fmt.Errorf("error adding item: provider metadata is not allowed for %s", room.RoomType),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "playlist_item_not_allowed",
						Message:   "This item cannot be played in this room. Music rooms accept music videos; all videos must allow embedded playback and must not be live.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				}, http.StatusBadRequest, false)
				return
			}

			if vibe.IsLiveVideo(req.SourceType, cachedItem.DurationSeconds) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: fmt.Errorf("error live videos cannot be added to rooms"),
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "youtube_live_video_not_supported",
							Message:   "Live videos cannot be added to rooms.",
							Propagate: true,
						},
						StatusCode: http.StatusBadRequest,
					},
					http.StatusBadRequest,
					false,
				)
				return
			}

			playbackRestriction = cachedItem.PlaybackRestriction
			if req.SourceType == vibe.SourceTypeYouTube {
				req.Title = cachedItem.Title
				artist = cachedItem.Publisher
				req.Thumbnail = cachedItem.ThumbnailURL
				req.Duration = cachedItem.DurationSeconds
			}
		}

		item := &vibe.PlaylistItem{
			ID:                  uuid.New().String(),
			RoomID:              roomID,
			SourceType:          req.SourceType,
			SourceID:            req.SourceID,
			ProviderURL:         providerURL,
			PlaybackRestriction: playbackRestriction,
			Title:               req.Title,
			Publisher:           artist,
			ThumbnailURL:        req.Thumbnail,
			Duration:            req.Duration,
			AddedBySessionID:    session.UserID,
			AddedAt:             time.Now(),
		}

		result, err := db.AddPlaylistItem(ctx, item)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error adding song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		items, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching songs: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		songs := make([]vibe.Song, len(items))
		for index, item := range items {
			songs[index] = *item.ToSong()
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
				if song.ID != result.PlaylistItem.ID {
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
				v2Payload, marshalErr := json.Marshal(vibe.SongIDUpdate{ID: result.PlaylistItem.ID})
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
			playbackState := &vibe.PlaybackStateV2{
				RoomID:              roomID,
				CurrentPlaylistItem: &result.PlaylistItem,
				IsPlaying:           true,
				PositionMs:          0,
				UpdatedAt:           time.Now(),
				ServerTimeMs:        int(time.Now().UnixMilli()),
			}

			err := db.UpsertPlaybackStateV2(ctx, playbackState)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error auto-playing first song: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			legacyPlayback := playbackState.ToPlaybackState()

			playbackPayload, err := json.Marshal(legacyPlayback)
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

		legacy := result.ToAddSongResult()

		body, err := json.Marshal(legacy)
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
			IsHost:    room.Mode == vibe.RoomModeHost && room.HostID == session.UserID,
			Text:      result.PlaylistItem.Title,
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
//
// Deprecated: Use DELETE /api/v2/rooms/{id}/playlist-items/{playlistItemId}. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use DELETE /api/v2/rooms/{id}/playlist-items/{playlistItemId} for the playlist-item contract. This endpoint retains its existing payloads.
func RemoveSong(
	db vibe.PlaylistItemQueueRemover,
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

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
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

		removedSong, err := db.GetPlaylistItem(ctx, roomID, songID)
		if err != nil {
			handleError(w, fmt.Errorf("error finding song to remove: %w", err), http.StatusInternalServerError, true)
			return
		}

		if removedSong.IsEmpty() {
			handleError(w, fmt.Errorf("error finding song to remove"), http.StatusNotFound, false)
			return
		}

		err = db.RemovePlaylistItem(ctx, roomID, songID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error removing song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		items, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetch songs in remove song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		songs := make([]vibe.Song, len(items))
		for index, item := range items {
			songs[index] = *item.ToSong()
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
			IsHost:    room.Mode == vibe.RoomModeHost && room.HostID == session.UserID,
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
//
// Deprecated: Use POST /api/v2/rooms/{id}/playlist-items/{playlistItemId}. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use POST /api/v2/rooms/{id}/playlist-items/{playlistItemId} for the playlist-item contract. This endpoint retains its existing payloads.
func VoteSong(
	db vibe.PlaylistItemQueueVoter,
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

		err := db.VotePlaylistItem(ctx, roomID, songID, userID)
		if err != nil {
			var alreadyVotedError internalerror.ErrAlreadyVoted
			if errors.As(err, &alreadyVotedError) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: alreadyVotedError,
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "song_vote_already_exists",
							Message:   "Your vote is already counted for this song.",
							Propagate: true,
						},
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

		items, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetch songs in vote song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		songs := make([]vibe.Song, len(items))
		for index, item := range items {
			songs[index] = *item.ToSong()
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

		chatRoom, err := db.GetRoomV2(ctx, roomID, userID)
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
			IsHost:    chatRoom.Mode == vibe.RoomModeHost && chatRoom.HostID == session.UserID,
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
