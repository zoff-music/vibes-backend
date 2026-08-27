package handler

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strings"

	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetSessionProfile handles GET /api/v1/sessions.
//
//	@Summary	Get the current session profile
//	@Tags		sessions
//	@Produce	json
//	@Success	200	{object}	vibe.SessionProfile
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/sessions [get]
func GetSessionProfile(db vibe.SessionProfileFetcherCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error getting session profile: unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		profile, err := db.GetOrCreateSessionProfile(ctx, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting session profile: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(profile)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling session profile: %w", err),
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

// UpdateSessionProfile handles PATCH /api/v1/sessions.
//
//	@Summary	Update the current session profile
//	@Tags		sessions
//	@Accept		json
//	@Produce	json
//	@Param		request	body		vibe.UpdateSessionProfileRequest	true	"Session profile"
//	@Success	200		{object}	vibe.SessionProfile
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v1/sessions [patch]
func UpdateSessionProfile(db vibe.SessionProfileUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error updating session profile: unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		var request vibe.UpdateSessionProfileRequest
		err := json.UnmarshalRead(r.Body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding session profile: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}
		if !request.Validate() {
			handleError(
				w,
				fmt.Errorf("error validating session profile name"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		profile, err := db.UpdateSessionProfile(ctx, session.UserID, strings.TrimSpace(request.Name))
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error updating session profile: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(profile)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling session profile: %w", err),
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
