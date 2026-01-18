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
			err = handleVote(ctx, roomID, userID, state, ra)
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

		// If the action was skip or vote, the queue likely changed
		if req.Action == vibe.RoomActionSkip || req.Action == vibe.RoomActionVote || req.Action == vibe.RoomActionPlay {
			// Fetch and broadcast updated queue
			// Note: ra (RoomActioner) includes SongsModifier which now has GetSongs
			songs, err := ra.GetSongs(ctx, roomID)
			if err == nil {
				eb.BroadcastToRoom(ctx, roomID, &vibe.RoomEvent{
					Type:    vibe.EventTypeQueueReordered,
					Payload: songs,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state)
	}
}

func skipToNextSong(ctx context.Context, roomID string, state *vibe.PlaybackState, ra vibe.RoomActioner, pc vibe.PlaybackController) error {
	// 1. Identify the song we are leaving (the currently playing one)
	var currentSongID *string
	currentPosition := -1

	if state.CurrentSongID != nil {
		currentSongID = state.CurrentSongID
		if state.CurrentSong != nil {
			currentPosition = state.CurrentSong.Position
		} else {
			// Fetch it if we only have the ID
			song, err := ra.GetSong(ctx, roomID, *currentSongID)
			if err == nil && !song.IsEmpty() {
				currentPosition = song.Position
			}
		}
	}

	// 2. Handle the old song (Remove or Loop)
	if currentSongID != nil {
		room, err := ra.GetRoom(ctx, roomID)
		if err == nil {
			if room.Settings.RemoveOnPlay {
				_ = ra.RemoveSong(ctx, roomID, *currentSongID)
			} else if room.Settings.LoopQueue {
				// Move to the end of the queue
				maxPos, err := ra.GetMaxPosition(ctx, roomID)
				if err == nil {
					_ = ra.ReorderSongs(ctx, roomID, *currentSongID, maxPos+1)
				}
			}
		}
	}

	// 3. Get the next song.
	nextSong, err := ra.GetNextSong(ctx, roomID, currentPosition)
	if err != nil {
		return fmt.Errorf("failed to get next song: %w", err)
	}

	// 4. If we reached the end of the queue and looping is enabled, wrap around
	if nextSong.IsEmpty() {
		room, err := ra.GetRoom(ctx, roomID)
		if err == nil && room.Settings.LoopQueue {
			firstSong, err := ra.GetNextSong(ctx, roomID, -1)
			if err == nil && !firstSong.IsEmpty() {
				nextSong = firstSong
			}
		}
	}

	// 5. Update state
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

func handleVote(ctx context.Context, roomID, userID string, state *vibe.PlaybackState, ra vibe.RoomActioner) error {
	if state.CurrentSongID == nil || userID == "" {
		return nil
	}

	songID := *state.CurrentSongID

	voted, err := ra.HasUserVoted(ctx, roomID, songID, userID)
	if err != nil {
		return fmt.Errorf("failed to check vote: %w", err)
	}

	if voted {
		return nil // Already voted
	}

	err = ra.AddSkipVote(ctx, roomID, songID, userID)
	if err != nil {
		return fmt.Errorf("failed to add vote: %w", err)
	}

	// Check if threshold reached
	room, err := ra.GetRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to fetch room: %w", err)
	}

	votes, err := ra.GetSkipVotes(ctx, roomID, songID)
	if err != nil {
		return fmt.Errorf("failed to fetch votes: %w", err)
	}

	userCount, err := ra.CountUsersInRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if float64(len(votes)) >= float64(userCount)*room.Settings.SkipVoteThreshold {
		err = skipToNextSong(ctx, roomID, state, ra, ra)
		if err != nil {
			return fmt.Errorf("failed to skip after vote: %w", err)
		}
		_ = ra.ClearSkipVotes(ctx, roomID, songID)
	}

	return nil
}
