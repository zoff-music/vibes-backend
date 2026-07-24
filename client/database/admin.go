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

func (c *Client) prepareGetAdminRoomsStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH active_users_q AS (
			SELECT
				a.room_id,
				COUNT(*) AS user_count
			FROM room_users a
			WHERE a.is_active_listener
			AND a.last_seen_at >= $1
			GROUP BY a.room_id
		),
		song_counts_q AS (
			SELECT
				a.room_id,
				COUNT(*) AS song_count,
				STRING_AGG(DISTINCT a.source_type, ',') AS active_sources
			FROM songs a
			GROUP BY a.room_id
		)
		SELECT
			a.id,
			a.name,
			COALESCE(b.user_count, 0) AS user_count,
			COALESCE(c.song_count, 0) AS song_count,
			COALESCE(c.active_sources, '') AS active_sources,
			(a.admin_password_hash IS NOT NULL AND a.admin_password_hash != '') AS has_admin_password
		FROM rooms a
		LEFT JOIN active_users_q b ON b.room_id = a.id
		LEFT JOIN song_counts_q c ON c.room_id = a.id
		ORDER BY user_count DESC, song_count DESC
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetAdminRoomsStatement: %w", err)
	}

	c.GetAdminRoomsStatement = stmt
	return nil
}

func (c *Client) ListAdminRooms(ctx context.Context) ([]vibe.AdminRoomSummary, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ListAdminRooms")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-15 * time.Second)
	rows, err := c.GetAdminRoomsStatement.QueryContext(cctx, cutoff)
	if err != nil {
		return nil, fmt.Errorf("error fetching admin rooms: %w", err)
	}
	defer rows.Close()

	rooms := []vibe.AdminRoomSummary{}
	for rows.Next() {
		var row adminRoomRow
		err := row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning admin room: %w", err)
		}

		rooms = append(rooms, row.toSummary())
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating admin rooms: %w", err)
	}

	return rooms, nil
}

type adminRoomRow struct {
	ID               sql.NullString
	Name             sql.NullString
	UserCount        sql.NullInt64
	SongCount        sql.NullInt64
	ActiveSources    sql.NullString
	HasAdminPassword sql.NullBool
}

func (r *adminRoomRow) scanRows(rows *sql.Rows) error {
	return rows.Scan(
		&r.ID,
		&r.Name,
		&r.UserCount,
		&r.SongCount,
		&r.ActiveSources,
		&r.HasAdminPassword,
	)
}

func (r *adminRoomRow) toSummary() vibe.AdminRoomSummary {
	sources := []string{}
	if r.ActiveSources.Valid && r.ActiveSources.String != "" {
		sources = strings.Split(r.ActiveSources.String, ",")
	}

	return vibe.AdminRoomSummary{
		ID:               r.ID.String,
		Name:             r.Name.String,
		UserCount:        int(r.UserCount.Int64),
		SongCount:        int(r.SongCount.Int64),
		ActiveSources:    sources,
		HasAdminPassword: r.HasAdminPassword.Bool,
	}
}

func (c *Client) prepareUpdateAdminRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE rooms a
		SET name = COALESCE($2::text, a.name),
		admin_password_hash = CASE
			WHEN $3::boolean THEN NULL
			ELSE a.admin_password_hash
		END
		WHERE a.id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdateAdminRoomStatement: %w", err)
	}

	c.UpdateAdminRoomStatement = stmt
	return nil
}

func (c *Client) UpdateAdminRoom(ctx context.Context, roomID string, request vibe.AdminUpdateRoomRequest) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdateAdminRoom")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var name *string
	if request.Name != nil {
		name = request.Name
	}

	clearAdminPassword := false
	if request.ClearAdminPassword != nil {
		clearAdminPassword = *request.ClearAdminPassword
	}

	result, err := c.UpdateAdminRoomStatement.ExecContext(
		cctx,
		roomID,
		name,
		clearAdminPassword,
	)
	if err != nil {
		return false, fmt.Errorf("error updating admin room: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error getting updated admin room rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (c *Client) prepareDeleteAdminRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH deleted_generation_q AS (
			DELETE FROM room_generations
			WHERE room_id = $1
		)
		DELETE FROM rooms a
		WHERE a.id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeleteAdminRoomStatement: %w", err)
	}

	c.DeleteAdminRoomStatement = stmt
	return nil
}

func (c *Client) DeleteAdminRoom(ctx context.Context, roomID string) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "DeleteAdminRoom")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.DeleteAdminRoomStatement.ExecContext(cctx, roomID)
	if err != nil {
		return false, fmt.Errorf("error deleting admin room: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error getting deleted admin room rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}
