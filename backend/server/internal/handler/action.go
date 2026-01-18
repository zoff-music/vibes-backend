package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/zoff-music/vibes/server/internal/middleware"
	"github.com/zoff-music/vibes/vibe"
)

// RoomAction handles POST /api/v1/rooms/:id
func RoomAction(
	db vibe.RoomActioner,
	ips vibe.RoomEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.RoomActionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("invalid request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		session, ok := ctx.Value(middleware.SessionKey).(middleware.SessionPayload)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}
		userID := session.UserID

		var state *vibe.PlaybackState
		// Fetch room to check mode
		room, err := db.GetRoom(ctx, roomID)
		if err != nil {
			handleError(w, fmt.Errorf("failed to fetch room: %w", err), http.StatusInternalServerError, true)
			return
		}

		switch req.Action {
		case vibe.RoomActionPlay, vibe.RoomActionPause, vibe.RoomActionSeek, vibe.RoomActionSkip:
			log.Infof("Room %s: User %s attempting %s (room.Mode=%s, room.HostID=%s)", roomID, userID, req.Action, room.Mode, room.HostID)
			// Register this user as an active participant so they can claim host
			if ps, ok := db.(vibe.ParticipantStorage); ok {
				_ = ps.UpdateParticipant(ctx, roomID, userID, true)
			}
			// Enforce host permissions for these actions in Host Mode
			if room.Mode == vibe.RoomModeHost {
				if room.HostID == "" {
					// No host yet, this user becomes the host
					ps, ok := db.(vibe.ParticipantStorage)
					if ok {
						err := ps.SetRoomHost(ctx, roomID, userID)
						if err != nil {
							log.Errorf("Failed to set room host: %v", err)
						} else {
							log.Infof("Room %s: Set new host to %s", roomID, userID)
							room.HostID = userID
						}
					} else {
						log.Warn("db does not implement ParticipantStorage, cannot set host")
					}
				} else if room.HostID != userID {
					// Check if current host is still active before denying
					ps, ok := db.(vibe.ParticipantStorage)
					hostStillActive := true
					if ok {
						activeParticipants, err := ps.GetActiveParticipants(ctx, roomID, 30*time.Second)
						if err == nil {
							hostStillActive = false
							for _, p := range activeParticipants {
								if p.UserID == room.HostID {
									hostStillActive = true
									break
								}
							}
							if !hostStillActive {
								// Current host is inactive, make requester the new host
								err := ps.SetRoomHost(ctx, roomID, userID)
								if err != nil {
									log.Errorf("Failed to set room host: %v", err)
								} else {
									log.Infof("Room %s: Host %s inactive, new host: %s", roomID, room.HostID, userID)
									room.HostID = userID
								}
							}
						}
					}
					// Re-check after potential host change
					if room.HostID != userID {
						handleError(w, fmt.Errorf("only the host can perform this action"), http.StatusForbidden, false)
						return
					}
				}
			}

			if req.Action == vibe.RoomActionSkip {
				state, err = db.SkipTrack(ctx, roomID)
			} else {
				state, err = db.UpdatePlayback(ctx, roomID, req.Action, req.PositionMs)
			}
		case vibe.RoomActionVote:
			// Vote is allowed for everyone?
			// Prompt: "Host mode is when... actions are reflected... Admin pauses... Everyone... eligible to become host"
			// Doesn't explicitly say normal users can't vote to skip.
			// Usually voting is democratic. But skipping (force skip) is host action.
			// Let's assume Vote is allowed, but Skip (force) is host only.
			// My switch block handles Skip and Vote separately.
			// The host check above blocks ALL actions including Vote if I place it here.
			// I should probably allow Vote.
			// Let's move the check inside the switch or before relevant cases.
			state, err = db.VoteToSkip(ctx, roomID, userID)
		default:
			handleError(
				w,
				fmt.Errorf("invalid action: %s", req.Action),
				http.StatusBadRequest,
				false,
			)
			return
		}

		if err != nil {
			handleError(
				w,
				fmt.Errorf("action %s failed: %w", req.Action, err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		// Update server time for response
		state.ServerTimeMs = time.Now().UnixMilli()

		// Handle skip or vote actions
		if req.Action == vibe.RoomActionSkip || req.Action == vibe.RoomActionVote {
			// If skip was requested, we broadcast the playback update
			// but also potentially queue update if loop/remove logic is handled in SkipTrack
			// The current SkipTrack implementation in playback.go handles queue modification.
			// So we should broadcast queue update as well.
			// Vote might invoke SkipTrack internally if threshold is met.
			songs, err := db.GetSongs(ctx, roomID)
			if err == nil {
				_ = ips.NotifyRoom(ctx, roomID, &vibe.RoomEvent{
					Type:    vibe.EventTypeQueueReordered,
					Payload: songs,
				})
			}
		}

		err = ips.NotifyRoom(ctx, roomID, &vibe.RoomEvent{
			Type:    vibe.EventTypePlaybackUpdate,
			Payload: state,
		})
		if err != nil {
			log.Errorf("failed to notify room: %v", err)
		}

		body, err := json.Marshal(state)
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
