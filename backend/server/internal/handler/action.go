package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/vibe"
)

// RoomAction handles POST /api/v1/rooms/:id
func RoomAction(
	ra vibe.RoomActioner,
	eb vibe.RoomEventBroadcaster,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.RoomActionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest, true)
			return
		}

		userID := r.Header.Get("X-User-ID")

		state, err := ra.GetPlaybackState(ctx, roomID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to fetch playback state: %w", err), http.StatusInternalServerError, true)
			return
		}

		switch req.Action {
		case vibe.RoomActionPlay:
			// If no song is currently playing, promote the first song from the queue
			if state.CurrentSongID == nil {
				err = skipToNextSong(ctx, roomID, state, ra, ra)
				if err != nil {
					handleError(w, fmt.Errorf("failed to start playback: %w", err), http.StatusInternalServerError, true)
					return
				}
			} else {
				state.IsPlaying = true
				state.UpdatedAt = time.Now()
				err = ra.UpsertPlaybackState(ctx, state)
			}
		case vibe.RoomActionPause:
			state.IsPlaying = false
			state.UpdatedAt = time.Now()
			err = ra.UpsertPlaybackState(ctx, state)
		case vibe.RoomActionSeek:
			state.PositionMs = req.PositionMs
			state.UpdatedAt = time.Now()
			err = ra.UpsertPlaybackState(ctx, state)
		case vibe.RoomActionSkip:
			err = skipToNextSong(ctx, roomID, state, ra, ra)
		case vibe.RoomActionVote:
			err = handleVote(ctx, roomID, userID, state, ra, ra, ra, ra, ra, ra)
		default:
			handleError(w, fmt.Errorf("invalid action: %s", req.Action), http.StatusBadRequest, false)
			return
		}

		if err != nil {
			handleError(w, fmt.Errorf("action failed: %w", err), http.StatusInternalServerError, true)
			return
		}

		// Update server time for response
		state.ServerTimeMs = time.Now().UnixMilli()

		eb.BroadcastToRoom(ctx, roomID, &vibe.RoomEvent{
			Type:    vibe.EventTypePlaybackUpdate,
			Payload: state,
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state)
	}
}

func skipToNextSong(ctx context.Context, roomID string, state *vibe.PlaybackState, sm vibe.SongsModifier, pc vibe.PlaybackController) error {
	currentPosition := -1
	if state.CurrentSong != nil {
		currentPosition = state.CurrentSong.Position
	} else if state.CurrentSongID != nil {
		// If we only have ID, we'd need to fetch the song to get position,
		// but typically state should have enough info or we can assume sequential
		// For now let's just assume we need to fetch if currentPosition is -1
		// Actually GetNextSong takes currentPosition.
	}

	nextSong, err := sm.GetNextSong(ctx, roomID, currentPosition)
	if err != nil {
		return fmt.Errorf("failed to get next song: %w", err)
	}

	if nextSong.IsEmpty() {
		state.CurrentSongID = nil
		state.CurrentSong = nil
		state.IsPlaying = false
	} else {
		state.CurrentSongID = &nextSong.ID
		state.CurrentSong = nextSong
		state.IsPlaying = true
	}

	state.PositionMs = 0
	state.UpdatedAt = time.Now()

	return pc.UpsertPlaybackState(ctx, state)
}

func handleVote(ctx context.Context, roomID, userID string, state *vibe.PlaybackState, rf vibe.RoomFetcher, svf vibe.SkipVoteFetcher, svm vibe.SkipVoteManager, sm vibe.SongsModifier, pc vibe.PlaybackController, uf vibe.UserFetcher) error {
	if state.CurrentSongID == nil || userID == "" {
		return nil
	}

	songID := *state.CurrentSongID

	voted, err := svf.HasUserVoted(ctx, roomID, songID, userID)
	if err != nil {
		return fmt.Errorf("failed to check vote: %w", err)
	}

	if voted {
		return nil // Already voted
	}

	err = svm.AddSkipVote(ctx, roomID, songID, userID)
	if err != nil {
		return fmt.Errorf("failed to add vote: %w", err)
	}

	// Check if threshold reached
	room, err := rf.GetRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to fetch room: %w", err)
	}

	votes, err := svf.GetSkipVotes(ctx, roomID, songID)
	if err != nil {
		return fmt.Errorf("failed to fetch votes: %w", err)
	}

	userCount, err := uf.CountUsersInRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if float64(len(votes)) >= float64(userCount)*room.Settings.SkipVoteThreshold {
		err = skipToNextSong(ctx, roomID, state, sm, pc)
		if err != nil {
			return fmt.Errorf("failed to skip after vote: %w", err)
		}
		_ = svm.ClearSkipVotes(ctx, roomID, songID)
	}

	return nil
}
