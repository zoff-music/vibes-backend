package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareSearchPublicRoomsStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH public_rooms_q AS (
			SELECT a.id, a.name
			FROM rooms a
			JOIN room_settings b ON b.room_id = a.id
			WHERE b.is_public
			AND a.admin_password_hash IS NOT NULL
			AND a.admin_password_hash != ''
			AND ($1 = '' OR a.name ILIKE '%' || $1 || '%')
		),
		participants_q AS (
			SELECT
				a.room_id,
				COUNT(*) FILTER (WHERE a.is_active_listener AND NOT a.is_cast_receiver) AS listeners,
				COUNT(*) FILTER (WHERE a.is_cast_receiver) AS receivers
			FROM room_users a
			JOIN public_rooms_q b ON b.id = a.room_id
			WHERE a.last_seen_at >= $2
			GROUP BY a.room_id
		),
		listeners_q AS (
			SELECT
				a.id,
				a.name,
				CASE
					WHEN b.listeners = 0 AND b.receivers > 0 THEN 1
					ELSE COALESCE(b.listeners, 0)
				END AS listener_count
			FROM public_rooms_q a
			LEFT JOIN participants_q b ON b.room_id = a.id
		),
		filtered_q AS (
			SELECT a.id, a.name, a.listener_count
			FROM listeners_q a
			WHERE NOT $3 OR a.listener_count > 0
		),
		page_q AS (
			SELECT
				a.id,
				a.name,
				a.listener_count,
				(
					SELECT COUNT(*)
					FROM songs b
					WHERE b.room_id = a.id
					AND b.source_type = ANY($4::text[])
				) AS song_count
			FROM filtered_q a
			ORDER BY a.listener_count DESC, song_count DESC, a.id DESC
			OFFSET $5 LIMIT $6
		),
		totals_q AS (
			SELECT COUNT(*) AS total FROM filtered_q
		)
		SELECT b.id, b.name, b.listener_count, b.song_count, a.total
		FROM totals_q a
		LEFT JOIN page_q b ON TRUE
		ORDER BY b.listener_count DESC, b.song_count DESC, b.id DESC
	`)
	if err != nil {
		return fmt.Errorf("error preparing SearchPublicRoomsStatement: %w", err)
	}

	c.SearchPublicRoomsStatement = stmt

	return nil
}

func (c *Client) SearchPublicRooms(
	ctx context.Context,
	search vibe.PublicRoomSearch,
) (*vibe.PublicRoomResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SearchPublicRooms")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Treat LIKE metacharacters as part of the room name.
	query := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search.Query)
	rows, err := c.SearchPublicRoomsStatement.QueryContext(
		cctx,
		query,
		time.Now().UTC().Add(-15*time.Second),
		search.Live,
		c.enabledProviders,
		search.From,
		search.To-search.From+1,
	)
	if err != nil {
		return nil, fmt.Errorf("error searching public rooms: %w", err)
	}

	defer rows.Close()

	result := &vibe.PublicRoomResult{
		Rooms: []vibe.PublicRoom{},
		From:  search.From,
		To:    search.From,
	}

	for rows.Next() {
		var row publicRoomResultRow
		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning public room result: %w", err)
		}

		result.Total = int(row.Total.Int64)
		if !row.ID.Valid {
			continue
		}

		room, err := row.toPublicRoom()
		if err != nil {
			return nil, fmt.Errorf("error converting public room in SearchPublicRooms: %w", err)
		}

		result.Rooms = append(result.Rooms, *room)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating public room results: %w", err)
	}

	result.Count = len(result.Rooms)
	if result.Count > 0 {
		result.To = result.From + result.Count - 1
	}

	return result, nil
}

type publicRoomResultRow struct {
	publicRoomRow
	Total sql.NullInt64
}

func (r *publicRoomResultRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(&r.ID, &r.Name, &r.ListenerCount, &r.SongCount, &r.Total)
	if err != nil {
		return fmt.Errorf("error scanning public room result row: %w", err)
	}

	return nil
}
