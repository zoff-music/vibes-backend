package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
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

	r := c.ProcessNextAbandonedHostStatement.QueryRowContext(cctx)

	var roomID sql.NullString
	err := r.Scan(&roomID)
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

	r = c.ElectNewHostStatement.QueryRowContext(cctx, roomID)

	var newHostID sql.NullString
	err = r.Scan(&newHostID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("error electing new host: %w", err)
	}
	// if ErrNoRows, newHostID is "", which is fine (no host)

	// Update the room
	err = c.SetRoomHost(ctx, roomID.String, newHostID.String)
	if err != nil {
		return nil, fmt.Errorf("error setting room host: %w", err)
	}

	return &vibe.RoomHostInfo{
		RoomID:    roomID.String,
		NewHostID: newHostID.String,
	}, nil
}

func (c *Client) prepareElectNewHostStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id FROM room_users 
		WHERE room_id = ?1 
		AND last_seen_at >= DATETIME('now', '-15 seconds')
		ORDER BY joined_at ASC
		LIMIT 1
	`)
	if err != nil {
		return fmt.Errorf("error preparing ElectNewHostStatement: %w", err)
	}
	c.ElectNewHostStatement = stmt
	return nil
}

// prepareGetRoomStmt prepares the GetRoomStatement.
func (c *Client) prepareGetRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			a.id,
			a.name,
			a.mode,
			a.host_id,
			a.admin_password_hash,
			a.created_at,
			b.skip_allowed,
			b.democratic_skip,
			b.skip_vote_threshold,
			b.max_continuous_adds,
			b.remove_on_play,
			b.loop_queue,
			b.allow_duplicates,
			COALESCE(c.is_admin, 0) as is_requester_admin
		FROM rooms a
		JOIN room_settings b
		ON b.room_id = a.id
		LEFT JOIN room_users c 
		ON c.room_id = a.id 
		AND c.id = ?2
		WHERE a.id = ?1
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
			a.id,
			a.name,
			a.mode,
			a.host_id,
			a.admin_password_hash,
			a.created_at,
			b.skip_allowed,
			b.democratic_skip,
			b.skip_vote_threshold,
			b.max_continuous_adds,
			b.remove_on_play,
			b.loop_queue,
			b.allow_duplicates,
			COALESCE(c.is_admin, 0) as is_requester_admin
		FROM rooms a
		JOIN room_settings b
		ON b.room_id = a.id
		LEFT JOIN room_users c 
		ON c.room_id = a.id 
		AND c.id = ?2
		WHERE a.name = ?1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRoomByNameStatement: %w", err)
	}

	c.GetRoomByNameStatement = stmt

	return nil
}

// GetRoomByName fetches a room by name.
func (c *Client) GetRoomByName(ctx context.Context, name string, userID string) (*vibe.Room, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetRoomByName")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetRoomByNameStatement.QueryRowContext(cctx, name, userID)

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

	room, err = c.fillActiveSources(ctx, *room)
	if err != nil {
		return nil, fmt.Errorf("error filling active sources: %w", err)
	}

	room.UserID = userID

	active, err := c.GetActiveParticipants(ctx, room.ID, 15*time.Second)
	if err == nil {
		room.UserCount = len(active)
	}

	return room, nil
}

func (c *Client) prepareGetActiveSourcesStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT DISTINCT source_type FROM songs WHERE room_id = ?1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetActiveSourcesStatement: %w", err)
	}
	c.GetActiveSourcesStatement = stmt
	return nil
}

func (c *Client) fillActiveSources(ctx context.Context, room vibe.Room) (*vibe.Room, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "fillActiveSources")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.GetActiveSourcesStatement.QueryContext(cctx, room.ID)
	if err != nil {
		return nil, fmt.Errorf("error in db: get active sources: %w", err)
	}
	defer rows.Close()

	sources := []string{}
	for rows.Next() {
		var row sql.NullString
		err := rows.Scan(&row)
		if err != nil {
			return nil, fmt.Errorf("error in db: scan active source: %w", err)
		}
		sources = append(sources, row.String)
	}

	room.ActiveSources = sources
	return &room, nil
}

// GetRoom fetches a room by ID.
// If userID is provided, it also populates the IsAdmin field for that user.
func (c *Client) GetRoom(ctx context.Context, id string, userID string) (*vibe.Room, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetRoom")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetRoomStatement.QueryRowContext(cctx, id, userID)
	log.Printf("database.GetRoom: roomID=%s, userID=%s", id, userID)

	var row roomRow
	err := row.scanRow(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.Room{}, nil
		}

		return nil, fmt.Errorf("error fetching room: %w", err)
	}

	room, err := row.toRoom()
	if err != nil {
		return nil, fmt.Errorf("error converting room row: %w", err)
	}

	room, err = c.fillActiveSources(ctx, *room)
	if err != nil {
		return nil, fmt.Errorf("error filling active sources: %w", err)
	}

	room.UserID = userID

	active, err := c.GetActiveParticipants(ctx, room.ID, 15*time.Second)
	if err == nil {
		room.UserCount = len(active)
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
	IsRequesterAdmin  sql.NullInt64
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
		&r.IsRequesterAdmin,
	)
}

func (r *roomRow) toRoom() (*vibe.Room, error) {
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
		IsAdmin:           r.IsRequesterAdmin.Int64 == 1,
		Settings:          *settings,
		CreatedAt:         r.CreatedAt.Time,
	}, nil
}

func (r *roomRow) toRoomSettings() (*vibe.RoomSettings, error) {

	return &vibe.RoomSettings{
		SkipAllowed:       int(r.SkipAllowed.Int64) == 1,
		DemocraticSkip:    int(r.DemocraticSkip.Int64) == 1,
		SkipVoteThreshold: r.SkipVoteThreshold.Float64,
		MaxContinuousAdds: int(r.MaxContinuousAdds.Int64),
		RemoveOnPlay:      int(r.RemoveOnPlay.Int64) == 1,
		LoopQueue:         int(r.LoopQueue.Int64) == 1,
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
			mode = ?10,
			host_id = ?11,
			skip_allowed = ?3,
			democratic_skip = ?4,
			skip_vote_threshold = ?5,
			max_continuous_adds = ?6,
			remove_on_play = ?7,
			loop_queue = ?8,
			allow_duplicates = ?9
		WHERE id = ?2
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
