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

// prepareGetUserStmt prepares the GetUserStatement.
func (c *Client) prepareGetUserStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, room_id, nickname, is_admin, joined_at, last_seen_at
		FROM room_users
		WHERE room_id = ?1 AND id = ?2
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetUserStatement: %w", err)
	}

	c.GetUserStatement = stmt

	return nil
}

// GetUser fetches a user by ID.
func (c *Client) GetUser(ctx context.Context, roomID, userID string) (*vibe.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetUser")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetUserStatement.QueryRowContext(cctx, roomID, userID)

	var scanned userRow

	err := scanned.scan(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.User{}, nil
		}

		return nil, fmt.Errorf("error fetching user: %w", err)
	}

	user, err := scanned.toUser()
	if err != nil {
		return nil, fmt.Errorf("error converting user row: %w", err)
	}

	return user, nil
}

// prepareGetUsersInRoomStmt prepares the GetUsersInRoomStatement.
func (c *Client) prepareGetUsersInRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, room_id, nickname, is_admin, joined_at, last_seen_at
		FROM room_users
		WHERE room_id = ?1
		ORDER BY joined_at ASC
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetUsersInRoomStatement: %w", err)
	}

	c.GetUsersInRoomStatement = stmt

	return nil
}

// GetUsersInRoom fetches all users in a room.
func (c *Client) GetUsersInRoom(ctx context.Context, roomID string) ([]vibe.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetUsersInRoom")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.GetUsersInRoomStatement.QueryContext(cctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error fetching users: %w", err)
	}
	defer rows.Close()

	var users []vibe.User

	for rows.Next() {
		var row userRow

		err := row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning user row: %w", err)
		}

		user, err := row.toUser()
		if err != nil {
			return nil, fmt.Errorf("error converting user row: %w", err)
		}

		users = append(users, *user)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}

type userRow struct {
	ID         sql.NullString
	RoomID     sql.NullString
	Nickname   sql.NullString
	IsAdmin    sql.NullInt64
	JoinedAt   sql.NullTime
	LastSeenAt sql.NullTime
}

func (r *userRow) scanRows(rows *sql.Rows) error {
	return rows.Scan(
		&r.ID,
		&r.RoomID,
		&r.Nickname,
		&r.IsAdmin,
		&r.JoinedAt,
		&r.LastSeenAt,
	)
}

func (r *userRow) scan(row *sql.Row) error {
	return row.Scan(
		&r.ID,
		&r.RoomID,
		&r.Nickname,
		&r.IsAdmin,
		&r.JoinedAt,
		&r.LastSeenAt,
	)
}

func (r *userRow) toUser() (*vibe.User, error) {
	var nickname *string
	if r.Nickname.Valid {
		nickname = &r.Nickname.String
	}

	if !r.JoinedAt.Valid {
		return nil, fmt.Errorf("error missing user joined_at")
	}

	if !r.LastSeenAt.Valid {
		return nil, fmt.Errorf("error missing user last_seen_at")
	}

	return &vibe.User{
		ID:         r.ID.String,
		RoomID:     r.RoomID.String,
		Nickname:   nickname,
		IsAdmin:    r.IsAdmin.Valid && r.IsAdmin.Int64 == 1,
		JoinedAt:   r.JoinedAt.Time,
		LastSeenAt: r.LastSeenAt.Time,
	}, nil
}

// prepareCountUsersInRoomStmt prepares the CountUsersInRoomStatement.
func (c *Client) prepareCountUsersInRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT COUNT(*) FROM room_users WHERE room_id = ?1
	`)
	if err != nil {
		return fmt.Errorf("error preparing CountUsersInRoomStatement: %w", err)
	}

	c.CountUsersInRoomStatement = stmt

	return nil
}

// CountUsersInRoom counts the number of users in a room.
func (c *Client) CountUsersInRoom(ctx context.Context, roomID string) (int, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CountUsersInRoom")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.CountUsersInRoomStatement.QueryRowContext(cctx, roomID)

	var scanned countUsersInRoomRow

	err := scanned.scan(row)
	if err != nil {
		return 0, fmt.Errorf("error counting users: %w", err)
	}

	return scanned.toCount(), nil
}

type countUsersInRoomRow struct {
	Count sql.NullInt64
}

func (r *countUsersInRoomRow) scan(row *sql.Row) error {
	return row.Scan(&r.Count)
}

func (r *countUsersInRoomRow) toCount() int {
	if !r.Count.Valid {
		return 0
	}

	return int(r.Count.Int64)
}

// prepareCreateUserStmt prepares the CreateUserStatement.
func (c *Client) prepareCreateUserStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO room_users (room_id, nickname, is_admin, joined_at, last_seen_at)
		VALUES (?1, ?2, ?3, ?4, ?5)
		RETURNING id
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateUserStatement: %w", err)
	}

	c.CreateUserStatement = stmt

	return nil
}

// CreateUser creates a new user session in a room.
func (c *Client) CreateUser(ctx context.Context, user *vibe.User) (*vibe.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CreateUser")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	isAdmin := 0
	if user.IsAdmin {
		isAdmin = 1
	}

	var returnedID string

	err := c.CreateUserStatement.QueryRowContext(cctx,
		// user.ID is removed
		user.RoomID,
		user.Nickname,
		isAdmin,
		user.JoinedAt,
		user.LastSeenAt,
	).Scan(&returnedID)
	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	user.ID = returnedID

	return user, nil
}

// prepareUpdateUserLastSeenStmt prepares the UpdateUserLastSeenStatement.
func (c *Client) prepareUpdateUserLastSeenStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE room_users SET last_seen_at = ?1 WHERE room_id = ?2 AND id = ?3
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdateUserLastSeenStatement: %w", err)
	}

	c.UpdateUserLastSeenStatement = stmt

	return nil
}

// UpdateUserLastSeen updates the last seen time for a user.
func (c *Client) UpdateUserLastSeen(ctx context.Context, roomID, userID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UpdateUserLastSeen")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.UpdateUserLastSeenStatement.ExecContext(cctx, time.Now(), roomID, userID)
	if err != nil {
		return fmt.Errorf("error updating user last seen: %w", err)
	}

	return nil
}

// prepareRemoveUserStmt prepares the RemoveUserStatement.
func (c *Client) prepareRemoveUserStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM room_users WHERE room_id = ?1 AND id = ?2
	`)
	if err != nil {
		return fmt.Errorf("error preparing RemoveUserStatement: %w", err)
	}

	c.RemoveUserStatement = stmt

	return nil
}

// RemoveUser removes a user from a room.
func (c *Client) RemoveUser(ctx context.Context, roomID, userID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RemoveUser")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.RemoveUserStatement.ExecContext(cctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("error removing user: %w", err)
	}

	return nil
}

// prepareCleanupInactiveUsersStmt prepares the CleanupInactiveUsersStatement.
func (c *Client) prepareCleanupInactiveUsersStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM room_users WHERE room_id = ?1 AND last_seen_at < ?2
	`)
	if err != nil {
		return fmt.Errorf("error preparing CleanupInactiveUsersStatement: %w", err)
	}

	c.CleanupInactiveUsersStatement = stmt

	return nil
}

// CleanupInactiveUsers removes users who haven't been seen within the threshold.
func (c *Client) CleanupInactiveUsers(ctx context.Context, roomID string, threshold time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CleanupInactiveUsers")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-threshold)

	_, err := c.CleanupInactiveUsersStatement.ExecContext(cctx, roomID, cutoff)
	if err != nil {
		return fmt.Errorf("error cleaning up inactive users: %w", err)
	}

	return nil
}
