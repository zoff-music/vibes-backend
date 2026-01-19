package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"
)

// UpdateParticipant updates the last seen time for a participant
func (c *Client) UpdateParticipant(ctx context.Context, roomID, userID string, isActiveListener bool) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UpdateParticipant")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.DB.ExecContext(cctx, `
		INSERT INTO room_participants (room_id, user_id, last_seen_at, is_active_listener)
		VALUES (?1, ?2, ?3, ?4)
		ON CONFLICT(room_id, user_id) DO UPDATE SET last_seen_at = ?3, is_active_listener = ?4
	`, roomID, userID, time.Now(), isActiveListener)
	if err != nil {
		return fmt.Errorf("error updating participant: %w", err)
	}

	return nil
}

// GetActiveParticipants returns a list of participants active in the room within the duration
func (c *Client) GetActiveParticipants(ctx context.Context, roomID string, activeWithin time.Duration) ([]vibe.Participant, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetActiveParticipants")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-activeWithin)

	rows, err := c.DB.QueryContext(cctx, `
		SELECT room_id, user_id, last_seen_at
		FROM room_participants
		WHERE room_id = ?1 AND last_seen_at > ?2
		ORDER BY last_seen_at DESC
	`, roomID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("error querying active participants: %w", err)
	}
	defer rows.Close()

	var participants []vibe.Participant
	for rows.Next() {
		var p vibe.Participant
		err := rows.Scan(&p.RoomID, &p.UserID, &p.LastSeenAt)
		if err != nil {
			return nil, fmt.Errorf("error scanning participant: %w", err)
		}
		participants = append(participants, p)
	}

	return participants, nil
}

// SetRoomHost updates the host for a room
func (c *Client) SetRoomHost(ctx context.Context, roomID, userID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SetRoomHost")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	hostID := sql.NullString{
		String: userID,
		Valid:  userID != "",
	}

	_, err := c.DB.ExecContext(cctx, `
		UPDATE rooms SET host_id = ?1 WHERE id = ?2
	`, hostID, roomID)
	if err != nil {
		return fmt.Errorf("error setting room host: %w", err)
	}

	return nil
}

// RemoveParticipant removes a participant from a room
func (c *Client) RemoveParticipant(ctx context.Context, roomID, userID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RemoveParticipant")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.DB.ExecContext(cctx, `
		DELETE FROM room_participants WHERE room_id = ?1 AND user_id = ?2
	`, roomID, userID)
	if err != nil {
		return fmt.Errorf("error removing participant: %w", err)
	}

	return nil
}

// DeleteInactiveParticipants removes participants who haven't been seen within the duration
func (c *Client) DeleteInactiveParticipants(ctx context.Context, olderThan time.Duration) (int64, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "DeleteInactiveParticipants")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-olderThan)

	result, err := c.DB.ExecContext(cctx, `
		DELETE FROM room_participants WHERE last_seen_at < ?1
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("error deleting inactive participants: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}
