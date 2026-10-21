package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareGetNextRoomNameReviewStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH candidate_q AS (
			SELECT a.id
			FROM room_reviews a
			WHERE (
				a.started_at IS NULL
				OR a.started_at < NOW() - INTERVAL '1 MINUTE'
			)
			AND EXISTS (
				SELECT 1
				FROM rooms b
				WHERE b.checked_at IS NULL
			)
			FOR UPDATE OF a SKIP LOCKED
		),
		claimed_q AS (
			UPDATE room_reviews a
			SET started_at = NOW()
			FROM candidate_q b
			WHERE a.id = b.id
			RETURNING a.started_at
		)
		SELECT
			a.started_at,
			b.id,
			b.name
		FROM claimed_q a
		CROSS JOIN rooms b
		WHERE b.checked_at IS NULL
		ORDER BY b.created_at, b.id
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetNextRoomNameReviewStatement: %w", err)
	}

	c.GetNextRoomNameReviewStatement = stmt

	return nil
}

func (c *Client) GetNextRoomNameReview(ctx context.Context) (*vibe.RoomNameReview, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetNextRoomNameReview")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.GetNextRoomNameReviewStatement.QueryContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error querying room name review: %w", err)
	}
	defer rows.Close()

	review := &vibe.RoomNameReview{}
	for rows.Next() {
		var row roomNameReviewRow
		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning room name review: %w", err)
		}

		room, err := row.toRoomNameDecision()
		if err != nil {
			return nil, fmt.Errorf("error mapping room name review: %w", err)
		}

		review.StartedAt = row.StartedAt.Time
		review.Rooms = append(review.Rooms, *room)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating room name review: %w", err)
	}

	if len(review.Rooms) == 0 {
		return nil, internalerror.ErrExpected{
			Err: fmt.Errorf("no rooms requiring review"),
		}
	}

	return review, nil
}

type roomNameReviewRow struct {
	StartedAt sql.NullTime
	ID        sql.NullString
	Name      sql.NullString
}

func (r *roomNameReviewRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(
		&r.StartedAt,
		&r.ID,
		&r.Name,
	)
	if err != nil {
		return fmt.Errorf("error scanning room name review row: %w", err)
	}

	return nil
}

func (r *roomNameReviewRow) toRoomNameDecision() (*vibe.RoomNameDecision, error) {
	return &vibe.RoomNameDecision{
		ID:   r.ID.String,
		Name: r.Name.String,
	}, nil
}

func (c *Client) prepareCompleteRoomNameReviewStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH updated_q AS (
			UPDATE rooms a
			SET flagged = a.id = ANY($3::TEXT[]),
				flagged_at = CASE
					WHEN a.id = ANY($3::TEXT[]) THEN COALESCE(a.flagged_at, NOW())
					ELSE NULL
				END,
				checked_at = NOW()
			WHERE a.id = ANY($2::TEXT[])
			AND a.checked_at IS NULL
			AND EXISTS (
				SELECT 1
				FROM room_reviews b
				WHERE b.started_at = $1
			)
			RETURNING
				a.id,
				a.flagged
		)
		UPDATE room_settings a
		SET is_public = FALSE
		FROM updated_q b
		WHERE a.room_id = b.id
		AND b.flagged
	`)
	if err != nil {
		return fmt.Errorf("error preparing CompleteRoomNameReviewStatement: %w", err)
	}

	c.CompleteRoomNameReviewStatement = stmt

	return nil
}

func (c *Client) CompleteRoomNameReview(ctx context.Context, review *vibe.RoomNameReview) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "CompleteRoomNameReview")
	defer span.End()

	ids := make([]string, 0, len(review.Rooms))
	flaggedIDs := make([]string, 0, len(review.Rooms))
	for _, room := range review.Rooms {
		ids = append(ids, room.ID)
		if room.Flagged {
			flaggedIDs = append(flaggedIDs, room.ID)
		}
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.CompleteRoomNameReviewStatement.ExecContext(
		cctx,
		review.StartedAt,
		ids,
		flaggedIDs,
	)
	if err != nil {
		return fmt.Errorf("error completing room name review: %w", err)
	}

	return nil
}
