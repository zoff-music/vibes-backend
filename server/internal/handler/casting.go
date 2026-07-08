package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// CreateCastingToken handles POST /api/v1/casting/tokens.
// It requires a cookie-authenticated session (not a cast bearer token).
func CreateCastingToken(
	db vibe.RoomFetcher,
	castTokenSecret string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

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
		if session.AuthType != "" && session.AuthType != "cookie" {
			handleError(
				w,
				fmt.Errorf("forbidden"),
				http.StatusForbidden,
				false,
			)
			return
		}

		var req vibe.CreateCastingTokenRequest
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
		if req.RoomID == "" {
			handleError(
				w,
				fmt.Errorf("roomId is required"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		room, err := db.GetRoom(ctx, req.RoomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if room == nil || room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		now := time.Now()
		expiresAt := now.Add(castTokenTTL)
		token, err := helper.SignCastToken(castTokenSecret, helper.CastTokenPayload{
			RoomID: req.RoomID,
			UserID: session.UserID,
			Iat:    now.Unix(),
			Exp:    expiresAt.Unix(),
		})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error signing cast token: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		resp := vibe.CastingTokenResponse{
			Token:     token,
			ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
			RoomID:    req.RoomID,
		}

		body, err := json.Marshal(resp)
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

const castTokenTTL = 6 * time.Hour
