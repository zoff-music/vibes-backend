package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zoff-music/vibes/internalerror"
	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"
)

// prepareProcessNextAbandonedHostStmt prepares the ProcessNextAbandonedHostStatement.
func (c *Client) prepareProcessNextAbandonedHostStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT r.id
		FROM rooms r
		LEFT JOIN room_users ru ON r.host_id = ru.id
		WHERE r.mode = 'host'
		  AND r.host_id IS NOT NULL AND r.host_id != ''
		  AND (ru.last_seen_at IS NULL OR ru.last_seen_at < datetime('now', '-15 seconds'))
		LIMIT 1
	`)
	if err != nil {
		return fmt.Errorf("error preparing ProcessNextAbandonedHostStatement: %w", err)
	}

	c.ProcessNextAbandonedHostStatement = stmt

	return nil
}

// ProcessNextAbandonedHost finds a room with an inactive host, elects a new one, and returns info.
func (c *Client) ProcessNextAbandonedHost(ctx context.Context) (*vibe.RoomHostInfo, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ProcessNextAbandonedHost")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var roomID string
	err := c.ProcessNextAbandonedHostStatement.QueryRowContext(cctx).Scan(&roomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf("no abandoned host found"),
				},
			}
		}
		return nil, fmt.Errorf("error finding abandoned host: %w", err)
	}

	// Helper to find next candidate
	// We do this via standard query as it's not pre-prepared in this specific unique flow usually,
	// but better to prepare it or use existing one. GetActiveParticipants is generic.
	// We'll use a direct query here for speed/conciseness or use a new prepared stmt?
	// Let's use direct query for now or reuse GetUsersInRoom logic if available.
	// Actually we need one that orders by join time filter by activity.
	// "GetActiveParticipants" typically does this.

	// Let's define the query for election inside this method or helper.
	// Since we can't easily add a new generic method in this Replace block without breaking flow,
	// I'll execute the query directly.

	var newHostID string
	err = c.DB.QueryRowContext(cctx, `
		SELECT id FROM room_users 
		WHERE room_id = ? 
		  AND last_seen_at >= datetime('now', '-15 seconds')
		ORDER BY joined_at ASC -- Oldest active user promotes
		LIMIT 1
	`, roomID).Scan(&newHostID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("error electing new host: %w", err)
	}
	// if ErrNoRows, newHostID is "", which is fine (no host)

	// Update the room
	err = c.SetRoomHost(ctx, roomID, newHostID)
	if err != nil {
		return nil, fmt.Errorf("error setting room host: %w", err)
	}

	return &vibe.RoomHostInfo{
		RoomID:    roomID,
		NewHostID: newHostID,
	}, nil
}

// prepareGetRoomStmt prepares the GetRoomStatement.
func (c *Client) prepareGetRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
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
		WHERE rooms.id = ?1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRoomStatement: %w", err)
	}

	c.GetRoomStatement = stmt

	return nil
}

// prepareGetRoomByNameStmt prepares the GetRoomByNameStatement.
func (c *Client) prepareGetRoomByNameStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
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
		WHERE rooms.name = ?1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRoomByNameStatement: %w", err)
	}

	c.GetRoomByNameStatement = stmt

	return nil
}

// GetRoomByName fetches a room by name.
func (c *Client) GetRoomByName(ctx context.Context, name string) (*vibe.Room, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetRoomByName")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetRoomByNameStatement.QueryRowContext(cctx, name)

	var scanned roomRow

	err := scanned.scanRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.Room{}, nil
		}

		return nil, fmt.Errorf("error fetching room by name: %w", err)
	}

	room, err := scanned.toRoom()
	if err != nil {
		return nil, fmt.Errorf("error converting room row: %w", err)
	}

	if err := c.fillActiveSources(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

func (c *Client) fillActiveSources(ctx context.Context, room *vibe.Room) error {
	rows, err := c.DB.QueryContext(ctx, `
		SELECT DISTINCT source_type FROM songs WHERE room_id = ?
	`, room.ID)
	if err != nil {
		return fmt.Errorf("error in db: get active sources: %w", err)
	}
	defer rows.Close()
	sources := []string{}
	for rows.Next() {
		var source string
		if err := rows.Scan(&source); err != nil {
			return fmt.Errorf("error in db: scan active source: %w", err)
		}
		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error in db: iterate active sources: %w", err)
	}

	room.ActiveSources = sources
	return nil
}

// GetRoom fetches a room by ID.
func (c *Client) GetRoom(ctx context.Context, id string) (*vibe.Room, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetRoom")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetRoomStatement.QueryRowContext(cctx, id)

	var scanned roomRow

	err := scanned.scanRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.Room{}, nil
		}

		return nil, fmt.Errorf("error fetching room: %w", err)
	}

	room, err := scanned.toRoom()
	if err != nil {
		return nil, fmt.Errorf("error converting room row: %w", err)
	}

	if err := c.fillActiveSources(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

type roomRow struct {
	ID                sql.NullString
	Name              sql.NullString
	Mode              sql.NullString
	HostID            sql.NullString
	AdminPasswordHash sql.NullString
	CreatedAt         sql.NullTime
	SkipAllowed       sql.NullInt64
	DemocraticSkip    sql.NullInt64
	SkipVoteThreshold sql.NullFloat64
	MaxContinuousAdds sql.NullInt64
	RemoveOnPlay      sql.NullInt64
	LoopQueue         sql.NullInt64
	AllowDuplicates   sql.NullInt64
}

func (r *roomRow) scanRow(row *sql.Row) error {
	return row.Scan(
		&r.ID,
		&r.Name,
		&r.Mode,
		&r.HostID,
		&r.AdminPasswordHash,
		&r.CreatedAt,
		&r.SkipAllowed,
		&r.DemocraticSkip,
		&r.SkipVoteThreshold,
		&r.MaxContinuousAdds,
		&r.RemoveOnPlay,
		&r.LoopQueue,
		&r.AllowDuplicates,
	)
}

func (r *roomRow) scanRows(rows *sql.Rows) error {
	return rows.Scan(
		&r.ID,
		&r.Name,
		&r.Mode,
		&r.HostID,
		&r.AdminPasswordHash,
		&r.CreatedAt,
		&r.SkipAllowed,
		&r.DemocraticSkip,
		&r.SkipVoteThreshold,
		&r.MaxContinuousAdds,
		&r.RemoveOnPlay,
		&r.LoopQueue,
		&r.AllowDuplicates,
	)
}

func (r *roomRow) toRoom() (*vibe.Room, error) {
	if !r.CreatedAt.Valid {
		return nil, fmt.Errorf("error missing room created_at")
	}

	settings, err := r.toRoomSettings()
	if err != nil {
		return nil, fmt.Errorf("error converting room settings: %w", err)
	}

	return &vibe.Room{
		ID:                r.ID.String,
		Name:              r.Name.String,
		Mode:              r.Mode.String,
		HostID:            r.HostID.String,
		AdminPasswordHash: r.AdminPasswordHash.String,
		HasPassword:       r.AdminPasswordHash.Valid && r.AdminPasswordHash.String != "",
		Settings:          *settings,
		CreatedAt:         r.CreatedAt.Time,
	}, nil
}

func (r *roomRow) toRoomSettings() (*vibe.RoomSettings, error) {
	if !r.SkipAllowed.Valid {
		return nil, fmt.Errorf("error missing room settings skip_allowed")
	}

	if !r.DemocraticSkip.Valid {
		return nil, fmt.Errorf("error missing room settings democratic_skip")
	}

	if !r.SkipVoteThreshold.Valid {
		return nil, fmt.Errorf("error missing room settings skip_vote_threshold")
	}

	if !r.MaxContinuousAdds.Valid {
		return nil, fmt.Errorf("error missing room settings max_continuous_adds")
	}

	if !r.RemoveOnPlay.Valid {
		return nil, fmt.Errorf("error missing room settings remove_on_play")
	}

	if !r.LoopQueue.Valid {
		return nil, fmt.Errorf("error missing room settings loop_queue")
	}

	if !r.AllowDuplicates.Valid {
		return nil, fmt.Errorf("error missing room settings allow_duplicates")
	}

	return &vibe.RoomSettings{
		SkipAllowed:       r.SkipAllowed.Int64 == 1,
		DemocraticSkip:    r.DemocraticSkip.Int64 == 1,
		SkipVoteThreshold: r.SkipVoteThreshold.Float64,
		MaxContinuousAdds: int(r.MaxContinuousAdds.Int64),
		RemoveOnPlay:      r.RemoveOnPlay.Int64 == 1,
		LoopQueue:         r.LoopQueue.Int64 == 1,
		AllowDuplicates:   r.AllowDuplicates.Int64 == 1,
	}, nil
}

// prepareCreateRoomStmt prepares the CreateRoomStatement.
func (c *Client) prepareCreateRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO rooms_view (
			id,
			name,
			mode,
			admin_password_hash,
			created_at,
			skip_allowed,
			democratic_skip,
			skip_vote_threshold,
			max_continuous_adds,
			remove_on_play,
			loop_queue,
			allow_duplicates
		) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)
		RETURNING id
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateRoomStatement: %w", err)
	}

	c.CreateRoomStatement = stmt

	return nil
}

// CreateRoom creates a new room.
func (c *Client) CreateRoom(ctx context.Context, room *vibe.Room) (*vibe.Room, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CreateRoom")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Atomic CTE query
	row := c.CreateRoomStatement.QueryRowContext(cctx,
		room.ID,
		room.Name,
		room.Mode,
		room.AdminPasswordHash,
		room.CreatedAt,
		boolToInt(room.Settings.SkipAllowed),
		boolToInt(room.Settings.DemocraticSkip),
		room.Settings.SkipVoteThreshold,
		room.Settings.MaxContinuousAdds,
		boolToInt(room.Settings.RemoveOnPlay),
		boolToInt(room.Settings.LoopQueue),
		boolToInt(room.Settings.AllowDuplicates),
	)

	var scanned createRoomRow
	err := scanned.scan(row)
	if err != nil {
		return nil, fmt.Errorf("error creating room: %w", err)
	}

	return room, nil
}

type createRoomRow struct {
	ID sql.NullString
}

func (r *createRoomRow) scan(row *sql.Row) error {
	return row.Scan(&r.ID)
}

// prepareUpdateRoomStmt prepares the UpdateRoomStatement.
func (c *Client) prepareUpdateRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE rooms_view
		SET
			name = ?1,
			admin_password_hash = ?2,
			mode = ?11,
			host_id = ?12,
			skip_allowed = ?4,
			democratic_skip = ?5,
			skip_vote_threshold = ?6,
			max_continuous_adds = ?7,
			remove_on_play = ?8,
			loop_queue = ?9,
			allow_duplicates = ?10
		WHERE id = ?3
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdateRoomStatement: %w", err)
	}

	c.UpdateRoomStatement = stmt

	return nil
}

// UpdateRoom updates an existing room.
func (c *Client) UpdateRoom(ctx context.Context, room *vibe.Room) (*vibe.Room, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UpdateRoom")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.UpdateRoomStatement.ExecContext(cctx,
		room.Name,
		room.AdminPasswordHash,
		room.ID,
		boolToInt(room.Settings.SkipAllowed),
		boolToInt(room.Settings.DemocraticSkip),
		room.Settings.SkipVoteThreshold,
		room.Settings.MaxContinuousAdds,
		boolToInt(room.Settings.RemoveOnPlay),
		boolToInt(room.Settings.LoopQueue),
		boolToInt(room.Settings.AllowDuplicates),
		room.Mode,
		room.HostID,
	)
	if err != nil {
		return nil, fmt.Errorf("error updating room: %w", err)
	}

	return room, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}
