package database

import (
	"context"
	"fmt"
	"time"

	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"
)

// GetRoomsWithActiveListeners fetches rooms that have active listeners and are in server mode.
func (c *Client) GetRoomsWithActiveListeners(ctx context.Context, activeWithin time.Duration) ([]vibe.Room, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetRoomsWithActiveListeners")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-activeWithin)

	rows, err := c.DB.QueryContext(cctx, `
		SELECT DISTINCT
			rooms.id,
			rooms.name,
			rooms.mode,
			rooms.host_id,
			rooms.admin_password_hash,
			rooms.created_at,
			room_settings.skip_allowed,
			room_settings.democratic_skip,
			room_settings.skip_vote_threshold,
			room_settings.max_continuous_adds,
			room_settings.remove_on_play,
			room_settings.loop_queue,
			room_settings.allow_duplicates
		FROM rooms
		JOIN room_settings ON room_settings.room_id = rooms.id
		JOIN room_participants ON room_participants.room_id = rooms.id
		WHERE (rooms.mode = 'server' OR rooms.mode = 'host')
		  AND room_participants.is_active_listener = 1
		  AND room_participants.last_seen_at > ?1
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("error querying active rooms: %w", err)
	}
	defer rows.Close()

	var rooms []vibe.Room
	for rows.Next() {
		var r roomRow
		err := r.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning active room: %w", err)
		}

		room, err := r.toRoom()
		if err != nil {
			return nil, fmt.Errorf("error converting room row: %w", err)
		}
		rooms = append(rooms, *room)
	}

	return rooms, nil
}
