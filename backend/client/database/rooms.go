package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"
)

// prepareGetRoomStmt prepares the GetRoomStatement.
func (c *Client) prepareGetRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			rooms.id,
			rooms.name,
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

	err := scanned.scan(row)
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

	return room, nil
}

// GetRoom fetches a room by ID.
func (c *Client) GetRoom(ctx context.Context, id string) (*vibe.Room, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetRoom")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetRoomStatement.QueryRowContext(cctx, id)

	var scanned roomRow

	err := scanned.scan(row)
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

	return room, nil
}

type roomRow struct {
	ID                sql.NullString
	Name              sql.NullString
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

func (r *roomRow) scan(row *sql.Row) error {
	return row.Scan(
		&r.ID,
		&r.Name,
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
		AdminPasswordHash: r.AdminPasswordHash.String,
		HasPassword:       r.AdminPasswordHash.Valid && r.AdminPasswordHash.String != "",
		Settings:          settings,
		CreatedAt:         r.CreatedAt.Time,
	}, nil
}

func (r *roomRow) toRoomSettings() (vibe.RoomSettings, error) {
	if !r.SkipAllowed.Valid {
		return vibe.RoomSettings{}, fmt.Errorf("error missing room settings skip_allowed")
	}

	if !r.DemocraticSkip.Valid {
		return vibe.RoomSettings{}, fmt.Errorf("error missing room settings democratic_skip")
	}

	if !r.SkipVoteThreshold.Valid {
		return vibe.RoomSettings{}, fmt.Errorf("error missing room settings skip_vote_threshold")
	}

	if !r.MaxContinuousAdds.Valid {
		return vibe.RoomSettings{}, fmt.Errorf("error missing room settings max_continuous_adds")
	}

	if !r.RemoveOnPlay.Valid {
		return vibe.RoomSettings{}, fmt.Errorf("error missing room settings remove_on_play")
	}

	if !r.LoopQueue.Valid {
		return vibe.RoomSettings{}, fmt.Errorf("error missing room settings loop_queue")
	}

	if !r.AllowDuplicates.Valid {
		return vibe.RoomSettings{}, fmt.Errorf("error missing room settings allow_duplicates")
	}

	return vibe.RoomSettings{
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
			admin_password_hash,
			created_at,
			skip_allowed,
			democratic_skip,
			skip_vote_threshold,
			max_continuous_adds,
			remove_on_play,
			loop_queue,
			allow_duplicates
		) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11)
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
