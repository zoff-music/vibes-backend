package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// RemoteEvents streams machine room changes between a machine and controller.
//
//	@Summary	Subscribe to remote control events
//	@Tags		remotes
//	@Produce	text/event-stream
//	@Param		id	path		string	true	"Remote ID"
//	@Success	200	{string}	string
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Router		/api/v1/remotes/{id}/events [get]
func RemoteEvents(
	subscriber vibe.Subscriber,
	fetcher vibe.RemoteControlFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		remoteID := mux.Vars(r)["id"]
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok ||
			session.UserID == "" ||
			(session.AuthType != "cookie" && session.AuthType != "remote") {
			handleError(
				w,
				fmt.Errorf("error remote event session required"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		remote, err := fetcher.GetRemoteControl(ctx, remoteID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting remote control for events: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if remote.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error remote control not found"),
				http.StatusNotFound,
				false,
			)
			return
		}
		if session.AuthType == "cookie" && remote.OwnerUserID != session.UserID {
			handleError(
				w,
				fmt.Errorf("error remote event access forbidden"),
				http.StatusForbidden,
				false,
			)
			return
		}
		if session.AuthType == "remote" && session.RemoteID != remote.ID {
			handleError(
				w,
				fmt.Errorf("error remote event access forbidden"),
				http.StatusForbidden,
				false,
			)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		container, err := subscriber.Subscribe(ctx, fmt.Sprintf("remote:%s", remoteID))
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error subscribing to remote events: %w", err),
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
				fmt.Errorf("error remote event streaming not supported"),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		initialEvent := vibe.RemoteEvent{
			Type:               vibe.RemoteStateUpdate,
			RoomID:             remote.CurrentRoomID,
			Origin:             vibe.RemoteOriginMachine,
			Online:             true,
			Paired:             remote.Paired,
			CurrentSongID:      remote.CurrentSongID,
			PlaybackPositionMs: remote.PlaybackPositionMs,
			PlaybackIsPlaying:  remote.PlaybackIsPlaying,
			PlaybackObservedAt: remote.PlaybackObservedAt,
		}
		data, err := json.Marshal(initialEvent)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling initial remote event: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", initialEvent.Type, data)
		flusher.Flush()

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		messages := container.Subscription.Listen()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Fprint(w, ": heartbeat\n\n")
				flusher.Flush()
			case data, ok := <-messages:
				if !ok {
					return
				}

				var event vibe.RemoteEvent
				err = json.Unmarshal(data, &event)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
				flusher.Flush()
			}
		}
	}
}
