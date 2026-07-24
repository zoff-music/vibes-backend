package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GenerateRoom handles POST /api/v1/rooms/generations.
//
//	@Summary	Generate a room with a YouTube playlist
//	@Tags		rooms
//	@Accept		json
//	@Produce	json
//	@Param		request	body		vibe.GeneratedPlaylistRequest	true	"Playlist prompt"
//	@Success	201		{object}	vibe.GeneratedRoom
//	@Failure	400		{object}	map[string]string
//	@Failure	429		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/rooms/generations [post]
func GenerateRoom(
	ai vibe.PlaylistGenerator,
	verifier vibe.GeneratedPlaylistVerifier,
	db vibe.GeneratedRoomCreator,
	maxPromptLength int,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var request vibe.GeneratedPlaylistRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, playlistRequestMaxBytes))
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&request)
		if err != nil {
			handleError(w, fmt.Errorf("error decoding playlist request: %w", err), http.StatusBadRequest, false)
			return
		}

		request.Prompt = strings.TrimSpace(request.Prompt)
		if request.Prompt == "" {
			handleError(w, fmt.Errorf("error playlist prompt is required"), http.StatusBadRequest, false)
			return
		}
		if utf8.RuneCountInString(request.Prompt) > maxPromptLength {
			handleError(w, fmt.Errorf("error playlist prompt exceeds %d characters", maxPromptLength), http.StatusBadRequest, false)
			return
		}

		playlist, err := ai.GeneratePlaylist(ctx, request.Prompt)
		if err != nil {
			handleError(w, fmt.Errorf("error generating playlist: %w", err), http.StatusInternalServerError, true)
			return
		}

		err = verifier.VerifyGeneratedPlaylist(ctx, playlist)
		if err != nil {
			handleError(w, fmt.Errorf("error verifying generated playlist: %w", err), http.StatusInternalServerError, true)
			return
		}

		candidates, err := vibe.GenerateRoomNameCandidates()
		if err != nil {
			handleError(w, fmt.Errorf("error generating room name candidates: %w", err), http.StatusInternalServerError, true)
			return
		}
		suggestion, err := db.SuggestRoomName(ctx, candidates)
		if err != nil {
			handleError(w, fmt.Errorf("error suggesting generated room name: %w", err), http.StatusInternalServerError, true)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(w, fmt.Errorf("error generated room session is missing"), http.StatusUnauthorized, false)
			return
		}

		room := &vibe.Room{
			ID:            helper.Slugify(suggestion.Name),
			Name:          suggestion.Name,
			Mode:          vibe.RoomModeServer,
			HostID:        session.UserID,
			Settings:      vibe.DefaultRoomSettings(),
			CreatedAt:     time.Now(),
			ActiveSources: []string{},
		}
		generatedRoom, err := db.CreateGeneratedRoom(ctx, room, playlist)
		if err != nil {
			handleError(w, fmt.Errorf("error creating generated room: %w", err), http.StatusInternalServerError, true)
			return
		}

		body, err := json.Marshal(generatedRoom)
		if err != nil {
			handleError(w, fmt.Errorf("error marshaling generated room: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	}
}

const playlistRequestMaxBytes = 4096
