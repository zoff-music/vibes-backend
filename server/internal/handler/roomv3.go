package handler

import (
	"context"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// AddItemV3 handles the addition of a new playlistItem to the room
//
//	@Summary	Add a playlist item
//	@Tags		playlist-items
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Room ID"
//	@Param		request	body		vibe.AddPlaylistItemRequest	true	"Playlist item payload"
//	@Success	200		{object}	vibe.AddPlaylistItemResult
//	@Success	201		{object}	vibe.AddPlaylistItemResult
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	403		{object}	vibe.ErrorResponse
//	@Failure	404		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v3/rooms/{id}/items [post]
func AddItemV3(
	db vibe.PlaylistItemQueueAdder,
	events vibe.CachedProviderItemRoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.AddPlaylistItemRequest
		err := jsonv2.UnmarshalRead(r.Body, &req)
		if err != nil {
			handleError(
				w,
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error decoding request body: %w", err),
					ResponseBody: internalerror.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_request_invalid",
						Message:   "The playlist item details could not be read. Search for the playlist item again and retry.",
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
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error unauthorized"),
					ResponseBody: internalerror.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_session_required",
						Message:   "Your room session is missing. Rejoin the room and try adding the playlist item again.",
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
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error room not found"),
					ResponseBody: internalerror.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_room_not_found",
						Message:   "This room no longer exists. Join another room to add playlist items.",
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
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error only admins can add playlist items in this room"),
					ResponseBody: internalerror.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_room_admin_required",
						Message:   "Only room admins can add playlist items here. Log in as a room admin in room settings, or ask an admin to allow everyone to add playlist items.",
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
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error source type %s is not enabled for this room", req.SourceType),
					ResponseBody: internalerror.ErrorCodeResponseBody{
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
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error live videos cannot be added to rooms"),
					ResponseBody: internalerror.ErrorCodeResponseBody{
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
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error getting canonical provider URL: %w", err),
					ResponseBody: internalerror.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_provider_url_invalid",
						Message:   "The playlist item link is not valid for this provider. Search for the playlist item again or paste a valid track link.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		publisher := req.Publisher
		playbackRestriction := ""
		cachedItem, err := events.GetCachedProviderItem(ctx, req.SourceType, req.SourceID)
		if err != nil {
			log.Printf("error getting cached provider track metadata: %v", err)
		}

		if err != nil || cachedItem.IsEmpty() {
			verificationError := "soundcloud_track_verification_required"
			if req.SourceType == vibe.SourceTypeYouTube {
				verificationError = "youtube_track_verification_required"
			}

			handleError(
				w,
				internalerror.ErrorCodeWrapper{
					Err: fmt.Errorf("error adding track: verified metadata is unavailable"),
					ResponseBody: internalerror.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     verificationError,
						Message:   "This track's availability needs to be checked again. Search for it or paste its link again before adding it.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				},
				http.StatusBadRequest,
				false,
			)
			return
		}

		if err == nil && !cachedItem.IsEmpty() {
			if !cachedItem.AllowedInRoom(room.RoomType) {
				message := "This item cannot be played in this room. Music rooms accept music videos; all videos must allow embedded playback and must not be live."
				if req.SourceType == vibe.SourceTypeSoundCloud {
					message = "SoundCloud does not offer full embedded playback for this track. Preview-only and restricted tracks cannot be added. Try another version."
				}

				handleError(
					w,
					internalerror.ErrorCodeWrapper{
						Err: fmt.Errorf("error adding item: provider metadata is not allowed for %s", room.RoomType),
						ResponseBody: internalerror.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "playlist_item_not_allowed",
							Message:   message,
							Propagate: true,
						},
						StatusCode: http.StatusBadRequest,
					},
					http.StatusBadRequest,
					false,
				)
				return
			}

			if vibe.IsLiveVideo(req.SourceType, cachedItem.DurationSeconds) {
				handleError(
					w,
					internalerror.ErrorCodeWrapper{
						Err: fmt.Errorf("error live videos cannot be added to rooms"),
						ResponseBody: internalerror.ErrorCodeResponseBody{
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
			req.Title = cachedItem.Title
			publisher = cachedItem.Publisher
			req.Thumbnail = cachedItem.ThumbnailURL
			req.Duration = cachedItem.DurationSeconds
			providerURL = cachedItem.ProviderURL
		}

		playlistItem := &vibe.PlaylistItem{
			ID:                  uuid.New().String(),
			RoomID:              roomID,
			SourceType:          req.SourceType,
			SourceID:            req.SourceID,
			ProviderURL:         providerURL,
			PlaybackRestriction: playbackRestriction,
			Title:               req.Title,
			Publisher:           publisher,
			ThumbnailURL:        req.Thumbnail,
			Duration:            req.Duration,
			AddedBySessionID:    session.UserID,
			AddedAt:             time.Now(),
		}

		result, err := db.AddPlaylistItem(ctx, playlistItem)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error adding playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItems, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching playlist items: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItemsPayload, err := json.Marshal(playlistItems)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling playlist items payload in add playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		var compactEvent *vibe.RoomEventV3Payload
		if result.Outcome == vibe.AddPlaylistItemOutcomeAdded || result.Outcome == vibe.AddPlaylistItemOutcomeDuplicateVoted {
			for position, playlistItem := range playlistItems {
				if playlistItem.ID != result.PlaylistItem.ID {
					continue
				}

				compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemPositionUpdate{
					PlaylistItem: playlistItem,
					Position:     position,
				})
				if marshalErr != nil {
					handleError(
						w,
						fmt.Errorf("error marshaling compact positioned playlist item event in add playlist item: %w", marshalErr),
						http.StatusInternalServerError,
						true,
					)
					return
				}

				compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemUpdated, Payload: compactPayload}
				break
			}

			if compactEvent == nil {
				compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemIDUpdate{ID: result.PlaylistItem.ID})
				if marshalErr != nil {
					handleError(
						w,
						fmt.Errorf("error marshaling compact playlist item removal fallback in add playlist item: %w", marshalErr),
						http.StatusInternalServerError,
						true,
					)
					return
				}

				compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemRemoved, Payload: compactPayload}
			}
		}

		err = events.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: playlistItemsPayload,
			Origin:  session.EventOrigin,
			Compact: compactEvent,
		})
		if err != nil {
			log.Printf("failed to notify room: %v", err)
		}

		if result.Outcome == vibe.AddPlaylistItemOutcomeAdded && len(playlistItems) == 1 {
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
					fmt.Errorf("error auto-playing first playlist item: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			playbackPayload, err := json.Marshal(playbackState)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling playback payload in add playlist item: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			err = events.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
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
		if result.Outcome == vibe.AddPlaylistItemOutcomeAdded {
			status = http.StatusCreated
		}

		w.WriteHeader(status)
		_, _ = w.Write(body)

		// Activity is best-effort after the playlistItem and playback changes succeed.
		if result.Outcome == vibe.AddPlaylistItemOutcomeDuplicateAlreadyVoted {
			return
		}

		kind := vibe.MessageKindAdded
		if result.Outcome == vibe.AddPlaylistItemOutcomeDuplicateVoted {
			kind = vibe.MessageKindVoted
		}

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			log.Printf("error fetching AddItemV3 chat author: %v", err)
			return
		}

		_, isModerator := helper.GetAdminUserFromContext(ctx)

		message := vibe.RoomMessage{
			IsModerator: isModerator,
			ID:          uuid.NewString(),
			UserID:      session.UserID,
			Name:        profile.Name,
			IsAdmin:     room.IsAdmin,
			Kind:        kind,
			Text:        result.PlaylistItem.Title,
			IsHost:      room.Mode == vibe.RoomModeHost && room.HostID == session.UserID,
			CreatedAt:   time.Now().UnixMilli(),
		}

		chatPayload, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling AddItemV3 chat activity: %v", err)
			return
		}

		err = events.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
			Type:    vibe.MessageEvent,
			Payload: chatPayload,
		})
		if err != nil {
			log.Printf("error publishing AddItemV3 chat activity: %v", err)
		}
	}
}

// RemoveItemV3 handles the removal of a playlist item from the room
//
//	@Summary	Remove a playlist item
//	@Tags		playlist-items
//	@Param		id		path	string	true	"Room ID"
//	@Param		itemId	path	string	true	"Playlist item ID"
//	@Success	204
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v3/rooms/{id}/items/{itemId} [delete]
func RemoveItemV3(
	db vibe.PlaylistItemQueueRemover,
	notifier vibe.RoomEventV3Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		playlistItemID := vars["itemId"]

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

		removedPlaylistItem, err := db.GetPlaylistItem(ctx, roomID, playlistItemID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error finding playlist item to remove: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if removedPlaylistItem.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error finding playlist item to remove"),
				http.StatusNotFound,
				false,
			)
			return
		}

		err = db.RemovePlaylistItem(ctx, roomID, playlistItemID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error removing playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItems, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetch playlist items in remove playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItemsPayload, err := json.Marshal(playlistItems)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling playlist items payload in remove playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		compactPayload, err := json.Marshal(vibe.PlaylistItemIDUpdate{ID: playlistItemID})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling compact remove playlist item event: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		compactEvent := &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemRemoved, Payload: compactPayload}

		err = notifier.NotifyRoomUpdateV3(ctx, roomID, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: playlistItemsPayload,
			Origin:  session.EventOrigin,
			Compact: compactEvent,
		})
		if err != nil {
			log.Printf("failed to notify room in remove playlist item: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			log.Printf("error fetching RemoveItemV3 chat author: %v", err)
			return
		}

		_, isModerator := helper.GetAdminUserFromContext(ctx)

		message := vibe.RoomMessage{
			IsModerator: isModerator,
			ID:          uuid.NewString(),
			UserID:      session.UserID,
			Name:        profile.Name,
			IsAdmin:     room.IsAdmin,
			Kind:        vibe.MessageKindDeleted,
			Text:        removedPlaylistItem.Title,
			IsHost:      room.Mode == vibe.RoomModeHost && room.HostID == session.UserID,
			CreatedAt:   time.Now().UnixMilli(),
		}

		chatPayload, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling RemoveItemV3 chat activity: %v", err)
			return
		}

		err = notifier.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
			Type:    vibe.MessageEvent,
			Payload: chatPayload,
		})
		if err != nil {
			log.Printf("error publishing RemoveItemV3 chat activity: %v", err)
		}
	}
}

// VoteItemV3 handles voting for a playlist item
//
//	@Summary	Vote for a playlist item
//	@Tags		playlist-items
//	@Param		id		path	string	true	"Room ID"
//	@Param		itemId	path	string	true	"Playlist item ID"
//	@Success	204
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	409	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v3/rooms/{id}/items/{itemId} [post]
func VoteItemV3(
	db vibe.PlaylistItemQueueVoter,
	notifier vibe.RoomEventV3Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		playlistItemID := vars["itemId"]

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

		err := db.VotePlaylistItem(ctx, roomID, playlistItemID, userID)
		if err != nil {
			var alreadyVotedError internalerror.ErrAlreadyVoted
			if errors.As(err, &alreadyVotedError) {
				handleError(
					w,
					internalerror.ErrorCodeWrapper{
						Err: alreadyVotedError,
						ResponseBody: internalerror.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "song_vote_already_exists",
							Message:   "Your vote is already counted for this playlist item.",
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
				fmt.Errorf("error voting for playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItems, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetch playlist items in vote playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItemsPayload, err := json.Marshal(playlistItems)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling playlist items payload: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		var compactEvent *vibe.RoomEventV3Payload
		votedTitle := "playlist item"
		for position, playlistItem := range playlistItems {
			if playlistItem.ID != playlistItemID {
				continue
			}

			compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemPositionUpdate{
				PlaylistItem: playlistItem,
				Position:     position,
			})
			if marshalErr != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling compact vote playlist item event: %w", marshalErr),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemUpdated, Payload: compactPayload}
			votedTitle = playlistItem.Title
			break
		}

		if compactEvent == nil {
			compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemIDUpdate{ID: playlistItemID})
			if marshalErr != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling compact vote playlist item removal fallback: %w", marshalErr),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemRemoved, Payload: compactPayload}
		}

		err = notifier.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: playlistItemsPayload,
			Origin:  session.EventOrigin,
			Compact: compactEvent,
		})
		if err != nil {
			log.Printf("failed to notify room in vote playlist item: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)

		chatRoom, err := db.GetRoomV2(ctx, roomID, userID)
		if err != nil {
			log.Printf("error fetching VoteItemV3 chat room: %v", err)
			return
		}

		if chatRoom.IsEmpty() {
			return
		}

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			log.Printf("error fetching VoteItemV3 chat author: %v", err)
			return
		}

		_, isModerator := helper.GetAdminUserFromContext(ctx)

		message := vibe.RoomMessage{
			IsModerator: isModerator,
			ID:          uuid.NewString(),
			UserID:      session.UserID,
			Name:        profile.Name,
			IsAdmin:     chatRoom.IsAdmin,
			IsHost:      chatRoom.Mode == vibe.RoomModeHost && chatRoom.HostID == session.UserID,
			Kind:        vibe.MessageKindVoted,
			Text:        votedTitle,
			CreatedAt:   time.Now().UnixMilli(),
		}

		chatPayload, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling VoteItemV3 chat activity: %v", err)
			return
		}

		err = notifier.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
			Type:    vibe.MessageEvent,
			Payload: chatPayload,
		})
		if err != nil {
			log.Printf("error publishing VoteItemV3 chat activity: %v", err)
		}
	}
}

// GetItemsV3 returns room items with optional row-number pagination.
//
// @Summary List room items
// @Description With neither from nor to, returns the full list. Bounds are inclusive and one-based, and must be supplied together. reqFrom/reqTo preserve requested bounds (zero when omitted); from/to identify returned rows (zero for an empty page). total counts matching items before pagination; count is the size of data.
// @Tags playlist-items
// @Produce json
// @Param id path string true "Room ID"
// @Param from query int false "First row, inclusive; requires to" minimum(1)
// @Param to query int false "Last row, inclusive; requires from; maximum 100 items per page" minimum(1)
// @Param sourceType query string false "Filter by provider" Enums(youtube,soundcloud)
// @Param excludeCurrent query bool false "Exclude the currently playing item from data and total" default(false)
// @Success 200 {object} vibe.ItemResultV3
// @Failure 400 {object} vibe.ErrorResponse
// @Failure 500 {object} vibe.ErrorResponse
// @Router /api/v3/rooms/{id}/items [get]
func GetItemsV3(db vibe.ItemsV3Fetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["id"]
		query := r.URL.Query()
		search := vibe.ItemSearchV3{
			SourceType: query.Get("sourceType"),
		}

		if search.SourceType != "" && search.SourceType != vibe.SourceTypeYouTube && search.SourceType != vibe.SourceTypeSoundCloud {
			handleError(
				w,
				fmt.Errorf("error invalid item provider"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		if query.Has("from") || query.Has("to") {
			from, err := strconv.Atoi(query.Get("from"))
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error parsing pagination from: %w", err),
					http.StatusBadRequest,
					false,
				)
				return
			}

			to, err := strconv.Atoi(query.Get("to"))
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error parsing pagination to: %w", err),
					http.StatusBadRequest,
					false,
				)
				return
			}

			if from < 1 || to < from || to-from >= 100 {
				handleError(
					w,
					fmt.Errorf("error invalid pagination: from and to must be positive inclusive row numbers spanning at most 100 items"),
					http.StatusBadRequest,
					false,
				)
				return
			}

			search.From = from
			search.To = to
		}

		if query.Has("excludeCurrent") {
			excludeCurrent, err := strconv.ParseBool(query.Get("excludeCurrent"))
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error parsing excludeCurrent: %w", err),
					http.StatusBadRequest,
					false,
				)
				return
			}

			search.ExcludeCurrent = excludeCurrent
		}

		result, err := db.GetItemsV3(ctx, roomID, search)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room items: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(result)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling room items: %w", err),
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

// RoomEventsV3 handles GET /api/v3/rooms/:id/events (SSE).
//
//	@Summary	Subscribe to compact room events
//
// @Description Returns an SSE stream, not a single JSON response. Frames contain an optional `id`, an `event` name, and a JSON `data` payload, followed by a blank line.
// @Description `connected` carries {"time": 1700000000000}; `settings_update` carries a RoomV2; `playlist_items_snapshot` carries a PlaylistItem array; `playback_update` carries a PlaybackStateV2. Compact queue updates use `playlist_item_added`, `playlist_item_updated`, and `playlist_item_removed`.
// @Description Reconnect with Last-Event-ID or lastEventId. See the [event contract](https://github.com/zoff-music/vibes-backend/blob/main/docs/flows/sessions.md) for event payloads and replay.
//
//	@Tags		rooms
//	@Produce	text/event-stream
//
// @Param lastEventId query string false "Last received stream cursor; used when Last-Event-ID is absent"
// @Param Last-Event-ID header string false "Last received stream cursor"
//
//	@Param		id	path	string	true	"Room ID"
//	@Success	200	{string}	string "SSE frames with event-specific JSON data; see the stream description"
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v3/rooms/{id}/events [get]
func RoomEventsV3(
	events vibe.RoomEventV3ReplayNotifier,
	state vibe.RoomEventV3StateFetcherUpdater,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		// Get UserID from session/context.
		userID := ""
		isCastReceiver := false
		isRemoteController := false
		castOwnerID := ""
		isNewSession := false
		session, ok := helper.GetSessionFromContext(ctx)
		if ok {
			userID = session.UserID
			isCastReceiver = session.AuthType == "cast"
			isRemoteController = session.AuthType == "remote"
			isNewSession = session.IsNew
			if isCastReceiver {
				castOwnerID = session.UserID
			}
		}

		if isCastReceiver {
			// Use a per-connection ID to avoid colliding with the real user session.
			// The underlying user identity is castOwnerID.
			if castOwnerID != "" {
				userID = fmt.Sprintf("cast:%s:%s", roomID, castOwnerID)
			}
		}

		lastUsersCount := -1

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		topicName := fmt.Sprintf("room:%s", roomID)
		lastEventID := r.Header.Get("Last-Event-ID")
		if lastEventID == "" {
			lastEventID = r.URL.Query().Get("lastEventId")
		}
		replay, err := events.PrepareReplay(ctx, topicName, lastEventID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error preparing room event replay: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		cursorID := ""
		if replay.AfterID != "" {
			milliseconds, sequence, found := strings.Cut(replay.AfterID, "-")
			if !found {
				handleError(
					w,
					fmt.Errorf("error parsing replay cursor in RoomEventsV3: missing sequence"),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			cursorTime, err := strconv.ParseUint(milliseconds, 10, 64)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error parsing replay cursor time in RoomEventsV3: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			cursorSequence, err := strconv.ParseUint(sequence, 10, 64)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error parsing replay cursor sequence in RoomEventsV3: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			cursorID = strconv.FormatUint(cursorTime, 10) + "-" + strconv.FormatUint(cursorSequence, 10)
		}

		container, err := events.SubscribeFrom(ctx, topicName, cursorID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error subscribing to room events: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		defer container.Subscription.Destroy()

		flusher, ok := w.(http.Flusher)
		if !ok {
			handleError(
				w,
				fmt.Errorf("error streaming not supported"),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		connection := vibe.RoomEventConnection{Time: time.Now().UnixMilli()}
		data, err := json.Marshal(connection)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling connection event in RoomEventsV3: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", vibe.Connected, data)
		if err != nil {
			log.Printf("error writing connection event in RoomEventsV3: %v", err)
			return
		}
		flusher.Flush()

		participantRegistered := false

		// Register signed sessions before reading the initial listener snapshot.
		// New cookie-less connections must survive one heartbeat before they
		// become listeners so short probes cannot manufacture participant rows.
		if userID != "" && !isRemoteController {
			// Leave this best-effort notification block on failure; still send the snapshot.
			switch {
			case !isNewSession:
				err = state.UpdateParticipant(ctx, roomID, userID, !isCastReceiver, isCastReceiver, castOwnerID)
				if err != nil && ctx.Err() != nil {
					return
				}
				if err != nil {
					log.Printf("error updating participant on connect: %v", err)
					break
				}
				participantRegistered = true

				counts, err := state.GetActiveListenerCounts(ctx, roomID, 15*time.Second)
				if err != nil {
					log.Printf("error fetching active listener count on connect: %v", err)
					break
				}

				count := counts.ActiveListeners
				if count == 0 && counts.ActiveCastReceivers > 0 {
					count = 1
				}

				payload, err := json.Marshal(count)
				if err != nil {
					log.Printf("error marshaling active listener count on connect: %v", err)
					break
				}

				err = events.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
					Type:    vibe.UsersUpdate,
					Payload: payload,
				})
				if err != nil {
					log.Printf("error notifying active listener count on connect: %v", err)
					break
				}
				lastUsersCount = count
			}

			defer func() {
				if !participantRegistered {
					return
				}

				departureCtx := context.WithoutCancel(ctx)
				err := events.NotifyAdminRoomDeparture(departureCtx, roomID)
				if err != nil {
					log.Printf("error notifying admin room departure: %v", err)
				}

				counts, err := state.GetActiveListenerCounts(departureCtx, roomID, 15*time.Second)
				if err != nil {
					log.Printf("error fetching active listener count on disconnect: %v", err)
					return
				}

				count := counts.ActiveListeners
				if count == 0 && counts.ActiveCastReceivers > 0 {
					count = 1
				}
				if count == lastUsersCount {
					return
				}

				payload, err := json.Marshal(count)
				if err != nil {
					log.Printf("error marshaling active listener count on disconnect: %v", err)
					return
				}

				err = events.NotifyRoomUpdateV3(context.WithoutCancel(departureCtx), roomID, vibe.RoomEventV3{
					Type:    vibe.UsersUpdate,
					Payload: payload,
				})
				if err != nil {
					log.Printf("error notifying active listener count on disconnect: %v", err)
					return
				}
				lastUsersCount = count
			}()
		}

		if replay.RequiresSnapshot {
			room, err := state.GetRoomV2(ctx, roomID, userID)
			if err != nil {
				log.Printf("error fetching room snapshot in RoomEventsV3: %v", err)
				return
			}
			if room == nil || room.IsEmpty() {
				log.Printf("error fetching room snapshot in RoomEventsV3: room is empty")
				return
			}
			roomData, err := json.Marshal(room)
			if err != nil {
				log.Printf("error marshaling room snapshot in RoomEventsV3: %v", err)
				return
			}

			playlistItems, err := state.GetPlaylistItems(ctx, roomID)
			if err != nil {
				log.Printf("error fetching playlist items snapshot in RoomEventsV3: %v", err)
				return
			}
			if playlistItems == nil {
				playlistItems = []vibe.PlaylistItem{}
			}
			playlistItemsData, err := json.Marshal(playlistItems)
			if err != nil {
				log.Printf("error marshaling playlist items snapshot in RoomEventsV3: %v", err)
				return
			}

			playback, err := state.GetPlaybackStateV2(ctx, roomID)
			if err != nil {
				log.Printf("error fetching playback snapshot in RoomEventsV3: %v", err)
				return
			}
			if playback == nil {
				log.Printf("error fetching playback snapshot in RoomEventsV3: playback is nil")
				return
			}
			if playback.IsPlaying && playback.UpdatedAt.Before(time.Now()) {
				playback.PositionMs += int(time.Since(playback.UpdatedAt).Milliseconds())
				playback.UpdatedAt = time.Now()
			}
			playback.ServerTimeMs = int(time.Now().UnixMilli())
			playbackData, err := json.Marshal(playback)
			if err != nil {
				log.Printf("error marshaling playback snapshot in RoomEventsV3: %v", err)
				return
			}

			if cursorID != "" {
				_, err = fmt.Fprintf(w, "id: %s\n", cursorID)
				if err != nil {
					log.Printf("error writing room snapshot event id in RoomEventsV3: %v", err)
					return
				}
			}
			_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", vibe.SettingsUpdate, roomData)
			if err != nil {
				log.Printf("error writing room snapshot in RoomEventsV3: %v", err)
				return
			}
			if cursorID != "" {
				cursorData, marshalErr := json.Marshal(vibe.RoomEventCursor{ID: cursorID})
				if marshalErr != nil {
					log.Printf("error marshaling room snapshot cursor in RoomEventsV3: %v", marshalErr)
					return
				}
				_, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", cursorID, vibe.EventCursor, cursorData)
				if err != nil {
					log.Printf("error writing room snapshot cursor in RoomEventsV3: %v", err)
					return
				}
			}
			flusher.Flush()

			if cursorID != "" {
				_, err = fmt.Fprintf(w, "id: %s\n", cursorID)
				if err != nil {
					log.Printf("error writing playlist items snapshot event id in RoomEventsV3: %v", err)
					return
				}
			}
			_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", vibe.PlaylistItemsSnapshot, playlistItemsData)
			if err != nil {
				log.Printf("error writing playlist items snapshot in RoomEventsV3: %v", err)
				return
			}
			if cursorID != "" {
				cursorData, marshalErr := json.Marshal(vibe.RoomEventCursor{ID: cursorID})
				if marshalErr != nil {
					log.Printf("error marshaling playlist items snapshot cursor in RoomEventsV3: %v", marshalErr)
					return
				}
				_, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", cursorID, vibe.EventCursor, cursorData)
				if err != nil {
					log.Printf("error writing playlist items snapshot cursor in RoomEventsV3: %v", err)
					return
				}
			}
			flusher.Flush()

			if cursorID != "" {
				_, err = fmt.Fprintf(w, "id: %s\n", cursorID)
				if err != nil {
					log.Printf("error writing playback snapshot event id in RoomEventsV3: %v", err)
					return
				}
			}
			_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", vibe.PlaybackUpdate, playbackData)
			if err != nil {
				log.Printf("error writing playback snapshot in RoomEventsV3: %v", err)
				return
			}
			if cursorID != "" {
				cursorData, marshalErr := json.Marshal(vibe.RoomEventCursor{ID: cursorID})
				if marshalErr != nil {
					log.Printf("error marshaling playback snapshot cursor in RoomEventsV3: %v", marshalErr)
					return
				}
				_, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", cursorID, vibe.EventCursor, cursorData)
				if err != nil {
					log.Printf("error writing playback snapshot cursor in RoomEventsV3: %v", err)
					return
				}
			}
			flusher.Flush()
		}

		presenceTicker := time.NewTicker(5 * time.Second)
		defer presenceTicker.Stop()
		keepAliveTicker := time.NewTicker(15 * time.Second)
		defer keepAliveTicker.Stop()

		messages := container.Subscription.Listen()

		for {
			select {
			case <-ctx.Done():
				return
			case <-presenceTicker.C:
				// Update participant status independently from the wire keep-alive.
				if userID == "" || isRemoteController {
					continue
				}

				err = state.UpdateParticipant(ctx, roomID, userID, !isCastReceiver, isCastReceiver, castOwnerID)
				if err != nil && ctx.Err() != nil {
					return
				}
				if err != nil {
					log.Printf("error updating participant on heartbeat: %v", err)
					continue
				}

				participantRegistered = true
				counts, err := state.GetActiveListenerCounts(ctx, roomID, 15*time.Second)
				if err != nil {
					log.Printf("error fetching active listener count on heartbeat: %v", err)
					continue
				}

				count := counts.ActiveListeners
				if count == 0 && counts.ActiveCastReceivers > 0 {
					count = 1
				}
				if count == lastUsersCount {
					continue
				}

				payload, err := json.Marshal(count)
				if err != nil {
					log.Printf("error marshaling active listener count on heartbeat: %v", err)
					continue
				}

				err = events.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
					Type:    vibe.UsersUpdate,
					Payload: payload,
				})
				if err != nil {
					log.Printf("error notifying active listener count on heartbeat: %v", err)
					continue
				}
				lastUsersCount = count
			case <-keepAliveTicker.C:
				_, err = fmt.Fprint(w, ": heartbeat\n\n")
				if err != nil {
					log.Printf("error writing heartbeat in RoomEventsV3: %v", err)
					return
				}
				flusher.Flush()
			case data, ok := <-messages:
				if !ok {
					return
				}

				event, err := vibe.ParseRoomEvent(data)
				if err != nil {
					log.Printf("failed to unmarshal room event: %v", err)
					continue
				}
				if event.Type == vibe.UsersUpdate {
					err = json.Unmarshal(event.Payload, &lastUsersCount)
					if err != nil {
						log.Printf("failed to unmarshal users update: %v", err)
						continue
					}
				}

				filterID := userID
				if isCastReceiver && castOwnerID != "" {
					filterID = castOwnerID
				}
				if event.UserID != "" &&
					event.UserID == filterID &&
					event.Origin != vibe.RoomEventOriginRemote {
					if event.ID != "" {
						cursorData, marshalErr := json.Marshal(vibe.RoomEventCursor{ID: event.ID})
						if marshalErr != nil {
							log.Printf("error marshaling filtered event cursor in RoomEventsV3: %v", marshalErr)
							return
						}
						_, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, vibe.EventCursor, cursorData)
						if err != nil {
							log.Printf("error writing filtered event cursor in RoomEventsV3: %v", err)
							return
						}
						flusher.Flush()
					}
					continue
				}

				converted, err := event.ToRoomEventV3Payload()
				if err != nil {
					log.Printf("error converting room event in RoomEventsV3: %v", err)
					return
				}

				eventType := converted.Type
				payload := converted.Payload

				if event.ID != "" {
					_, err = fmt.Fprintf(w, "id: %s\n", event.ID)
					if err != nil {
						log.Printf("error writing event id in RoomEventsV3: %v", err)
						return
					}
				}
				_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, payload)
				if err != nil {
					log.Printf("error writing event in RoomEventsV3: %v", err)
					return
				}
				if event.ID != "" {
					cursorData, marshalErr := json.Marshal(vibe.RoomEventCursor{ID: event.ID})
					if marshalErr != nil {
						log.Printf("error marshaling event cursor in RoomEventsV3: %v", marshalErr)
						return
					}
					_, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, vibe.EventCursor, cursorData)
					if err != nil {
						log.Printf("error writing event cursor in RoomEventsV3: %v", err)
						return
					}
				}
				flusher.Flush()
			}
		}
	}
}

// GetPublicRoomsV3 handles GET /api/v3/rooms/public.
//
//	@Summary		Browse public rooms
//	@Description	Returns public rooms ordered by listener count, playlist item count, then ID, all descending. Ranges are zero-based and inclusive. Only rooms with protected admin controls and at least one playlist item from an enabled provider are listed, including when filtering for live rooms.
//	@Tags			rooms
//	@Produce		json
//	@Param			q		query		string	false	"Case-insensitive room name search (literal substring, up to 100 characters)"
//	@Param			live	query		boolean	false	"Only rooms with active listeners" default(false)
//	@Param			roomType	query	string	false	"Room type to browse" Enums(MUSIC,WATCH) default(MUSIC)
//	@Param			from	query		int		false	"First row, zero-based" minimum(0) default(0)
//	@Param			to		query		int		false	"Last row, inclusive. Defaults to from + 9; at most 100 rooms per page" minimum(0)
//	@Success		200		{object}	vibe.PublicRoomResultV3
//	@Failure		400		{object}	vibe.ErrorResponse
//	@Failure		500		{object}	vibe.ErrorResponse
//	@Router			/api/v3/rooms/public [get]
func GetPublicRoomsV3(db vibe.PublicRoomsV3Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomType := vibe.RoomType(r.URL.Query().Get("roomType"))
		if roomType == "" {
			roomType = vibe.RoomTypeMusic
		}

		if !roomType.IsValid() {
			handleError(
				w,
				fmt.Errorf("error browsing public rooms: roomType must be MUSIC or WATCH"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if !utf8.ValidString(query) || strings.ContainsRune(query, '\x00') {
			handleError(
				w,
				fmt.Errorf("error invalid room search text"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		if utf8.RuneCountInString(query) > publicRoomMaximumQueryLength {
			handleError(
				w,
				fmt.Errorf("error room search is too long"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		live := r.URL.Query().Get("live")
		if live != "" && live != "true" && live != "false" {
			handleError(
				w,
				fmt.Errorf("error invalid live room filter"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		var from int64
		var err error
		fromValue := r.URL.Query().Get("from")
		if fromValue != "" {
			from, err = strconv.ParseInt(fromValue, 10, 32)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error parsing first room row: %w", err),
					http.StatusBadRequest,
					false,
				)
				return
			}

			if from < 0 {
				handleError(
					w,
					fmt.Errorf("error invalid first room row"),
					http.StatusBadRequest,
					false,
				)
				return
			}
		}

		to := from + publicRoomPageSize - 1
		toValue := r.URL.Query().Get("to")
		if toValue != "" {
			to, err = strconv.ParseInt(toValue, 10, 32)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error parsing last room row: %w", err),
					http.StatusBadRequest,
					false,
				)
				return
			}

			if to < from {
				handleError(
					w,
					fmt.Errorf("error invalid last room row"),
					http.StatusBadRequest,
					false,
				)
				return
			}
		}

		if to-from+1 > publicRoomMaximumPageSize {
			handleError(
				w,
				fmt.Errorf("error room page is too large"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		result, err := db.SearchPublicRoomsV3(ctx, vibe.PublicRoomSearch{
			RoomType: roomType,
			Query:    query,
			Live:     live == "true",
			From:     int(from),
			To:       int(to),
		})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error searching public rooms: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(result)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling public rooms: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
