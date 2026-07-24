package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

// prepareProcessNextAbandonedHostStmt prepares the ProcessNextAbandonedHostStatement.
func (c *Client) prepareProcessNextAbandonedHostStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH abandoned_room_q AS (
			SELECT a.id
			FROM rooms a
			LEFT JOIN room_users b ON a.host_id = b.id
			WHERE a.mode = 'host'
			AND a.host_id IS NOT NULL
			AND a.host_id != ''
			AND (b.last_seen_at IS NULL OR b.last_seen_at < NOW() - INTERVAL '15 seconds')
			LIMIT 1
			FOR UPDATE OF a SKIP LOCKED
		),
		elected_host_q AS (
			SELECT a.room_id, a.id
			FROM room_users a
			JOIN abandoned_room_q b ON b.id = a.room_id
			WHERE a.last_seen_at >= NOW() - INTERVAL '15 seconds'
			AND a.is_active_listener
			AND NOT a.is_cast_receiver
			ORDER BY a.joined_at ASC
			LIMIT 1
		),
		updated_room_q AS (
			UPDATE rooms a
			SET host_id = COALESCE((SELECT id FROM elected_host_q), '')
			FROM abandoned_room_q b
			WHERE a.id = b.id
			RETURNING a.id, a.host_id
		)
		SELECT id, host_id FROM updated_room_q
	`)
	if err != nil {
		return fmt.Errorf("error preparing ProcessNextAbandonedHostStatement: %w", err)
	}

	c.ProcessNextAbandonedHostStatement = stmt

	return nil
}

func (c *Client) processNextAbandonedHost(ctx context.Context) (*vibe.RoomHostInfo, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "processNextAbandonedHost")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.ProcessNextAbandonedHostStatement.QueryRowContext(cctx)

	var row abandonedHostRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf("error no abandoned host found"),
				},
			}
		}
		return nil, fmt.Errorf("error scanning abandoned host: %w", err)
	}

	return row.toRoomHostInfo(), nil
}

type abandonedHostRow struct {
	RoomID sql.NullString
	HostID sql.NullString
}

func (r *abandonedHostRow) scan(row *sql.Row) error {
	err := row.Scan(&r.RoomID, &r.HostID)
	if err != nil {
		return fmt.Errorf("error scanning abandoned host row: %w", err)
	}

	return nil
}

func (r *abandonedHostRow) toRoomHostInfo() *vibe.RoomHostInfo {
	return &vibe.RoomHostInfo{
		RoomID:    r.RoomID.String,
		NewHostID: r.HostID.String,
	}
}

// ProcessNextAbandonedHost finds a room with an inactive host, elects a new one, and returns info.
func (c *Client) ProcessNextAbandonedHost(ctx context.Context) (*vibe.RoomHostInfo, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ProcessNextAbandonedHost")
	defer span.End()

	hostInfo, err := c.processNextAbandonedHost(ctx)
	if err != nil {
		return nil, fmt.Errorf("error processing abandoned host: %w", err)
	}

	return hostInfo, nil
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
			COALESCE(c.is_admin, FALSE) as is_requester_admin,
			b.enabled_sources,
			b.only_admin_add_songs,
			d.room_id IS NOT NULL AND (
				SELECT COUNT(*)
				FROM songs e
				WHERE e.room_id = a.id
			) <= 2 AS is_generating
		FROM rooms a
		JOIN room_settings b
		ON b.room_id = a.id
		LEFT JOIN room_users c
		ON c.room_id = a.id
		AND c.id = $2
		LEFT JOIN room_generations d
		ON d.room_id = a.id
		WHERE a.id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRoomStatement: %w", err)
	}

	c.GetRoomStatement = stmt

	return nil
}

// GetRoom fetches a room by ID.
// If userID is provided, it also populates the IsAdmin field for that user.
func (c *Client) GetRoom(ctx context.Context, id string, userID string) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetRoom")
	defer span.End()

	room, err := c.getRoom(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting room: %w", err)
	}

	room, err = c.fillRoomDetails(ctx, *room, userID)
	if err != nil {
		return nil, fmt.Errorf("error filling room details: %w", err)
	}

	return room, nil
}

func (c *Client) getRoom(ctx context.Context, id string, userID string) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getRoom")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetRoomStatement.QueryRowContext(cctx, id, userID)

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

	return room, nil
}

func (c *Client) fillRoomDetails(ctx context.Context, room vibe.Room, userID string) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "fillRoomDetails")
	defer span.End()

	filledRoom, err := c.fillActiveSources(ctx, room)
	if err != nil {
		return nil, fmt.Errorf("error filling active sources: %w", err)
	}

	filledRoom.UserID = userID

	counts, err := c.GetActiveListenerCounts(ctx, filledRoom.ID, 15*time.Second)
	if err == nil {
		filledRoom.UserCount = counts.ActiveListeners
		if counts.ActiveListeners == 0 && counts.ActiveCastReceivers > 0 {
			filledRoom.UserCount = 1
		}
	}

	return filledRoom, nil
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
			COALESCE(c.is_admin, FALSE) as is_requester_admin,
			b.enabled_sources,
			b.only_admin_add_songs,
			d.room_id IS NOT NULL AND (
				SELECT COUNT(*)
				FROM songs e
				WHERE e.room_id = a.id
			) <= 2 AS is_generating
		FROM rooms a
		JOIN room_settings b
		ON b.room_id = a.id
		LEFT JOIN room_users c
		ON c.room_id = a.id
		AND c.id = $2
		LEFT JOIN room_generations d
		ON d.room_id = a.id
		WHERE a.name = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRoomByNameStatement: %w", err)
	}

	c.GetRoomByNameStatement = stmt

	return nil
}

// GetRoomByName fetches a room by name.
func (c *Client) GetRoomByName(ctx context.Context, name string, userID string) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetRoomByName")
	defer span.End()

	room, err := c.getRoomByName(ctx, name, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting room by name: %w", err)
	}

	room, err = c.fillRoomDetails(ctx, *room, userID)
	if err != nil {
		return nil, fmt.Errorf("error filling room details: %w", err)
	}

	return room, nil
}

func (c *Client) getRoomByName(ctx context.Context, name string, userID string) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getRoomByName")
	defer span.End()

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

	return room, nil
}

// prepareSuggestRoomNameStmt prepares the SuggestRoomNameStatement.
func (c *Client) prepareSuggestRoomNameStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT candidate_name
		FROM UNNEST($1::text[]) WITH ORDINALITY AS CANDIDATES(candidate_name, candidate_order)
		WHERE NOT EXISTS (
			SELECT 1
			FROM rooms
			WHERE rooms.id = candidate_name
		)
		ORDER BY candidate_order
		LIMIT 1
	`)
	if err != nil {
		return fmt.Errorf("error preparing SuggestRoomNameStatement: %w", err)
	}

	c.SuggestRoomNameStatement = stmt

	return nil
}

// SuggestRoomName returns the first candidate whose room ID is not already in use.
func (c *Client) SuggestRoomName(
	ctx context.Context,
	candidates []string,
) (*vibe.RoomNameSuggestion, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SuggestRoomName")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.SuggestRoomNameStatement.QueryRowContext(cctx, candidates)

	var suggestion vibe.RoomNameSuggestion
	err := row.Scan(&suggestion.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf("error no available room name"),
				},
			}
		}

		return nil, fmt.Errorf("error scanning room name suggestion: %w", err)
	}

	return &suggestion, nil
}

// prepareRoomExistsStmt prepares the RoomExistsStatement.
func (c *Client) prepareRoomExistsStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT EXISTS (
			SELECT 1
			FROM rooms
			WHERE rooms.id = $1
		)
	`)
	if err != nil {
		return fmt.Errorf("error preparing RoomExistsStatement: %w", err)
	}

	c.RoomExistsStatement = stmt

	return nil
}

// RoomExists reports whether a room ID is already in use.
func (c *Client) RoomExists(
	ctx context.Context,
	roomID string,
) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "RoomExists")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.RoomExistsStatement.QueryRowContext(cctx, roomID)

	var exists bool
	err := row.Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error scanning room existence: %w", err)
	}

	return exists, nil
}

type roomRow struct {
	ID                sql.NullString
	Name              sql.NullString
	Mode              sql.NullString
	HostID            sql.NullString
	AdminPasswordHash sql.NullString
	CreatedAt         sql.NullTime
	SkipAllowed       sql.NullBool
	DemocraticSkip    sql.NullBool
	SkipVoteThreshold sql.NullFloat64
	MaxContinuousAdds sql.NullInt64
	RemoveOnPlay      sql.NullBool
	LoopQueue         sql.NullBool
	AllowDuplicates   sql.NullBool
	IsRequesterAdmin  sql.NullBool
	EnabledSources    sql.NullString
	OnlyAdminAddSongs sql.NullBool
	IsGenerating      sql.NullBool
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
		&r.EnabledSources,
		&r.OnlyAdminAddSongs,
		&r.IsGenerating,
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
		IsAdmin:           r.IsRequesterAdmin.Bool,
		Settings:          *settings,
		CreatedAt:         r.CreatedAt.Time,
		IsGenerating:      r.IsGenerating.Bool,
	}, nil
}

func (r *roomRow) toRoomSettings() (*vibe.RoomSettings, error) {
	sources := []string{}
	if r.EnabledSources.Valid && r.EnabledSources.String != "" {
		sources = strings.Split(r.EnabledSources.String, ",")
	}
	if r.EnabledSources.Valid && r.EnabledSources.String == "" {
		sources = []string{}
	}
	if !r.EnabledSources.Valid {
		sources = []string{"youtube", "spotify", "soundcloud"}
	}

	return &vibe.RoomSettings{
		SkipAllowed:       r.SkipAllowed.Bool,
		DemocraticSkip:    r.DemocraticSkip.Bool,
		SkipVoteThreshold: r.SkipVoteThreshold.Float64,
		MaxContinuousAdds: int(r.MaxContinuousAdds.Int64),
		RemoveOnPlay:      r.RemoveOnPlay.Bool,
		LoopQueue:         r.LoopQueue.Bool,
		AllowDuplicates:   r.AllowDuplicates.Bool,
		EnabledSources:    sources,
		OnlyAdminAddSongs: r.OnlyAdminAddSongs.Bool,
	}, nil
}

func (c *Client) prepareGetActiveSourcesStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT DISTINCT source_type FROM songs WHERE room_id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetActiveSourcesStatement: %w", err)
	}
	c.GetActiveSourcesStatement = stmt
	return nil
}

func (c *Client) fillActiveSources(ctx context.Context, room vibe.Room) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "fillActiveSources")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.GetActiveSourcesStatement.QueryContext(cctx, room.ID)
	if err != nil {
		return nil, fmt.Errorf("error in db: get active sources: %w", err)
	}
	defer rows.Close()

	sources := []string{}
	for rows.Next() {
		var row activeSourceRow
		err := row.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("error in db: scan active source: %w", err)
		}
		sources = append(sources, row.toSource())
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error in db: iterate active sources: %w", err)
	}

	room.ActiveSources = sources
	return &room, nil
}

type activeSourceRow struct {
	Source sql.NullString
}

func (a *activeSourceRow) scan(rows *sql.Rows) error {
	return rows.Scan(&a.Source)
}

func (a *activeSourceRow) toSource() string {
	return a.Source.String
}

// prepareCreateRoomStmt prepares the CreateRoomStatement.
func (c *Client) prepareCreateRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH created_room_q AS (
			INSERT INTO rooms (id, name, mode, host_id, admin_password_hash, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		),
		created_settings_q AS (
			INSERT INTO room_settings (
				room_id,
				skip_allowed,
				democratic_skip,
				skip_vote_threshold,
				max_continuous_adds,
				remove_on_play,
				loop_queue,
				allow_duplicates,
				enabled_sources,
				only_admin_add_songs
			)
			SELECT id, $7, $8, $9, $10, $11, $12, $13, $14, $15 FROM created_room_q
		)
		SELECT id FROM created_room_q
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateRoomStatement: %w", err)
	}

	c.CreateRoomStatement = stmt

	return nil
}

// CreateRoom creates a new room.
func (c *Client) CreateRoom(ctx context.Context, room *vibe.Room) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateRoom")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.CreateRoomStatement.QueryRowContext(cctx,
		room.ID,
		room.Name,
		room.Mode,
		room.HostID,
		room.AdminPasswordHash,
		room.CreatedAt,
		room.Settings.SkipAllowed,
		room.Settings.DemocraticSkip,
		room.Settings.SkipVoteThreshold,
		room.Settings.MaxContinuousAdds,
		room.Settings.RemoveOnPlay,
		room.Settings.LoopQueue,
		room.Settings.AllowDuplicates,
		strings.Join(room.Settings.EnabledSources, ","),
		room.Settings.OnlyAdminAddSongs,
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
		WITH updated_room_q AS (
			UPDATE rooms
			SET name = $1,
			mode = $10,
			host_id = $11,
			admin_password_hash = $12
			WHERE id = $2
			RETURNING id
		)
		UPDATE room_settings
		SET skip_allowed = $3,
		democratic_skip = $4,
		skip_vote_threshold = $5,
		max_continuous_adds = $6,
		remove_on_play = $7,
		loop_queue = $8,
		allow_duplicates = $9,
		enabled_sources = $13,
		only_admin_add_songs = $14
		FROM updated_room_q a
		WHERE room_settings.room_id = a.id
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdateRoomStatement: %w", err)
	}

	c.UpdateRoomStatement = stmt

	return nil
}

// UpdateRoom updates an existing room.
func (c *Client) UpdateRoom(ctx context.Context, room *vibe.Room) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdateRoom")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.UpdateRoomStatement.ExecContext(cctx,
		room.Name,
		room.ID,
		room.Settings.SkipAllowed,
		room.Settings.DemocraticSkip,
		room.Settings.SkipVoteThreshold,
		room.Settings.MaxContinuousAdds,
		room.Settings.RemoveOnPlay,
		room.Settings.LoopQueue,
		room.Settings.AllowDuplicates,
		room.Mode,
		room.HostID,
		room.AdminPasswordHash,
		strings.Join(room.Settings.EnabledSources, ","),
		room.Settings.OnlyAdminAddSongs,
	)
	if err != nil {
		return nil, fmt.Errorf("error updating room: %w", err)
	}

	updatedRoom, err := c.GetRoom(ctx, room.ID, room.UserID)
	if err != nil {
		return nil, fmt.Errorf("error fetching updated room: %w", err)
	}

	return updatedRoom, nil
}
