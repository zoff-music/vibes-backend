package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// DeleteRemoteControl revokes a machine's remote capability.
//
//	@Summary	Disable remote control
//	@Tags		remotes
//	@Param		id	path	string	true	"Remote ID"
//	@Success	204
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/remotes/{id} [delete]
func DeleteRemoteControl(deleter vibe.OwnedRemoteControlDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		remoteID := mux.Vars(r)["id"]
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" || session.AuthType != "cookie" {
			handleError(
				w,
				fmt.Errorf("error cookie session required to disable remote control"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		err := deleter.DeleteRemoteControl(ctx, remoteID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error deleting remote control: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

const remoteSessionCookieName = "remote_session"

const remotePairingTokenMaxLength = 128

const remotePairingCodeLength = 8

const remotePlaybackPositionMaxMs = int64((24 * time.Hour) / time.Millisecond)
