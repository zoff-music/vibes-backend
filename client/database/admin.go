package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareSearchAdminRoomsV2Stmt() error {
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
		playlist_item_counts_q AS (
			SELECT
				a.room_id,
				COUNT(*) AS playlist_item_count,
				STRING_AGG(DISTINCT a.source_type, ',') AS active_sources
			FROM playlist_items a
			WHERE a.source_type = ANY($2::text[])
			GROUP BY a.room_id
		),
		data_q AS (
			SELECT
				a.id,
				a.name,
				a.room_type,
				COALESCE(d.is_public, FALSE) AS is_public,
				COALESCE(b.user_count, 0) AS user_count,
				COALESCE(c.playlist_item_count, 0) AS playlist_item_count,
				COALESCE(c.active_sources, '') AS active_sources,
				(a.admin_password_hash IS NOT NULL AND a.admin_password_hash != '') AS has_admin_password
			FROM rooms a
			LEFT JOIN active_users_q b ON b.room_id = a.id
			LEFT JOIN playlist_item_counts_q c ON c.room_id = a.id
			LEFT JOIN room_settings d ON d.room_id = a.id
			WHERE $3 = ''
			OR a.name ILIKE '%' || $3 || '%'
		),
		numbered_q AS (
			SELECT
				a.*,
				COUNT(*) OVER() AS total_count,
				ROW_NUMBER() OVER (
					ORDER BY
						CASE WHEN $4 = 'listeners' AND $5 THEN a.user_count END DESC,
						CASE WHEN $4 = 'listeners' AND $5 THEN a.playlist_item_count END DESC,
						CASE WHEN $4 = 'listeners' AND NOT $5 THEN a.user_count END ASC,
						CASE WHEN $4 = 'listeners' AND NOT $5 THEN a.playlist_item_count END ASC,
						CASE WHEN $4 = 'playlistItems' AND $5 THEN a.playlist_item_count END DESC,
						CASE WHEN $4 = 'playlistItems' AND $5 THEN a.user_count END DESC,
						CASE WHEN $4 = 'playlistItems' AND NOT $5 THEN a.playlist_item_count END ASC,
						CASE WHEN $4 = 'playlistItems' AND NOT $5 THEN a.user_count END ASC,
						a.name ASC,
						a.id ASC
				) AS row_number
			FROM data_q a
		)
		SELECT
			a.id,
			a.name,
			a.room_type,
			a.is_public,
			a.user_count,
			a.playlist_item_count,
			a.active_sources,
			a.has_admin_password,
			a.total_count
		FROM numbered_q a
		WHERE a.row_number >= $6
		AND a.row_number <= $7
		ORDER BY a.row_number
	`)
	if err != nil {
		return fmt.Errorf("error preparing SearchAdminRoomsV2Statement: %w", err)
	}

	c.SearchAdminRoomsV2Statement = stmt
	return nil
}

func (c *Client) ListAdminRoomsV2(ctx context.Context) ([]vibe.AdminRoomSummaryV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ListAdminRoomsV2")
	defer span.End()

	result, err := c.SearchAdminRoomsV2(ctx, vibe.AdminRoomSearchV2{
		SortBy:     vibe.AdminRoomSortListenersV2,
		Descending: true,
		From:       0,
		To:         adminRoomsMaximumRow,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing admin rooms in ListAdminRoomsV2: %w", err)
	}

	return result.Rooms, nil
}

func (c *Client) SearchAdminRoomsV2(
	ctx context.Context,
	search vibe.AdminRoomSearchV2,
) (*vibe.AdminRoomResultV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SearchAdminRoomsV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-15 * time.Second)
	rows, err := c.SearchAdminRoomsV2Statement.QueryContext(
		cctx,
		cutoff,
		c.enabledProviders,
		search.Query,
		search.SortBy,
		search.Descending,
		search.From+1,
		search.To+1,
	)
	if err != nil {
		return nil, fmt.Errorf("error searching admin rooms: %w", err)
	}
	defer rows.Close()

	rooms := []vibe.AdminRoomSummaryV2{}
	total := 0
	for rows.Next() {
		var row adminRoomRow
		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning admin room: %w", err)
		}

		total = int(row.TotalCount.Int64)
		summary, err := row.toSummary()
		if err != nil {
			return nil, fmt.Errorf("error mapping summary: %w", err)
		}

		rooms = append(rooms, *summary)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating admin rooms: %w", err)
	}

	count := len(rooms)
	actualTo := search.From
	if count > 0 {
		actualTo = search.From + count - 1
	}

	return &vibe.AdminRoomResultV2{
		Rooms: rooms,
		From:  search.From,
		To:    actualTo,
		Total: total,
		Count: count,
	}, nil
}

type adminRoomRow struct {
	ID                sql.NullString
	Name              sql.NullString
	RoomType          sql.NullString
	IsPublic          sql.NullBool
	UserCount         sql.NullInt64
	PlaylistItemCount sql.NullInt64
	ActiveSources     sql.NullString
	HasAdminPassword  sql.NullBool
	TotalCount        sql.NullInt64
}

func (r *adminRoomRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(
		&r.ID,
		&r.Name,
		&r.RoomType,
		&r.IsPublic,
		&r.UserCount,
		&r.PlaylistItemCount,
		&r.ActiveSources,
		&r.HasAdminPassword,
		&r.TotalCount,
	)
	if err != nil {
		return fmt.Errorf("error scanning admin room row: %w", err)
	}

	return nil
}

func (r *adminRoomRow) toSummary() (*vibe.AdminRoomSummaryV2, error) {
	sources := []string{}
	if r.ActiveSources.Valid && r.ActiveSources.String != "" {
		sources = strings.Split(r.ActiveSources.String, ",")
	}

	return &vibe.AdminRoomSummaryV2{
		ID:                r.ID.String,
		Name:              r.Name.String,
		RoomType:          vibe.RoomType(r.RoomType.String),
		IsPublic:          r.IsPublic.Bool,
		UserCount:         int(r.UserCount.Int64),
		PlaylistItemCount: int(r.PlaylistItemCount.Int64),
		ActiveSources:     sources,
		HasAdminPassword:  r.HasAdminPassword.Bool,
	}, nil
}

func (c *Client) prepareUpdateAdminRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH updated_room_q AS (
			UPDATE rooms a
			SET name = COALESCE($2::text, a.name),
			admin_password_hash = CASE
				WHEN $3::boolean THEN NULL
				ELSE a.admin_password_hash
			END
			WHERE a.id = $1
			RETURNING a.id
		)
		UPDATE room_settings a
		SET is_public = CASE
			WHEN $3::boolean THEN FALSE
			ELSE a.is_public
		END
		FROM updated_room_q b
		WHERE a.room_id = b.id
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
		WITH deleted_skip_votes_q AS (
			DELETE FROM skip_votes
			WHERE room_id = $1
		),
		deleted_playlist_item_votes_q AS (
			DELETE FROM playlist_item_votes
			WHERE room_id = $1
		),
		deleted_playback_state_q AS (
			DELETE FROM playback_state
			WHERE room_id = $1
		),
		deleted_room_settings_q AS (
			DELETE FROM room_settings
			WHERE room_id = $1
		),
		deleted_room_users_q AS (
			DELETE FROM room_users
			WHERE room_id = $1
		),
		deleted_playlist_items_q AS (
			DELETE FROM playlist_items
			WHERE room_id = $1
		),
		deleted_generation_q AS (
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

const adminRoomsMaximumRow = 2147483647

func (c *Client) prepareGetAdminUserStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			id,
			username,
			password_hash,
			session_version,
			created_at,
			updated_at
		FROM admin_users
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetAdminUserStatement: %w", err)
	}

	c.GetAdminUserStatement = stmt

	return nil
}

func (c *Client) GetAdminUser(
	ctx context.Context,
	adminID string,
) (*vibe.AdminUser, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetAdminUser")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetAdminUserStatement.QueryRowContext(cctx, adminID)
	var rowData adminUserRow
	err := rowData.scanRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.AdminUser{}, nil
		}

		return nil, fmt.Errorf(
			"error scanning admin user in GetAdminUser: %w",
			err,
		)
	}

	admin, err := rowData.toAdminUser()
	if err != nil {
		return nil, fmt.Errorf("error mapping admin: %w", err)
	}

	return admin, nil
}

type adminUserRow struct {
	ID             string
	Username       string
	PasswordHash   string
	SessionVersion int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (r *adminUserRow) scanRow(row *sql.Row) error {
	err := row.Scan(
		&r.ID,
		&r.Username,
		&r.PasswordHash,
		&r.SessionVersion,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error scanning admin user row: %w", err)
	}

	return nil
}

func (r *adminUserRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(
		&r.ID,
		&r.Username,
		&r.PasswordHash,
		&r.SessionVersion,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error scanning admin user rows: %w", err)
	}

	return nil
}

func (r *adminUserRow) toAdminUser() (*vibe.AdminUser, error) {
	return &vibe.AdminUser{
		ID:             r.ID,
		Username:       r.Username,
		PasswordHash:   r.PasswordHash,
		SessionVersion: r.SessionVersion,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}, nil
}

func (c *Client) prepareGetAdminUserByUsernameStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			id,
			username,
			password_hash,
			session_version,
			created_at,
			updated_at
		FROM admin_users
		WHERE username = $1
	`)
	if err != nil {
		return fmt.Errorf(
			"error preparing GetAdminUserByUsernameStatement: %w",
			err,
		)
	}

	c.GetAdminUserByUsernameStatement = stmt

	return nil
}

func (c *Client) GetAdminUserByUsername(
	ctx context.Context,
	username string,
) (*vibe.AdminUser, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetAdminUserByUsername")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetAdminUserByUsernameStatement.QueryRowContext(cctx, username)
	var rowData adminUserRow
	err := rowData.scanRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.AdminUser{}, nil
		}

		return nil, fmt.Errorf(
			"error scanning admin user in GetAdminUserByUsername: %w",
			err,
		)
	}

	admin, err := rowData.toAdminUser()
	if err != nil {
		return nil, fmt.Errorf("error mapping admin: %w", err)
	}

	return admin, nil
}

func (c *Client) prepareListAdminUsersStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			id,
			username,
			password_hash,
			session_version,
			created_at,
			updated_at
		FROM admin_users
		ORDER BY username
	`)
	if err != nil {
		return fmt.Errorf("error preparing ListAdminUsersStatement: %w", err)
	}

	c.ListAdminUsersStatement = stmt

	return nil
}

func (c *Client) ListAdminUsers(
	ctx context.Context,
) ([]vibe.AdminUser, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ListAdminUsers")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.ListAdminUsersStatement.QueryContext(cctx)
	if err != nil {
		return nil, fmt.Errorf(
			"error querying admin users in ListAdminUsers: %w",
			err,
		)
	}
	defer rows.Close()

	users := make([]vibe.AdminUser, 0)
	for rows.Next() {
		var rowData adminUserRow
		err = rowData.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"error scanning admin users in ListAdminUsers: %w",
				err,
			)
		}

		admin, err := rowData.toAdminUser()
		if err != nil {
			return nil, fmt.Errorf("error mapping admin: %w", err)
		}

		users = append(users, *admin)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf(
			"error iterating admin users in ListAdminUsers: %w",
			err,
		)
	}

	return users, nil
}

func (c *Client) prepareCreateAdminUserStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO admin_users (
			id,
			username,
			password_hash
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			username,
			password_hash,
			session_version,
			created_at,
			updated_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateAdminUserStatement: %w", err)
	}

	c.CreateAdminUserStatement = stmt

	return nil
}

func (c *Client) CreateAdminUser(
	ctx context.Context,
	user vibe.AdminUser,
) (*vibe.AdminUser, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateAdminUser")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.CreateAdminUserStatement.QueryRowContext(
		cctx,
		user.ID,
		user.Username,
		user.PasswordHash,
	)
	var rowData adminUserRow
	err := rowData.scanRow(row)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.ConstraintName == adminUsernameConstraint {
			return nil, internalerror.ErrAdminUsernameUnavailable{
				Err: fmt.Errorf(
					"error creating admin user in CreateAdminUser: %w",
					err,
				),
			}
		}

		return nil, fmt.Errorf(
			"error scanning admin user in CreateAdminUser: %w",
			err,
		)
	}

	admin, err := rowData.toAdminUser()
	if err != nil {
		return nil, fmt.Errorf("error mapping admin: %w", err)
	}

	return admin, nil
}

func (c *Client) prepareUpdateAdminUserPasswordStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE admin_users
		SET
			password_hash = $2,
			session_version = session_version + 1,
			updated_at = NOW()
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf(
			"error preparing UpdateAdminUserPasswordStatement: %w",
			err,
		)
	}

	c.UpdateAdminUserPasswordStatement = stmt

	return nil
}

func (c *Client) UpdateAdminUserPassword(
	ctx context.Context,
	adminID string,
	passwordHash string,
) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdateAdminUserPassword")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.UpdateAdminUserPasswordStatement.ExecContext(
		cctx,
		adminID,
		passwordHash,
	)
	if err != nil {
		return false, fmt.Errorf(
			"error updating admin password in UpdateAdminUserPassword: %w",
			err,
		)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf(
			"error getting affected rows in UpdateAdminUserPassword: %w",
			err,
		)
	}

	return affected > 0, nil
}

func (c *Client) prepareDeleteAdminUserStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM admin_users
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeleteAdminUserStatement: %w", err)
	}

	c.DeleteAdminUserStatement = stmt

	return nil
}

func (c *Client) DeleteAdminUser(
	ctx context.Context,
	adminID string,
) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "DeleteAdminUser")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.DeleteAdminUserStatement.ExecContext(cctx, adminID)
	if err != nil {
		return false, fmt.Errorf(
			"error deleting admin user in DeleteAdminUser: %w",
			err,
		)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf(
			"error getting affected rows in DeleteAdminUser: %w",
			err,
		)
	}

	return affected > 0, nil
}

const adminUsernameConstraint = "admin_users_username_key"

func (c *Client) prepareListAdminRoomSearchUsageStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH windows_q (period, starts_at) AS (
			VALUES
				('hour', DATE_TRUNC('hour', NOW(), 'UTC') - INTERVAL '23 hours'),
				('day', DATE_TRUNC('day', NOW(), 'UTC') - INTERVAL '29 days')
		)
		SELECT
			w.period,
			DATE_TRUNC(w.period, u.created_at, 'UTC'),
			u.room_id,
			u.provider,
			SUM(u.search_count),
			SUM(u.cached_count),
			r.room_type
		FROM windows_q w
		JOIN room_search_usage u ON u.created_at >= w.starts_at
		LEFT JOIN rooms r ON r.id = u.room_id
		GROUP BY
			w.period,
			DATE_TRUNC(w.period, u.created_at, 'UTC'),
			u.room_id,
			u.provider,
			r.room_type
		ORDER BY 1, 2, 3, 4
	`)
	if err != nil {
		return fmt.Errorf("error preparing ListAdminRoomSearchUsageStatement: %w", err)
	}

	c.ListAdminRoomSearchUsageStatement = stmt

	return nil
}

func (c *Client) ListAdminRoomSearchUsage(ctx context.Context) ([]vibe.RoomSearchUsagePoint, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ListAdminRoomSearchUsage")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.ListAdminRoomSearchUsageStatement.QueryContext(cctx)
	if err != nil {
		return nil, fmt.Errorf("error querying room search usage: %w", err)
	}

	defer rows.Close()

	points := make([]vibe.RoomSearchUsagePoint, 0)
	for rows.Next() {
		var row roomSearchUsageRow
		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning room search usage: %w", err)
		}

		point, err := row.toRoomSearchUsagePoint()
		if err != nil {
			return nil, fmt.Errorf("error mapping room search usage: %w", err)
		}

		points = append(points, *point)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating room search usage: %w", err)
	}

	return points, nil
}

type roomSearchUsageRow struct {
	RoomType  sql.NullString
	Window    sql.NullString
	Timestamp sql.NullTime
	RoomID    sql.NullString
	Provider  sql.NullString
	Total     sql.NullInt64
	Cached    sql.NullInt64
}

func (r *roomSearchUsageRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(&r.Window, &r.Timestamp, &r.RoomID, &r.Provider, &r.Total, &r.Cached, &r.RoomType)
	if err != nil {
		return fmt.Errorf("error scanning room search usage row: %w", err)
	}

	return nil
}

func (r *roomSearchUsageRow) toRoomSearchUsagePoint() (*vibe.RoomSearchUsagePoint, error) {
	return &vibe.RoomSearchUsagePoint{
		RoomType:  vibe.RoomType(r.RoomType.String),
		RoomID:    r.RoomID.String,
		Window:    r.Window.String,
		Timestamp: r.Timestamp.Time,
		Provider:  r.Provider.String,
		Total:     int(r.Total.Int64),
		Cached:    int(r.Cached.Int64),
		Live:      int(r.Total.Int64 - r.Cached.Int64),
	}, nil
}

func (c *Client) prepareListAdminSearchUsageStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH usage_q AS (
			SELECT
				'hour'::text AS aggregation_window,
				DATE_TRUNC('hour', created_at, 'UTC') AS bucket,
				provider,
				SUM(search_count) AS total,
				COUNT(DISTINCT query_hash) AS unique_count,
				COALESCE(
					SUM(search_count) FILTER (WHERE cached),
					0
				) AS cached_count,
				COALESCE(
					SUM(search_count) FILTER (WHERE NOT cached),
					0
				) AS live_count
			FROM search_usage
			WHERE created_at >=
				DATE_TRUNC('hour', NOW(), 'UTC') - INTERVAL '23 hours'
			GROUP BY DATE_TRUNC('hour', created_at, 'UTC'), provider

			UNION ALL

			SELECT
				'day'::text AS aggregation_window,
				DATE_TRUNC('day', created_at, 'UTC') AS bucket,
				provider,
				SUM(search_count) AS total,
				COUNT(DISTINCT query_hash) AS unique_count,
				COALESCE(
					SUM(search_count) FILTER (WHERE cached),
					0
				) AS cached_count,
				COALESCE(
					SUM(search_count) FILTER (WHERE NOT cached),
					0
				) AS live_count
			FROM search_usage
			WHERE created_at >=
				DATE_TRUNC('day', NOW(), 'UTC') - INTERVAL '29 days'
			GROUP BY DATE_TRUNC('day', created_at, 'UTC'), provider
		)
		SELECT
			aggregation_window,
			bucket,
			provider,
			total,
			unique_count,
			cached_count,
			live_count
		FROM usage_q
		ORDER BY aggregation_window, bucket, provider
	`)
	if err != nil {
		return fmt.Errorf("error preparing ListAdminSearchUsageStatement: %w", err)
	}

	c.ListAdminSearchUsageStatement = stmt
	return nil
}

func (c *Client) ListAdminSearchUsage(
	ctx context.Context,
) ([]vibe.AdminSearchUsagePoint, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ListAdminSearchUsage")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.ListAdminSearchUsageStatement.QueryContext(cctx)
	if err != nil {
		return nil, fmt.Errorf(
			"error listing admin search usage in ListAdminSearchUsage: %w",
			err,
		)
	}
	defer rows.Close()

	points := make([]vibe.AdminSearchUsagePoint, 0)
	for rows.Next() {
		var row adminSearchUsageRow
		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"error scanning admin search usage in ListAdminSearchUsage: %w",
				err,
			)
		}

		point, err := row.toAdminSearchUsagePoint()
		if err != nil {
			return nil, fmt.Errorf("error converting search usage in ListAdminSearchUsage: %w", err)
		}

		points = append(points, *point)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf(
			"error iterating admin search usage in ListAdminSearchUsage: %w",
			err,
		)
	}

	return points, nil
}

type adminSearchUsageRow struct {
	Window    sql.NullString
	Timestamp sql.NullTime
	Provider  sql.NullString
	Total     sql.NullInt64
	Unique    sql.NullInt64
	Cached    sql.NullInt64
	Live      sql.NullInt64
}

func (r *adminSearchUsageRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(
		&r.Window,
		&r.Timestamp,
		&r.Provider,
		&r.Total,
		&r.Unique,
		&r.Cached,
		&r.Live,
	)
	if err != nil {
		return fmt.Errorf("error scanning admin search usage row: %w", err)
	}

	return nil
}

func (r *adminSearchUsageRow) toAdminSearchUsagePoint() (*vibe.AdminSearchUsagePoint, error) {
	return &vibe.AdminSearchUsagePoint{
		Window:    r.Window.String,
		Timestamp: r.Timestamp.Time,
		Provider:  r.Provider.String,
		Total:     int(r.Total.Int64),
		Unique:    int(r.Unique.Int64),
		Cached:    int(r.Cached.Int64),
		Live:      int(r.Live.Int64),
	}, nil
}
