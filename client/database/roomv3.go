package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareGetItemsV3Stmt() error {
	// Resolve contributor names after pagination, not once for every item in the room.
	stmt, err := c.DB.Prepare(`
		WITH items_q AS (
			SELECT
				a.id,
				a.added_at,
				COUNT(b.user_id)::INT AS vote_count,
				MAX(b.created_at) AS last_vote_at
			FROM playlist_items a
			LEFT JOIN playlist_item_votes b
				ON a.id = b.playlist_item_id AND a.room_id = b.room_id
			WHERE a.room_id = $1
				AND a.source_type = ANY($2::TEXT[])
				AND NOT (a.source_type = 'youtube' AND a.duration <= 0)
				AND ($5 = '' OR a.source_type = $5)
				AND (NOT $6 OR a.id IS DISTINCT FROM (
					SELECT current_playlist_item_id FROM playback_state WHERE room_id = $1
				))
			GROUP BY a.id
		), numbered_q AS (
			SELECT
				a.id,
				a.vote_count,
				ROW_NUMBER() OVER (
					ORDER BY a.vote_count DESC, a.last_vote_at ASC, a.added_at ASC, a.id ASC
				) AS row_number
			FROM items_q a
		), totals_q AS (
			SELECT COUNT(*)::INT AS total FROM items_q a
		)
		SELECT
			c.id,
			c.room_id,
			c.source_type,
			c.source_id,
			c.provider_url,
			c.playback_restriction,
			c.title,
			c.publisher,
			c.thumbnail_url,
			c.duration,
			c.added_by,
			COALESCE(d.name, c.added_by_nickname) AS added_by_nickname,
			c.added_at,
			b.vote_count,
			b.row_number::INT,
			a.total
		FROM totals_q a
		LEFT JOIN numbered_q b
			ON ($3 = 0 OR b.row_number >= $3)
			AND ($4 = 0 OR b.row_number <= $4)
		LEFT JOIN playlist_items c ON c.id = b.id
		LEFT JOIN sessions d ON d.id = c.added_by
		ORDER BY b.row_number
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetItemsV3Statement: %w", err)
	}

	c.GetItemsV3Statement = stmt

	return nil
}

func (c *Client) GetItemsV3(ctx context.Context, roomID string, search vibe.ItemSearchV3) (*vibe.ItemResultV3, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetItemsV3")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.GetItemsV3Statement.QueryContext(
		cctx,
		roomID,
		c.enabledProviders,
		search.From,
		search.To,
		search.SourceType,
		search.ExcludeCurrent,
	)
	if err != nil {
		return nil, fmt.Errorf("error fetching items in GetItemsV3: %w", err)
	}
	defer rows.Close()

	result := vibe.ItemResultV3{
		Data:    []vibe.PlaylistItem{},
		ReqFrom: search.From,
		ReqTo:   search.To,
	}

	for rows.Next() {
		var row itemV3Row

		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning item in GetItemsV3: %w", err)
		}

		result.Total = row.Total
		if !row.ID.Valid {
			continue
		}

		item, err := row.toPlaylistItem()
		if err != nil {
			return nil, fmt.Errorf("error mapping item in GetItemsV3: %w", err)
		}

		position := int(row.Position.Int32)
		if len(result.Data) == 0 {
			result.From = position
		}

		result.To = position
		result.Data = append(result.Data, *item)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating items in GetItemsV3: %w", err)
	}

	result.Count = len(result.Data)

	return &result, nil
}

type itemV3Row struct {
	playlistItemRow
	Position sql.NullInt32
	Total    int
}

func (r *itemV3Row) scanRows(rows *sql.Rows) error {
	err := rows.Scan(
		&r.ID,
		&r.RoomID,
		&r.SourceType,
		&r.SourceID,
		&r.ProviderURL,
		&r.PlaybackRestriction,
		&r.Title,
		&r.Publisher,
		&r.ThumbnailURL,
		&r.Duration,
		&r.AddedBySessionID,
		&r.AddedBy,
		&r.AddedAt,
		&r.VoteCount,
		&r.Position,
		&r.Total,
	)
	if err != nil {
		return fmt.Errorf("error scanning item v3 row: %w", err)
	}

	return nil
}

func (c *Client) prepareGetRoomsV3Stmt() error {
	stmt, err := c.DB.Prepare(`
		WITH selected_rooms_q AS (
			SELECT a.id, a.name, a.room_type
			FROM rooms a
			JOIN room_settings b ON b.room_id = a.id
			WHERE NOT a.flagged
			AND (
				(CARDINALITY($1::TEXT[]) > 0 AND a.id = ANY($1::TEXT[]))
				OR (
					CARDINALITY($1::TEXT[]) = 0
					AND b.is_public
					AND a.admin_password_hash IS NOT NULL
					AND a.admin_password_hash != ''
					AND EXISTS (
						SELECT 1 FROM playlist_items c
						WHERE c.room_id = a.id AND c.source_type = ANY($3::TEXT[])
					)
				)
			)
		),
		participants_q AS (
			SELECT
				a.room_id,
				COUNT(*) FILTER (WHERE a.is_active_listener AND NOT a.is_cast_receiver) AS listeners,
				COUNT(*) FILTER (WHERE a.is_cast_receiver) AS receivers
			FROM room_users a
			JOIN selected_rooms_q b ON b.id = a.room_id
			WHERE a.last_seen_at >= $2
			GROUP BY a.room_id
		),
		page_q AS (
			SELECT
				a.id,
				a.name,
				a.room_type,
				CASE
					WHEN b.listeners = 0 AND b.receivers > 0 THEN 1
					ELSE COALESCE(b.listeners, 0)
				END AS listener_count,
				(
					SELECT COUNT(*) FROM playlist_items c
					WHERE c.room_id = a.id AND c.source_type = ANY($3::TEXT[])
				) AS playlist_item_count,
				ARRAY_POSITION($1::TEXT[], a.id) AS request_position
			FROM selected_rooms_q a
			LEFT JOIN participants_q b ON b.room_id = a.id
			ORDER BY request_position, listener_count DESC, playlist_item_count DESC, a.id DESC
			OFFSET $4 LIMIT $5
		),
		totals_q AS (
			SELECT COUNT(*) AS total FROM selected_rooms_q
		)
		SELECT b.id, b.name, b.room_type, b.listener_count, b.playlist_item_count, a.total
		FROM totals_q a
		LEFT JOIN page_q b ON TRUE
		ORDER BY b.request_position, b.listener_count DESC, b.playlist_item_count DESC, b.id DESC
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRoomsV3Statement: %w", err)
	}
	c.GetRoomsV3Statement = stmt
	return nil
}

func (c *Client) GetRoomsV3(ctx context.Context, search vibe.RoomsSearchV3) (*vibe.PublicRoomResultV3, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetRoomsV3")
	defer span.End()
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ids := search.IDs
	if ids == nil {
		ids = make([]string, 0)
	}
	rows, err := c.GetRoomsV3Statement.QueryContext(cctx, ids, time.Now().UTC().Add(-15*time.Second), c.enabledProviders, search.From, search.To-search.From+1)
	if err != nil {
		return nil, fmt.Errorf("error fetching rooms in GetRoomsV3: %w", err)
	}
	defer rows.Close()
	result := vibe.PublicRoomResultV3{Rooms: make([]vibe.PublicRoomV3, 0), From: search.From, To: search.From}
	for rows.Next() {
		var row publicRoomResultRow
		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning room in GetRoomsV3: %w", err)
		}
		result.Total = int(row.Total.Int64)
		if !row.ID.Valid {
			continue
		}
		room, err := row.toPublicRoomV3()
		if err != nil {
			return nil, fmt.Errorf("error mapping room in GetRoomsV3: %w", err)
		}
		result.Rooms = append(result.Rooms, *room)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating rooms in GetRoomsV3: %w", err)
	}
	result.Count = len(result.Rooms)
	if result.Count > 0 {
		result.To = result.From + result.Count - 1
	}
	return &result, nil
}
