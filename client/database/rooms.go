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

// prepareProcessNextAbandonedHostStmt prepares the ProcessNextAbandonedHostStatement.
func (c *Client) prepareProcessNextAbandonedHostStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH abandoned_room_q AS (
			SELECT a.id
			FROM rooms a
			LEFT JOIN room_users b ON a.host_id = b.id AND a.id = b.room_id
			WHERE a.mode = 'host'
			AND (
				(a.host_id IS NOT NULL AND a.host_id != '' AND (
					b.last_seen_at IS NULL
					OR b.last_seen_at < NOW() - INTERVAL '15 seconds'
					OR NOT b.is_active_listener
				))
				OR ((a.host_id IS NULL OR a.host_id = '') AND EXISTS (
					SELECT 1 FROM room_users c
					WHERE c.room_id = a.id
					AND c.last_seen_at >= NOW() - INTERVAL '15 seconds'
					AND c.is_active_listener
					AND NOT c.is_cast_receiver
				))
			)
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

	hostInfo, err := row.toRoomHostInfo()
	if err != nil {
		return nil, fmt.Errorf("error mapping hostInfo: %w", err)
	}

	return hostInfo, nil
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

func (r *abandonedHostRow) toRoomHostInfo() (*vibe.RoomHostInfo, error) {
	return &vibe.RoomHostInfo{
		RoomID:    r.RoomID.String,
		NewHostID: r.HostID.String,
	}, nil
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

// prepareGetRoomV2Stmt prepares the GetRoomV2Statement.
func (c *Client) prepareGetRoomV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			a.id,
			a.name,
			a.mode,
			a.room_type,
			a.host_id,
			a.admin_password_hash,
			a.created_at,
			b.skip_allowed,
			b.democratic_skip,
			b.skip_vote_threshold,
			b.max_continuous_adds,
			b.remove_on_play,
			b.allow_duplicates,
			COALESCE(c.is_admin, FALSE) as is_requester_admin,
			b.enabled_sources,
			b.only_admin_add_playlist_items,
			b.is_public,
			b.playlist_import,
			EXISTS (
				SELECT 1
				FROM room_generations d
				WHERE d.room_id = a.id
				AND d.attempt < $3
				AND d.completed_at IS NULL
				AND d.failed_at IS NULL
			) AS is_generating,
			(
				SELECT COUNT(*)
				FROM room_generations e
				WHERE e.room_id = a.id
				AND e.created_at >= NOW() - INTERVAL '24 hours'
			) AS generation_count,
			(
				SELECT f.failure_reason
				FROM room_generations f
				WHERE f.room_id = a.id
				ORDER BY f.created_at DESC
				LIMIT 1
			) AS generation_error
		FROM rooms a
		JOIN room_settings b
		ON b.room_id = a.id
		LEFT JOIN room_users c
		ON c.room_id = a.id
		AND c.id = $2
		WHERE a.id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRoomV2Statement: %w", err)
	}

	c.GetRoomV2Statement = stmt

	return nil
}

// GetRoom fetches a room by ID.
// If userID is provided, it also populates the IsAdmin field for that user.
func (c *Client) GetRoomV2(ctx context.Context, id string, userID string) (*vibe.RoomV2, error) {
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

func (c *Client) getRoom(ctx context.Context, id string, userID string) (*vibe.RoomV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getRoom")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetRoomV2Statement.QueryRowContext(
		cctx,
		id,
		userID,
		c.roomGenerationMaxAttempts,
	)

	var row roomRow
	err := row.scanRow(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RoomV2{}, nil
		}

		return nil, fmt.Errorf("error fetching room: %w", err)
	}

	room, err := row.toRoomV2(c.enabledProviders)
	if err != nil {
		return nil, fmt.Errorf("error converting room row: %w", err)
	}

	return room, nil
}

func (c *Client) fillRoomDetails(ctx context.Context, room vibe.RoomV2, userID string) (*vibe.RoomV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "fillRoomDetails")
	defer span.End()

	filledRoom, err := c.fillActiveSources(ctx, room)
	if err != nil {
		return nil, fmt.Errorf("error filling active sources: %w", err)
	}

	filledRoom.UserID = userID
	filledRoom.RoomGenerationMaxDailyCount = c.roomGenerationMaxDailyCount
	filledRoom.RoomGenerationMaxExistingPlaylistItems = c.roomGenerationMaxExistingPlaylistItems

	counts, err := c.GetActiveListenerCounts(ctx, filledRoom.ID, 15*time.Second)
	if err == nil {
		filledRoom.UserCount = counts.ActiveListeners
		if counts.ActiveListeners == 0 && counts.ActiveCastReceivers > 0 {
			filledRoom.UserCount = 1
		}
	}

	return filledRoom, nil
}

// prepareGetRoomByNameV2Stmt prepares the GetRoomByNameV2Statement.
func (c *Client) prepareGetRoomByNameV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			a.id,
			a.name,
			a.mode,
			a.room_type,
			a.host_id,
			a.admin_password_hash,
			a.created_at,
			b.skip_allowed,
			b.democratic_skip,
			b.skip_vote_threshold,
			b.max_continuous_adds,
			b.remove_on_play,
			b.allow_duplicates,
			COALESCE(c.is_admin, FALSE) as is_requester_admin,
			b.enabled_sources,
			b.only_admin_add_playlist_items,
			b.is_public,
			b.playlist_import,
			EXISTS (
				SELECT 1
				FROM room_generations d
				WHERE d.room_id = a.id
				AND d.attempt < $3
				AND d.completed_at IS NULL
				AND d.failed_at IS NULL
			) AS is_generating,
			(
				SELECT COUNT(*)
				FROM room_generations e
				WHERE e.room_id = a.id
				AND e.created_at >= NOW() - INTERVAL '24 hours'
			) AS generation_count,
			(
				SELECT f.failure_reason
				FROM room_generations f
				WHERE f.room_id = a.id
				ORDER BY f.created_at DESC
				LIMIT 1
			) AS generation_error
		FROM rooms a
		JOIN room_settings b
		ON b.room_id = a.id
		LEFT JOIN room_users c
		ON c.room_id = a.id
		AND c.id = $2
		WHERE a.name = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRoomByNameV2Statement: %w", err)
	}

	c.GetRoomByNameV2Statement = stmt

	return nil
}

// GetRoomByName fetches a room by name.
func (c *Client) GetRoomByNameV2(ctx context.Context, name string, userID string) (*vibe.RoomV2, error) {
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

func (c *Client) getRoomByName(ctx context.Context, name string, userID string) (*vibe.RoomV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getRoomByName")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetRoomByNameV2Statement.QueryRowContext(
		cctx,
		name,
		userID,
		c.roomGenerationMaxAttempts,
	)

	var scanned roomRow
	err := scanned.scanRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RoomV2{}, nil
		}

		return nil, fmt.Errorf("error fetching room by name: %w", err)
	}

	room, err := scanned.toRoomV2(c.enabledProviders)
	if err != nil {
		return nil, fmt.Errorf("error converting room row: %w", err)
	}

	return room, nil
}

func (c *Client) prepareGetPublicRoomsStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH participant_counts_q AS (
			SELECT
				room_id,
				COUNT(*) FILTER (
					WHERE is_active_listener
					AND NOT is_cast_receiver
				) AS active_listeners,
				COUNT(*) FILTER (
					WHERE is_cast_receiver
				) AS active_cast_receivers
			FROM room_users
			WHERE last_seen_at >= NOW() - INTERVAL '15 seconds'
			GROUP BY room_id
		),
		active_rooms_q AS (
			SELECT
				a.id,
				a.name,
				CASE
					WHEN c.active_listeners = 0
					AND c.active_cast_receivers > 0
					THEN 1
					ELSE c.active_listeners
				END AS listener_count
			FROM rooms a
			JOIN room_settings b
			ON b.room_id = a.id
			JOIN participant_counts_q c
			ON c.room_id = a.id
			WHERE b.is_public
			AND a.room_type = 'MUSIC'
			AND a.admin_password_hash IS NOT NULL
			AND a.admin_password_hash != ''
			AND (
				c.active_listeners > 0
				OR c.active_cast_receivers > 0
			)
		)
		SELECT
			a.id,
			a.name,
			a.listener_count,
			COUNT(b.id) AS playlist_item_count
		FROM active_rooms_q a
		LEFT JOIN playlist_items b
		ON b.room_id = a.id
		AND b.source_type = ANY($1::text[])
		GROUP BY a.id, a.name, a.listener_count
		ORDER BY
			a.listener_count DESC,
			playlist_item_count DESC,
			a.name ASC
		LIMIT 3
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetPublicRoomsStatement: %w", err)
	}

	c.GetPublicRoomsStatement = stmt

	return nil
}

// GetPublicRooms fetches public, password-protected rooms with active listeners.
func (c *Client) GetPublicRooms(ctx context.Context) ([]vibe.PublicRoom, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetPublicRooms")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.GetPublicRoomsStatement.QueryContext(
		cctx,
		c.enabledProviders,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying public rooms: %w", err)
	}
	defer rows.Close()

	publicRooms := []vibe.PublicRoom{}
	for rows.Next() {
		var row publicRoomRow
		err = row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning public room: %w", err)
		}

		room, err := row.toPublicRoom()
		if err != nil {
			return nil, fmt.Errorf("error converting public room in GetPublicRooms: %w", err)
		}

		publicRooms = append(publicRooms, *room)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating public rooms: %w", err)
	}

	return publicRooms, nil
}

type publicRoomRow struct {
	ID                sql.NullString
	Name              sql.NullString
	RoomType          sql.NullString
	ListenerCount     sql.NullInt64
	PlaylistItemCount sql.NullInt64
}

func (r *publicRoomRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(
		&r.ID,
		&r.Name,
		&r.ListenerCount,
		&r.PlaylistItemCount,
	)
	if err != nil {
		return fmt.Errorf("error scanning public room row: %w", err)
	}

	return nil
}

func (r *publicRoomRow) toPublicRoom() (*vibe.PublicRoom, error) {
	return &vibe.PublicRoom{
		ID:            r.ID.String,
		Name:          r.Name.String,
		ListenerCount: int(r.ListenerCount.Int64),
		SongCount:     int(r.PlaylistItemCount.Int64),
	}, nil
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

	r := c.RoomExistsStatement.QueryRowContext(cctx, roomID)

	var row roomExistsRow
	err := row.scan(r)
	if err != nil {
		return false, fmt.Errorf("error scanning room existence: %w", err)
	}

	return row.Exists, nil
}

type roomExistsRow struct {
	Exists bool
}

func (r *roomExistsRow) scan(row *sql.Row) error {
	err := row.Scan(&r.Exists)
	if err != nil {
		return fmt.Errorf("error scanning room existence row: %w", err)
	}

	return nil
}

type roomRow struct {
	ID                        sql.NullString
	Name                      sql.NullString
	Mode                      sql.NullString
	RoomType                  sql.NullString
	HostID                    sql.NullString
	AdminPasswordHash         sql.NullString
	CreatedAt                 sql.NullTime
	SkipAllowed               sql.NullBool
	DemocraticSkip            sql.NullBool
	SkipVoteThreshold         sql.NullFloat64
	MaxContinuousAdds         sql.NullInt64
	RemoveOnPlay              sql.NullBool
	AllowDuplicates           sql.NullBool
	IsRequesterAdmin          sql.NullBool
	EnabledSources            sql.NullString
	OnlyAdminAddPlaylistItems sql.NullBool
	Public                    sql.NullBool
	PlaylistImport            sql.NullBool
	IsGenerating              sql.NullBool
	GenerationCount           sql.NullInt64
	GenerationError           sql.NullString
}

func (r *roomRow) scanRow(row *sql.Row) error {
	err := row.Scan(
		&r.ID,
		&r.Name,
		&r.Mode,
		&r.RoomType,
		&r.HostID,
		&r.AdminPasswordHash,
		&r.CreatedAt,
		&r.SkipAllowed,
		&r.DemocraticSkip,
		&r.SkipVoteThreshold,
		&r.MaxContinuousAdds,
		&r.RemoveOnPlay,
		&r.AllowDuplicates,
		&r.IsRequesterAdmin,
		&r.EnabledSources,
		&r.OnlyAdminAddPlaylistItems,
		&r.Public,
		&r.PlaylistImport,
		&r.IsGenerating,
		&r.GenerationCount,
		&r.GenerationError,
	)
	if err != nil {
		return fmt.Errorf("error scanning room row: %w", err)
	}

	return nil
}

func (r *roomRow) toRoomV2(enabledProviders []string) (*vibe.RoomV2, error) {
	defaultSettings, err := vibe.DefaultRoomSettings()
	if err != nil {
		return nil, fmt.Errorf("error getting default room settings in toRoom: %w", err)
	}

	storedSources := defaultSettings.EnabledSources
	if r.EnabledSources.Valid {
		storedSources = []string{}
		if r.EnabledSources.String != "" {
			storedSources = strings.Split(r.EnabledSources.String, ",")
		}
	}

	settings, err := r.toRoomSettingsV2(storedSources, enabledProviders)
	if err != nil {
		return nil, fmt.Errorf("error converting room settings: %w", err)
	}

	return &vibe.RoomV2{
		ID:                   r.ID.String,
		Name:                 r.Name.String,
		Mode:                 r.Mode.String,
		RoomType:             vibe.RoomType(r.RoomType.String),
		HostID:               r.HostID.String,
		AdminPasswordHash:    r.AdminPasswordHash.String,
		HasPassword:          r.AdminPasswordHash.Valid && r.AdminPasswordHash.String != "",
		IsAdmin:              r.IsRequesterAdmin.Bool,
		Settings:             *settings,
		CreatedAt:            r.CreatedAt.Time,
		IsGenerating:         r.IsGenerating.Bool,
		GenerationCount:      int(r.GenerationCount.Int64),
		GenerationError:      vibe.PublicGenerationError(r.GenerationError.String),
		StoredEnabledSources: storedSources,
	}, nil
}

func (r *roomRow) toRoomSettingsV2(
	storedSources []string,
	enabledProviders []string,
) (*vibe.RoomSettingsV2, error) {
	sources := []string{}
	for _, source := range storedSources {
		for _, provider := range enabledProviders {
			if source == provider {
				sources = append(sources, source)
				break
			}
		}
	}

	return &vibe.RoomSettingsV2{
		SkipAllowed:               r.SkipAllowed.Bool,
		DemocraticSkip:            r.DemocraticSkip.Bool,
		SkipVoteThreshold:         r.SkipVoteThreshold.Float64,
		MaxContinuousAdds:         int(r.MaxContinuousAdds.Int64),
		RemoveOnPlay:              r.RemoveOnPlay.Bool,
		AllowDuplicates:           r.AllowDuplicates.Bool,
		EnabledSources:            sources,
		OnlyAdminAddPlaylistItems: r.OnlyAdminAddPlaylistItems.Bool,
		Public:                    r.Public.Bool,
		PlaylistImport:            r.PlaylistImport.Bool,
	}, nil
}

func (c *Client) prepareGetActiveSourcesStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT DISTINCT source_type
		FROM playlist_items
		WHERE room_id = $1
		AND source_type = ANY($2::text[])
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetActiveSourcesStatement: %w", err)
	}
	c.GetActiveSourcesStatement = stmt
	return nil
}

func (c *Client) fillActiveSources(ctx context.Context, room vibe.RoomV2) (*vibe.RoomV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "fillActiveSources")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.GetActiveSourcesStatement.QueryContext(
		cctx,
		room.ID,
		c.enabledProviders,
	)
	if err != nil {
		return nil, fmt.Errorf("error in db: get active sources: %w", err)
	}
	defer rows.Close()

	sources := []string{}
	for rows.Next() {
		var row activeSourceRow
		err := row.scanRows(rows)
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

func (a *activeSourceRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(&a.Source)
	if err != nil {
		return fmt.Errorf("error scanning active source row: %w", err)
	}

	return nil
}

func (a *activeSourceRow) toSource() string {
	return a.Source.String
}

// prepareCreateRoomV2Stmt prepares the CreateRoomV2Statement.
func (c *Client) prepareCreateRoomV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		WITH valid_name_q AS (
			SELECT $1::TEXT AS id
			WHERE NOT EXISTS (
				SELECT 1
				FROM rooms
				WHERE rooms.id = $1
			)
			AND (
				(
					$17 != ''
					AND EXISTS (
						SELECT 1
						FROM room_name_reservations
						WHERE room_name_reservations.name = $1
						AND room_name_reservations.token::TEXT = $17
						AND room_name_reservations.owner_id = $4
						AND room_name_reservations.expires_at >
							CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
					)
				)
				OR (
					$17 = ''
					AND NOT EXISTS (
						SELECT 1
						FROM room_name_reservations
						WHERE room_name_reservations.name = $1
						AND room_name_reservations.expires_at >
							CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
					)
				)
			)
		),
		created_room_q AS (
			INSERT INTO rooms (id, name, mode, host_id, admin_password_hash, created_at, room_type)
			SELECT id, $2, $3, $4, $5, $6, $18
			FROM valid_name_q
			ON CONFLICT (id) DO NOTHING
			RETURNING id, room_type
		),
		created_settings_q AS (
			INSERT INTO room_settings (
				room_id,
				skip_allowed,
				democratic_skip,
				skip_vote_threshold,
				max_continuous_adds,
				remove_on_play,
				allow_duplicates,
				enabled_sources,
				only_admin_add_playlist_items,
				is_public,
				playlist_import
			)
			SELECT id, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16 FROM created_room_q
		),
		created_admin_q AS (
			INSERT INTO room_users (id, room_id, is_admin, is_active_listener, joined_at, last_seen_at)
			SELECT $4, id, TRUE, FALSE, $6, $6
			FROM created_room_q
			WHERE $4 != '' AND $5 != ''
		),
		consumed_name_q AS (
			INSERT INTO room_name_pool (name, generated, consumed_at)
			SELECT
				id,
				FALSE,
				CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
			FROM created_room_q
			ON CONFLICT (name) DO UPDATE
			SET consumed_at = EXCLUDED.consumed_at
		),
		deleted_reservation_q AS (
			DELETE FROM room_name_reservations
			USING created_room_q
			WHERE room_name_reservations.name = created_room_q.id
			AND (
				$17 = ''
				OR room_name_reservations.token::TEXT = $17
			)
		)
		SELECT id, room_type FROM created_room_q
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateRoomV2Statement: %w", err)
	}

	c.CreateRoomV2Statement = stmt

	return nil
}

// CreateRoom creates a new room and consumes its name reservation.
func (c *Client) CreateRoomV2(
	ctx context.Context,
	room *vibe.RoomV2,
	reservationToken string,
) (*vibe.RoomV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateRoom")
	defer span.End()

	roomType := room.RoomType
	if roomType == "" {
		roomType = vibe.RoomTypeMusic
	}

	if !roomType.IsValid() {
		return nil, fmt.Errorf("error creating room: invalid room type %q", roomType)
	}

	for _, source := range room.Settings.EnabledSources {
		if !roomType.AllowsSource(source) {
			return nil, fmt.Errorf("error creating room: provider %q is not allowed for %s", source, roomType)
		}
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.CreateRoomV2Statement.QueryRowContext(cctx,
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
		room.Settings.AllowDuplicates,
		strings.Join(room.Settings.EnabledSources, ","),
		room.Settings.OnlyAdminAddPlaylistItems,
		room.Settings.Public,
		room.Settings.PlaylistImport,
		reservationToken,
		roomType,
	)

	var scanned createRoomRow
	err := scanned.scan(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrRoomNameUnavailable{
				Err: fmt.Errorf("error room name reservation is unavailable or expired"),
			}
		}

		return nil, fmt.Errorf("error creating room: %w", err)
	}

	createdRoom := *room
	createdRoom.RoomType = vibe.RoomType(scanned.RoomType.String)
	createdRoom.UserID = room.HostID
	createdRoom.IsAdmin = room.HostID != "" && room.AdminPasswordHash != ""
	createdSettings := room.Settings
	createdSettings.EnabledSources = []string{}
	for _, source := range room.Settings.EnabledSources {
		for _, provider := range c.enabledProviders {
			if source == provider {
				createdSettings.EnabledSources = append(
					createdSettings.EnabledSources,
					source,
				)
				break
			}
		}
	}
	createdRoom.Settings = createdSettings
	createdRoom.ActiveSources = []string{}
	createdRoom.RoomGenerationMaxDailyCount = c.roomGenerationMaxDailyCount
	createdRoom.RoomGenerationMaxExistingPlaylistItems = c.roomGenerationMaxExistingPlaylistItems
	createdRoom.StoredEnabledSources = append(
		[]string{},
		room.Settings.EnabledSources...,
	)

	return &createdRoom, nil
}

type createRoomRow struct {
	ID       sql.NullString
	RoomType sql.NullString
}

func (r *createRoomRow) scan(row *sql.Row) error {
	err := row.Scan(&r.ID, &r.RoomType)
	if err != nil {
		return fmt.Errorf("error scanning created room row: %w", err)
	}

	return nil
}

// prepareUpdateRoomV2Stmt prepares the UpdateRoomV2Statement.
func (c *Client) prepareUpdateRoomV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		WITH updated_room_q AS (
			UPDATE rooms
			SET name = $1,
			mode = $9,
			host_id = $10,
			admin_password_hash = $11
			WHERE id = $2
			RETURNING id, room_type
		)
		UPDATE room_settings
		SET skip_allowed = $3,
		democratic_skip = $4,
		skip_vote_threshold = $5,
		max_continuous_adds = $6,
		remove_on_play = $7,
		allow_duplicates = $8,
		enabled_sources = CASE
			WHEN a.room_type = 'WATCH' THEN
				CASE WHEN 'youtube' = ANY(string_to_array($12, ',')) THEN 'youtube' ELSE '' END
			ELSE $12
		END,
		only_admin_add_playlist_items = $13,
		is_public = $14,
		playlist_import = $15
		FROM updated_room_q a
		WHERE room_settings.room_id = a.id
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdateRoomV2Statement: %w", err)
	}

	c.UpdateRoomV2Statement = stmt

	return nil
}

// UpdateRoom updates an existing room.
func (c *Client) UpdateRoomV2(ctx context.Context, room *vibe.RoomV2) (*vibe.RoomV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdateRoom")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	enabledSources := []string{}
	for _, source := range room.Settings.EnabledSources {
		for _, provider := range c.enabledProviders {
			if source == provider {
				enabledSources = append(enabledSources, source)
				break
			}
		}
	}
	for _, source := range room.StoredEnabledSources {
		providerEnabled := false
		for _, provider := range c.enabledProviders {
			if source == provider {
				providerEnabled = true
				break
			}
		}
		if !providerEnabled {
			enabledSources = append(enabledSources, source)
		}
	}

	_, err := c.UpdateRoomV2Statement.ExecContext(cctx,
		room.Name,
		room.ID,
		room.Settings.SkipAllowed,
		room.Settings.DemocraticSkip,
		room.Settings.SkipVoteThreshold,
		room.Settings.MaxContinuousAdds,
		room.Settings.RemoveOnPlay,
		room.Settings.AllowDuplicates,
		room.Mode,
		room.HostID,
		room.AdminPasswordHash,
		strings.Join(enabledSources, ","),
		room.Settings.OnlyAdminAddPlaylistItems,
		room.Settings.Public,
		room.Settings.PlaylistImport,
	)
	if err != nil {
		return nil, fmt.Errorf("error updating room: %w", err)
	}

	updatedRoom, err := c.GetRoomV2(ctx, room.ID, room.UserID)
	if err != nil {
		return nil, fmt.Errorf("error fetching updated room: %w", err)
	}

	return updatedRoom, nil
}

const postgresUniqueViolation = "23505"

func (c *Client) prepareReserveRoomNameStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH inserted_pool_q AS (
			INSERT INTO room_name_pool (name, generated)
			VALUES ($1, FALSE)
			ON CONFLICT (name) DO NOTHING
			RETURNING name, consumed_at
		),
		pool_q AS (
			SELECT name, consumed_at
			FROM inserted_pool_q

			UNION ALL

			SELECT name, consumed_at
			FROM room_name_pool
			WHERE name = $1
			AND NOT EXISTS (SELECT 1 FROM inserted_pool_q)
		),
		candidate_q AS (
			SELECT pool_q.name
			FROM pool_q
			WHERE pool_q.consumed_at IS NULL
			AND NOT EXISTS (
				SELECT 1
				FROM rooms
				WHERE rooms.id = pool_q.name
			)
			AND NOT EXISTS (
				SELECT 1
				FROM room_name_reservations
				WHERE room_name_reservations.name = pool_q.name
				AND room_name_reservations.owner_id != $2
				AND room_name_reservations.expires_at >
					CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
			)
		),
		deleted_owned_q AS (
			DELETE FROM room_name_reservations
			USING candidate_q
			WHERE room_name_reservations.owner_id = $2
			AND room_name_reservations.name != candidate_q.name
			RETURNING room_name_reservations.name
		),
		reserved_q AS (
			INSERT INTO room_name_reservations (
				name,
				owner_id,
				expires_at
			)
			SELECT
				candidate_q.name,
				$2,
				(CURRENT_TIMESTAMP AT TIME ZONE 'UTC') + ($3 * INTERVAL '1 second')
			FROM candidate_q
			CROSS JOIN (
				SELECT COUNT(*) AS deleted_count
				FROM deleted_owned_q
			) deleted_owned_count_q
			ON CONFLICT (name) DO UPDATE
			SET
				token = gen_random_uuid(),
				owner_id = EXCLUDED.owner_id,
				created_at = CURRENT_TIMESTAMP AT TIME ZONE 'UTC',
				expires_at = EXCLUDED.expires_at
			WHERE room_name_reservations.expires_at <= CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
			OR room_name_reservations.owner_id = EXCLUDED.owner_id
			RETURNING name, token, expires_at
		)
		SELECT name, token, expires_at
		FROM reserved_q
	`)
	if err != nil {
		return fmt.Errorf("error preparing ReserveRoomNameStatement: %w", err)
	}

	c.ReserveRoomNameStatement = stmt

	return nil
}

func (c *Client) ReserveRoomName(
	ctx context.Context,
	name string,
	ownerID string,
) (*vibe.RoomNameReservation, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ReserveRoomName")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.ReserveRoomNameStatement.QueryRowContext(
		cctx,
		name,
		ownerID,
		int64(c.roomNameReservationTTL/time.Second),
	)

	var rowData roomNameReservationRow
	err := rowData.scan(row)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.Code == postgresUniqueViolation {
			return nil, internalerror.ErrRoomNameUnavailable{
				Err: fmt.Errorf("error room name reservation conflict: %w", err),
			}
		}

		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrRoomNameUnavailable{
				Err: fmt.Errorf("error room name is unavailable"),
			}
		}

		return nil, fmt.Errorf("error scanning room name reservation: %w", err)
	}

	reservation, err := rowData.toRoomNameReservation()
	if err != nil {
		return nil, fmt.Errorf("error mapping room name reservation in ReserveRoomName: %w", err)
	}

	return reservation, nil
}

func (c *Client) prepareReserveSuggestedRoomNameStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH maximum_q AS (
			SELECT MAX(id) AS maximum_id
			FROM room_name_pool
		),
		start_q AS (
			SELECT GREATEST(
				1,
				FLOOR(RANDOM() * maximum_q.maximum_id)::BIGINT
			) AS start_id
			FROM maximum_q
		),
		after_start_q AS (
			SELECT pool_q.id, pool_q.name
			FROM room_name_pool pool_q
			CROSS JOIN start_q
			WHERE pool_q.id >= start_q.start_id
			AND pool_q.generated
			AND pool_q.consumed_at IS NULL
			AND NOT EXISTS (
				SELECT 1
				FROM room_name_reservations reservation_q
				WHERE reservation_q.name = pool_q.name
				AND reservation_q.expires_at > CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
			)
			ORDER BY pool_q.id
			FOR UPDATE OF pool_q SKIP LOCKED
			LIMIT 1
		),
		before_start_q AS (
			SELECT pool_q.id, pool_q.name
			FROM room_name_pool pool_q
			CROSS JOIN start_q
			WHERE pool_q.id < start_q.start_id
			AND pool_q.generated
			AND pool_q.consumed_at IS NULL
			AND NOT EXISTS (SELECT 1 FROM after_start_q)
			AND NOT EXISTS (
				SELECT 1
				FROM room_name_reservations reservation_q
				WHERE reservation_q.name = pool_q.name
				AND reservation_q.expires_at > CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
			)
			ORDER BY pool_q.id
			FOR UPDATE OF pool_q SKIP LOCKED
			LIMIT 1
		),
		candidate_q AS (
			SELECT id, name
			FROM after_start_q

			UNION ALL

			SELECT id, name
			FROM before_start_q
		),
		deleted_owned_q AS (
			DELETE FROM room_name_reservations
			USING candidate_q
			WHERE room_name_reservations.owner_id = $1
			AND room_name_reservations.name != candidate_q.name
			RETURNING room_name_reservations.name
		),
		reserved_q AS (
			INSERT INTO room_name_reservations (
				name,
				owner_id,
				expires_at
			)
			SELECT
				candidate_q.name,
				$1,
				(CURRENT_TIMESTAMP AT TIME ZONE 'UTC') + ($2 * INTERVAL '1 second')
			FROM candidate_q
			CROSS JOIN (
				SELECT COUNT(*) AS deleted_count
				FROM deleted_owned_q
			) deleted_owned_count_q
			ON CONFLICT (name) DO UPDATE
			SET
				token = gen_random_uuid(),
				owner_id = EXCLUDED.owner_id,
				created_at = CURRENT_TIMESTAMP AT TIME ZONE 'UTC',
				expires_at = EXCLUDED.expires_at
			WHERE room_name_reservations.expires_at <= CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
			RETURNING name, token, expires_at
		)
		SELECT name, token, expires_at
		FROM reserved_q
	`)
	if err != nil {
		return fmt.Errorf("error preparing ReserveSuggestedRoomNameStatement: %w", err)
	}

	c.ReserveSuggestedRoomNameStatement = stmt

	return nil
}

func (c *Client) ReserveSuggestedRoomName(
	ctx context.Context,
	ownerID string,
) (*vibe.RoomNameReservation, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ReserveSuggestedRoomName")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.ReserveSuggestedRoomNameStatement.QueryRowContext(
		cctx,
		ownerID,
		int64(c.roomNameReservationTTL/time.Second),
	)

	var rowData roomNameReservationRow
	err := rowData.scan(row)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.Code == postgresUniqueViolation {
			return nil, internalerror.ErrRoomNameUnavailable{
				Err: fmt.Errorf("error room name reservation conflict: %w", err),
			}
		}

		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerror.ErrRoomNameUnavailable{
				Err: fmt.Errorf("error no room names are available"),
			}
		}

		return nil, fmt.Errorf("error scanning suggested room name reservation: %w", err)
	}

	reservation, err := rowData.toRoomNameReservation()
	if err != nil {
		return nil, fmt.Errorf("error mapping suggested reservation in ReserveSuggestedRoomName: %w", err)
	}

	return reservation, nil
}

type roomNameReservationRow struct {
	Name      string
	Token     string
	ExpiresAt time.Time
}

func (r *roomNameReservationRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.Name,
		&r.Token,
		&r.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("error scanning room name reservation row: %w", err)
	}

	return nil
}

func (r *roomNameReservationRow) toRoomNameReservation() (*vibe.RoomNameReservation, error) {
	return &vibe.RoomNameReservation{
		Name:      r.Name,
		Token:     r.Token,
		ExpiresAt: r.ExpiresAt,
	}, nil
}

func (c *Client) prepareDeleteExpiredRoomNameReservationsStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM room_name_reservations
		WHERE expires_at <= CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
	`)
	if err != nil {
		return fmt.Errorf(
			"error preparing DeleteExpiredRoomNameReservationsStatement: %w",
			err,
		)
	}

	c.DeleteExpiredRoomNameReservationsStatement = stmt

	return nil
}

func (c *Client) DeleteExpiredRoomNameReservations(
	ctx context.Context,
) (int, error) {
	span, ctx := tracing.StartSpanFromContext(
		ctx,
		"DeleteExpiredRoomNameReservations",
	)
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.DeleteExpiredRoomNameReservationsStatement.ExecContext(cctx)
	if err != nil {
		return 0, fmt.Errorf("error deleting expired room name reservations: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf(
			"error getting deleted room name reservation count: %w",
			err,
		)
	}

	count := int(deleted)
	return count, nil
}

func (c *Client) prepareSearchPublicRoomsV3Stmt() error {
	stmt, err := c.DB.Prepare(`
		WITH public_rooms_q AS (
			SELECT a.id, a.name, a.room_type
			FROM rooms a
			JOIN room_settings b ON b.room_id = a.id
			WHERE b.is_public
			AND a.room_type = $7
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
				a.room_type,
				CASE
					WHEN b.listeners = 0 AND b.receivers > 0 THEN 1
					ELSE COALESCE(b.listeners, 0)
				END AS listener_count
			FROM public_rooms_q a
			LEFT JOIN participants_q b ON b.room_id = a.id
		),
		filtered_q AS (
			SELECT a.id, a.name, a.room_type, a.listener_count
			FROM listeners_q a
			WHERE NOT $3 OR a.listener_count > 0
		),
		page_q AS (
			SELECT
				a.id,
				a.name,
				a.room_type,
				a.listener_count,
				(
					SELECT COUNT(*)
					FROM playlist_items b
					WHERE b.room_id = a.id
					AND b.source_type = ANY($4::text[])
				) AS playlist_item_count
			FROM filtered_q a
			ORDER BY a.listener_count DESC, playlist_item_count DESC, a.id DESC
			OFFSET $5 LIMIT $6
		),
		totals_q AS (
			SELECT COUNT(*) AS total FROM filtered_q
		)
		SELECT b.id, b.name, b.room_type, b.listener_count, b.playlist_item_count, a.total
		FROM totals_q a
		LEFT JOIN page_q b ON TRUE
		ORDER BY b.listener_count DESC, b.playlist_item_count DESC, b.id DESC
	`)
	if err != nil {
		return fmt.Errorf("error preparing SearchPublicRoomsV3Statement: %w", err)
	}

	c.SearchPublicRoomsV3Statement = stmt

	return nil
}

func (c *Client) SearchPublicRoomsV3(
	ctx context.Context,
	search vibe.PublicRoomSearch,
) (*vibe.PublicRoomResultV3, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SearchPublicRoomsV3")
	defer span.End()

	roomType := search.RoomType
	if roomType == "" {
		roomType = vibe.RoomTypeMusic
	}

	if !roomType.IsValid() {
		return nil, fmt.Errorf("error searching public rooms: invalid room type %q", roomType)
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Treat LIKE metacharacters as part of the room name.
	query := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search.Query)
	rows, err := c.SearchPublicRoomsV3Statement.QueryContext(
		cctx,
		query,
		time.Now().UTC().Add(-15*time.Second),
		search.Live,
		c.enabledProviders,
		search.From,
		search.To-search.From+1,
		roomType,
	)
	if err != nil {
		return nil, fmt.Errorf("error searching public rooms: %w", err)
	}

	defer rows.Close()

	result := &vibe.PublicRoomResultV3{
		Rooms: []vibe.PublicRoomV3{},
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

		room, err := row.toPublicRoomV3()
		if err != nil {
			return nil, fmt.Errorf("error converting public room in SearchPublicRoomsV3: %w", err)
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

func (r *publicRoomRow) toPublicRoomV3() (*vibe.PublicRoomV3, error) {
	return &vibe.PublicRoomV3{
		ID:                r.ID.String,
		Name:              r.Name.String,
		RoomType:          vibe.RoomType(r.RoomType.String),
		ListenerCount:     int(r.ListenerCount.Int64),
		PlaylistItemCount: int(r.PlaylistItemCount.Int64),
	}, nil
}

func (r *publicRoomResultRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(&r.ID, &r.Name, &r.RoomType, &r.ListenerCount, &r.PlaylistItemCount, &r.Total)
	if err != nil {
		return fmt.Errorf("error scanning public room result row: %w", err)
	}

	return nil
}
