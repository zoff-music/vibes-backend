package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetPlaybackState handles GET /rooms/{id}/states
//
//	@Summary	Get playback state
//	@Tags		playback
//	@Produce	json
//	@Param		id	path		string	true	"Room ID"
//	@Success	200	{object}	vibe.PlaybackState
//	@Failure	401	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/states [get]
func GetPlaybackState(
	db vibe.PlaybackFetcher,
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

		state, err := db.GetPlaybackState(ctx, roomID)
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

// ReportPlaybackFailure handles POST /rooms/{id}/playbackfailures
//
//	@Summary	Report a restricted playback failure
//	@Tags		playback
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Room ID"
//	@Param		request	body		vibe.PlaybackFailureRequest	true	"Failed song"
//	@Success	200		{object}	vibe.PlaybackState
//	@Failure	400		{object}	map[string]string
//	@Failure	401		{object}	map[string]string
//	@Failure	409		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/playbackfailures [post]
func ReportPlaybackFailure(
	db vibe.PlaybackFailureStorage,
	trackFetcher vibe.MusicTrackFetcher,
	ips vibe.RoomBatchEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["id"]

		var request vibe.PlaybackFailureRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding playback failure request: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}
		if request.SongID == "" {
			handleError(
				w,
				fmt.Errorf("error song ID is required"),
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

		state, err := db.GetPlaybackState(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching playback state for playback failure: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if state.CurrentSong == nil || state.CurrentSong.ID != request.SongID {
			handleError(
				w,
				fmt.Errorf("error song is no longer the current song"),
				http.StatusConflict,
				false,
			)
			return
		}

		if state.CurrentSong.PlaybackRestriction == "" {
			if state.CurrentSong.SourceType != vibe.SourceTypeYouTube {
				handleError(
					w,
					fmt.Errorf("error provider does not support playback restriction verification"),
					http.StatusConflict,
					false,
				)
				return
			}

			track, err := trackFetcher.GetTrack(ctx, state.CurrentSong.SourceID)
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

			err = db.UpdateSongPlaybackRestriction(
				ctx,
				roomID,
				request.SongID,
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

		state, err = db.SkipRestrictedSong(ctx, roomID, request.SongID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error skipping failed restricted song: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if state.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error song is no longer the current restricted song"),
				http.StatusConflict,
				false,
			)
			return
		}

		state.ServerTimeMs = int(time.Now().UnixMilli())
		stateBody, err := json.Marshal(state)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling restricted playback state: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		songs, err := db.GetSongs(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching songs after restricted playback skip: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		songsBody, err := json.Marshal(songs)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling songs after restricted playback skip: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = ips.NotifyRoomUpdates(context.WithoutCancel(ctx), roomID, []vibe.RoomEvent{
			{
				Type:    vibe.QueueReordered,
				Payload: songsBody,
			},
			{
				Type:    vibe.PlaybackUpdate,
				Payload: stateBody,
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

// UpdatePlaybackState handles PUT /rooms/{id}/states
//
//	@Summary	Update playback state
//	@Tags		playback
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Room ID"
//	@Param		request	body		vibe.RoomActionRequest	true	"Playback action"
//	@Success	200		{object}	vibe.PlaybackState
//	@Failure	400		{object}	map[string]string
//	@Failure	401		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/states [put]
func UpdatePlaybackState(
	db vibe.RoomGetterPlaybackUpdater,
	ips vibe.RoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["id"]

		var req vibe.RoomActionRequest
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

		room, err := db.GetRoom(ctx, roomID, userID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		var state *vibe.PlaybackState

		if room.Mode == vibe.RoomModeServer {
			handleError(
				w,
				fmt.Errorf("error shared playback controls are unavailable in server mode"),
				http.StatusForbidden,
				false,
			)
			return
		}

		state, err = db.UpdatePlayback(ctx, roomID, userID, req.Action, req.PositionMs)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error updating playback: %w", err),
				http.StatusInternalServerError,
				true,
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

			err = ips.NotifyRoomUpdate(context.WithoutCancel(ctx), roomID, vibe.RoomEvent{
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

// ReviewRoomPlayback handles playback monitoring for server-mode rooms
type ReviewRoomPlayback struct {
	DB  vibe.ExpiredPlaybackSongFetcher
	IPS vibe.RoomBatchEventNotifier
}

// Handle checks for rooms that need to auto-advance
func (h *ReviewRoomPlayback) Handle(ctx context.Context, data []byte) error {
	state, err := h.DB.ProcessNextExpiredPlayback(ctx)
	if err != nil {
		return fmt.Errorf("error processing next expired playback: %w", err)
	}

	statePayload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("error marshaling playback state payload: %w", err)
	}

	songs, err := h.DB.GetSongs(ctx, state.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching songs for room %s: %w", state.RoomID, err)
	}

	songsPayload, err := json.Marshal(songs)
	if err != nil {
		return fmt.Errorf("error marshaling songs payload: %w", err)
	}

	err = h.IPS.NotifyRoomUpdates(ctx, state.RoomID, []vibe.RoomEvent{
		{
			Type:    vibe.PlaybackUpdate,
			Payload: statePayload,
		},
		{
			Type:    vibe.QueueReordered,
			Payload: songsPayload,
		},
	})
	if err != nil {
		return fmt.Errorf("error notifying room %s update: %w", state.RoomID, err)
	}

	return nil
}

// ReviewHostHealth handles host health checks
type ReviewHostHealth struct {
	DB  vibe.AbandonedHostProcessor
	IPS vibe.RoomEventNotifier
}

// Handle checks for rooms that need a new host
func (h *ReviewHostHealth) Handle(ctx context.Context, data []byte) error {
	info, err := h.DB.ProcessNextAbandonedHost(ctx)
	if err != nil {
		return fmt.Errorf("error processing next abandoned host: %w", err)
	}

	payloadMap := map[string]string{"userId": info.NewHostID, "message": "You are now the host"}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return fmt.Errorf("error marshaling new host payload: %w", err)
	}

	err = h.IPS.NotifyRoomUpdate(ctx, info.RoomID, vibe.RoomEvent{
		Type:    vibe.NewHost,
		Payload: payloadBytes,
	})
	if err != nil {
		return fmt.Errorf("error notifying room %s host update: %w", info.RoomID, err)
	}

	return nil
}
