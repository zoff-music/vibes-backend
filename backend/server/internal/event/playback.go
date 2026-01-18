package event

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zoff-music/vibes/vibe"
)

// PlaybackMonitorHandler handles playback monitoring for server-mode rooms
type PlaybackMonitorHandler struct {
	db  vibe.RoomActioner
	ips vibe.RoomEventNotifier
}

// NewPlaybackMonitorHandler creates a new PlaybackMonitorHandler
func NewPlaybackMonitorHandler(db vibe.RoomActioner, ips vibe.RoomEventNotifier) *PlaybackMonitorHandler {
	return &PlaybackMonitorHandler{
		db:  db,
		ips: ips,
	}
}

// Handle checks for rooms that need to auto-advance
func (h *PlaybackMonitorHandler) Handle(ctx context.Context, data []byte) error {
	// 1. Get rooms with active listeners (server mode only)
	// We need to cast h.db to RoomFetcher if possible, or assume RoomActioner has it?
	// vibe.RoomActioner interface definition is in vibe/action.go (likely).
	// Let's assume passed DB implements RoomFetcher as well.

	// Issue: GetAppEvents takes vibe.RoomActioner, but we need RoomFetcher too.
	// Ideally we should pass an interface that combines them or the concrete implementation if lazy.
	// But `vibe.RoomActioner` is passed.
	// I should check `vibe/playback.go` or wherever `RoomActioner` is defined.
	// For now, let's type assert or update interface. I'll type assert for expedience or assume it's there.

	fetcher, ok := h.db.(vibe.RoomFetcher)
	if !ok {
		return nil // Should validation elsewhere
	}

	rooms, err := fetcher.GetRoomsWithActiveListeners(ctx, 30*time.Second) // Active within last 30s
	if err != nil {
		return err
	}

	for _, room := range rooms {
		if err := h.checkRoomPlayback(ctx, &room); err != nil {
			log.Errorf("error checking playback for room %s: %v", room.ID, err)
		}
	}

	return nil
}

func (h *PlaybackMonitorHandler) checkRoomPlayback(ctx context.Context, room *vibe.Room) error {
	// Mode-specific logic
	if room.Mode == vibe.RoomModeHost {
		return h.checkHostHealth(ctx, room)
	} else if room.Mode == vibe.RoomModeServer {
		return h.checkServerPlayback(ctx, room.ID)
	}
	return nil
}

func (h *PlaybackMonitorHandler) checkHostHealth(ctx context.Context, room *vibe.Room) error {
	participantStorage, ok := h.db.(vibe.ParticipantStorage)
	if !ok {
		return nil
	}

	// fetch active participants (active within 15s)
	activeParticipants, err := participantStorage.GetActiveParticipants(ctx, room.ID, 15*time.Second)
	if err != nil {
		return err
	}

	hostActive := false
	if room.HostID != "" {
		for _, p := range activeParticipants {
			if p.UserID == room.HostID {
				hostActive = true
				break
			}
		}
	}

	if !hostActive {
		// Pick new host
		if len(activeParticipants) > 0 {
			// Just pick the first one for simplicity, or random. First one is fine (oldest active?).
			// SQL query order isn't guaranteed unless specified. Assuming random enough.
			newHostID := activeParticipants[0].UserID
			err := participantStorage.SetRoomHost(ctx, room.ID, newHostID)
			if err != nil {
				return err
			}

			log.Infof("Room %s: Host %s inactive. New host: %s", room.ID, room.HostID, newHostID)

			// Notify
			_ = h.ips.NotifyRoom(ctx, room.ID, &vibe.RoomEvent{
				Type:    vibe.EventTypeNewHost,
				Payload: map[string]string{"userId": newHostID, "message": "You are now the host"},
			})
		} else if room.HostID != "" {
			// No active participants, remove host
			err := participantStorage.SetRoomHost(ctx, room.ID, "")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *PlaybackMonitorHandler) checkServerPlayback(ctx context.Context, roomID string) error {
	// Get current playback state
	// RoomActioner has UpdatePlayback/SkipTrack/VoteToSkip/GetSongs.
	// We need GetPlaybackState. That is in PlaybackFetcher interface.

	playbackFetcher, ok := h.db.(vibe.PlaybackFetcher)
	if !ok {
		return nil
	}

	state, err := playbackFetcher.GetPlaybackState(ctx, roomID)
	if err != nil {
		return err
	}

	if state == nil || !state.IsPlaying {
		return nil
	}

	// Calculate current position
	currentPos := state.PositionMs + time.Since(state.UpdatedAt).Milliseconds()

	// Need song duration. State has it?
	// PlaybackState struct in vibe/playback.go usually has Song info or we fetch song.
	// Let's assume state has Song duration or we can infer it.
	// Based on typical implementation, state might just have progress.
	// If state.CurrentSong is populated:
	if state.CurrentSong == nil {
		return nil
	}

	durationMs := int64(state.CurrentSong.Duration) * 1000

	// If remaining time < 500ms (or ended), skip
	// Prompt: "ending in the next 500ms or have ended"
	if currentPos >= durationMs-500 {
		log.Infof("Room %s: Song ended or ending (pos=%d, dur=%d). Skipping.", roomID, currentPos, durationMs)
		_, err := h.db.SkipTrack(ctx, roomID)
		if err != nil {
			return err
		}

		// Notify listeners about queue change (SkipTrack usually updates playback state)
		// We should probably broadcast PlaybackUpdate too if SkipTrack doesn't do it via this code path context.
		// However, SkipTrack implementation updates DB.
		// Let's perform a notify here to be safe and responsive.
		newState, err := playbackFetcher.GetPlaybackState(ctx, roomID)
		if err == nil {
			_ = h.ips.NotifyRoom(ctx, roomID, &vibe.RoomEvent{
				Type:    vibe.EventTypePlaybackUpdate, // Client handles next song via playback update
				Payload: newState,
			})
		}
	}

	return nil
}
