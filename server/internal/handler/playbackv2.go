package handler

import (
	"context"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
	"log"
	"net/http"
	"time"
)

// GetPlaybackStateV2 handles GET /rooms/{id}/states
//
//	@Summary	Get playback state
//	@Tags		playback
//	@Produce	json
//	@Param		id	path		string	true	"Room ID"
//	@Success	200	{object}	vibe.PlaybackStateV2
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/states [get]
func GetPlaybackStateV2(
	db vibe.PlaybackV2Fetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

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

		state, err := db.GetPlaybackStateV2(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching playback state: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		state.ServerTimeMs = int(time.Now().UnixMilli())

		body, err := json.Marshal(state)
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

// UpdatePlaybackStateV2 handles PUT /rooms/{id}/states
//
//	@Summary	Update playback state
//	@Tags		playback
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Room ID"
//	@Param		request	body		vibe.RoomActionRequest	true	"Playback action"
//	@Success	200		{object}	vibe.PlaybackStateV2
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	403		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/states [put]
func UpdatePlaybackStateV2(
	db vibe.RoomV2GetterPlaybackUpdater,
	events vibe.RoomRemoteEventV3Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["id"]

		var req vibe.RoomActionRequest
		err := jsonv2.UnmarshalRead(r.Body, &req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		switch req.Action {
		case vibe.RoomActionPlay, vibe.RoomActionPause, vibe.RoomActionSeek:
			// Allowed
		default:
			handleError(
				w,
				fmt.Errorf("error invalid action for state update: %s", req.Action),
				http.StatusBadRequest,
				false,
			)
			return
		}

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

		room, err := db.GetRoomV2(ctx, roomID, userID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		var state *vibe.PlaybackStateV2

		if session.AuthType == "remote" && room.Mode == vibe.RoomModeServer {
			state, err = db.GetPlaybackStateV2(ctx, roomID)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error fetching playback for remote update: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			state.PositionMs = req.PositionMs
			if req.Action == vibe.RoomActionPlay {
				state.IsPlaying = true
			}
			if req.Action == vibe.RoomActionPause {
				state.IsPlaying = false
			}
			observedAt := time.Now()
			state.UpdatedAt = observedAt
			state.ServerTimeMs = int(observedAt.UnixMilli())
			currentPlaylistItemID := ""
			if state.CurrentPlaylistItem != nil {
				currentPlaylistItemID = state.CurrentPlaylistItem.ID
			}

			err = events.NotifyRemoteUpdateV2(context.WithoutCancel(ctx), session.RemoteID, vibe.RemoteEventV2{
				Type:                  vibe.RemoteStateUpdate,
				RoomID:                roomID,
				Origin:                vibe.RemoteOriginController,
				Online:                true,
				Paired:                true,
				CurrentPlaylistItemID: currentPlaylistItemID,
				PlaybackPositionMs:    state.PositionMs,
				PlaybackIsPlaying:     state.IsPlaying,
				PlaybackObservedAt:    observedAt,
			})
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error notifying machine of remote playback update: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			body, err := json.Marshal(state)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshalling remote playback response in update playback state handler: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}

		if room.Mode == vibe.RoomModeServer {
			handleError(
				w,
				fmt.Errorf("error shared playback controls are unavailable in server mode"),
				http.StatusForbidden,
				false,
			)
			return
		}

		state, err = db.UpdatePlaybackV2(ctx, roomID, userID, req.Action, req.PositionMs)
		if err != nil {
			var permissionError internalerror.ErrorCodeWrapper
			isHostDenial := errors.As(err, &permissionError) &&
				permissionError.StatusCode == http.StatusForbidden &&
				permissionError.ResponseBody.Error == "playback_host_required"

			handleError(
				w,
				fmt.Errorf("error updating playback: %w", err),
				http.StatusInternalServerError,
				!isHostDenial,
			)
			return
		}

		state.ServerTimeMs = int(time.Now().UnixMilli())

		if room.Mode == vibe.RoomModeHost {
			statePayload, err := json.Marshal(state)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshalling playback state payload in update playback state handler: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			err = events.NotifyRoomUpdateV3(context.WithoutCancel(ctx), roomID, vibe.RoomEventV3{
				Type:    vibe.PlaybackUpdate,
				Payload: statePayload,
				Origin:  session.EventOrigin,
			})
			if err != nil {
				log.Printf("error notifying room of playback update: %v", err)
			}
		}

		body, err := json.Marshal(state)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshalling response in update playback state handler: %w", err),
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

// ReportPlaybackFailureV2 handles POST /rooms/{id}/failures
//
//	@Summary	Report a restricted playback failure
//	@Tags		playback
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Room ID"
//	@Param		request	body		vibe.PlaybackFailureRequestV2	true	"Failed playlist item"
//	@Success	200		{object}	vibe.PlaybackStateV2
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	409		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v2/rooms/{id}/failures [post]
func ReportPlaybackFailureV2(
	db vibe.PlaybackFailureV2Storage,
	itemFetcher vibe.ProviderItemFetcher,
	notifier vibe.RoomBatchEventV3Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["id"]

		var request vibe.PlaybackFailureRequestV2
		err := jsonv2.UnmarshalRead(r.Body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding playback failure request: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}
		if request.PlaylistItemID == "" {
			handleError(
				w,
				fmt.Errorf("error playlist item ID is required"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" || session.AuthType != "cast" {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		state, err := db.GetPlaybackStateV2(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching playback state for playback failure: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if state.CurrentPlaylistItem == nil || state.CurrentPlaylistItem.ID != request.PlaylistItemID {
			handleError(
				w,
				fmt.Errorf("error playlist item is no longer the current playlist item"),
				http.StatusConflict,
				false,
			)
			return
		}

		if state.CurrentPlaylistItem.PlaybackRestriction == "" {
			if state.CurrentPlaylistItem.SourceType != vibe.SourceTypeYouTube {
				handleError(
					w,
					fmt.Errorf("error provider does not support playback restriction verification"),
					http.StatusConflict,
					false,
				)
				return
			}

			track, err := itemFetcher.GetProviderItem(ctx, state.CurrentPlaylistItem.SourceID)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error verifying playback restriction: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			if track.PlaybackRestriction == "" {
				handleError(
					w,
					fmt.Errorf("error provider metadata does not restrict playback"),
					http.StatusConflict,
					false,
				)
				return
			}

			err = db.UpdatePlaylistItemPlaybackRestriction(
				ctx,
				roomID,
				request.PlaylistItemID,
				track.PlaybackRestriction,
			)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error storing playback restriction: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
		}

		advance, err := db.SkipRestrictedPlaylistItem(ctx, roomID, request.PlaylistItemID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error skipping failed restricted playlist item: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if advance.Playback.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error playlist item is no longer the current restricted playlist item"),
				http.StatusConflict,
				false,
			)
			return
		}

		advance.Playback.ServerTimeMs = int(time.Now().UnixMilli())
		stateBody, err := json.Marshal(advance.Playback)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling restricted playback state: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		playlistItems, err := db.GetPlaylistItems(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching playlist items after restricted playback skip: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		playlistItemsBody, err := json.Marshal(playlistItems)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling playlist items after restricted playback skip: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		var compactEvent *vibe.RoomEventV3Payload
		for position, playlistItem := range playlistItems {
			if playlistItem.ID != advance.PreviousPlaylistItemID {
				continue
			}

			compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemPositionUpdate{
				PlaylistItem: playlistItem,
				Position:     position,
			})
			if marshalErr != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling compact restricted skip event: %w", marshalErr),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemUpdated, Payload: compactPayload}
			break
		}
		if compactEvent == nil {
			compactPayload, marshalErr := json.Marshal(vibe.PlaylistItemIDUpdate{ID: advance.PreviousPlaylistItemID})
			if marshalErr != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling compact restricted skip removal fallback: %w", marshalErr),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			compactEvent = &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemRemoved, Payload: compactPayload}
		}

		playbackBody, err := json.Marshal(advance.Playback)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling restricted playback: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = notifier.NotifyRoomUpdatesV3(context.WithoutCancel(ctx), roomID, []vibe.RoomEventV3{
			{
				Type:    vibe.PlaylistItemsUpdate,
				Payload: playlistItemsBody,
				Compact: compactEvent,
			},
			{
				Type:    vibe.PlaybackUpdate,
				Payload: playbackBody,
			},
		})
		if err != nil {
			log.Printf("error notifying room of restricted playback skip: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(stateBody)
	}
}
