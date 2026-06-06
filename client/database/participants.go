package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/opentracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareUpdateParticipantStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO room_users (
			room_id,
			id,
			last_seen_at,
			is_active_listener,
			is_cast_receiver,
			cast_owner_id
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT(id, room_id) DO UPDATE SET
			last_seen_at = $3,
			is_active_listener = $4,
			is_cast_receiver = $5,
			cast_owner_id = $6
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdateParticipantStatement: %w", err)
	}
	c.UpdateParticipantStatement = stmt
	return nil
}

// UpdateParticipant updates the last seen time for a participant
func (c *Client) UpdateParticipant(
	ctx context.Context,
	roomID string,
	userID string,
	isActiveListener bool,
	isCastReceiver bool,
	castOwnerID string,
) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UpdateParticipant")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.UpdateParticipantStatement.ExecContext(
		cctx,
		roomID,
		userID,
		time.Now(),
		isActiveListener,
		isCastReceiver,
		castOwnerID,
	)
	if err != nil {
		return fmt.Errorf("error updating participant: %w", err)
	}

	return nil
}

func (c *Client) prepareGetActiveParticipantsStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT room_id, id, last_seen_at, is_active_listener, is_cast_receiver, cast_owner_id
		FROM room_users
		WHERE room_id = $1 AND last_seen_at > $2
		ORDER BY last_seen_at DESC
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetActiveParticipantsStatement: %w", err)
	}
	c.GetActiveParticipantsStatement = stmt
	return nil
}

func (c *Client) prepareGetActiveListenerCountsStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			COALESCE(SUM(CASE WHEN is_active_listener = 1 AND is_cast_receiver = 0 THEN 1 ELSE 0 END), 0) as active_listeners,
			COALESCE(SUM(CASE WHEN is_cast_receiver = 1 THEN 1 ELSE 0 END), 0) as active_cast
		FROM room_users
		WHERE room_id = $1 AND last_seen_at > $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetActiveListenerCountsStatement: %w", err)
	}
	c.GetActiveListenerCountsStatement = stmt
	return nil
}

// GetActiveParticipants returns a list of participants active in the room within the duration
func (c *Client) GetActiveParticipants(ctx context.Context, roomID string, activeWithin time.Duration) ([]vibe.Participant, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetActiveParticipants")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-activeWithin)

	rows, err := c.GetActiveParticipantsStatement.QueryContext(cctx, roomID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("error querying active participants: %w", err)
	}
	defer rows.Close()

	var participants []vibe.Participant
	for rows.Next() {
		var row participantRow
		err := row.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning participant: %w", err)
		}
		participants = append(participants, row.toParticipant())
	}

	return participants, nil
}

// GetActiveListenerCounts returns listener counts within the duration.
func (c *Client) GetActiveListenerCounts(ctx context.Context, roomID string, activeWithin time.Duration) (vibe.ListenerCounts, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetActiveListenerCounts")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-activeWithin)

	row := c.GetActiveListenerCountsStatement.QueryRowContext(cctx, roomID, cutoff)

	var listeners int
	var castReceivers int
	err := row.Scan(&listeners, &castReceivers)
	if err != nil {
		return vibe.ListenerCounts{}, fmt.Errorf("error scanning listener counts: %w", err)
	}

	return vibe.ListenerCounts{
		ActiveListeners:     listeners,
		ActiveCastReceivers: castReceivers,
	}, nil
}

type participantRow struct {
	RoomID     string
	UserID     string
	LastSeenAt time.Time
	IsActive   sql.NullInt64
	IsCast     sql.NullInt64
	CastOwner  sql.NullString
}

func (p *participantRow) scan(rows *sql.Rows) error {
	return rows.Scan(
		&p.RoomID,
		&p.UserID,
		&p.LastSeenAt,
		&p.IsActive,
		&p.IsCast,
		&p.CastOwner,
	)
}

func (p *participantRow) toParticipant() vibe.Participant {
	return vibe.Participant{
		RoomID:           p.RoomID,
		UserID:           p.UserID,
		LastSeenAt:       p.LastSeenAt,
		IsActiveListener: p.IsActive.Int64 == 1,
		IsCastReceiver:   p.IsCast.Int64 == 1,
		CastOwnerID:      p.CastOwner.String,
	}
}

func (c *Client) prepareSetRoomHostStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE rooms
		SET host_id = $1
		WHERE id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing SetRoomHostStatement: %w", err)
	}
	c.SetRoomHostStatement = stmt
	return nil
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

	_, err := c.SetRoomHostStatement.ExecContext(cctx, hostID, roomID)
	if err != nil {
		return fmt.Errorf("error setting room host: %w", err)
	}

	return nil
}

func (c *Client) prepareRemoveParticipantStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM room_users WHERE room_id = $1 AND id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing RemoveParticipantStatement: %w", err)
	}
	c.RemoveParticipantStatement = stmt
	return nil
}

// RemoveParticipant removes a participant from a room
func (c *Client) RemoveParticipant(ctx context.Context, roomID, userID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RemoveParticipant")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.RemoveParticipantStatement.ExecContext(cctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("error removing participant: %w", err)
	}

	return nil
}

func (c *Client) prepareDeleteInactiveParticipantsStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM room_users WHERE last_seen_at < $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeleteInactiveParticipantsStatement: %w", err)
	}
	c.DeleteInactiveParticipantsStatement = stmt
	return nil
}

// DeleteInactiveParticipants removes participants who haven't been seen within the duration
func (c *Client) DeleteInactiveParticipants(ctx context.Context, olderThan time.Duration) (int, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "DeleteInactiveParticipants")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-olderThan)

	result, err := c.DeleteInactiveParticipantsStatement.ExecContext(cctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("error deleting inactive participants: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}
