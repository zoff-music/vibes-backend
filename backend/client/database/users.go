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
		WHERE room_id = ? AND id = ?
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

	var row userRow

	err := c.GetUserStatement.QueryRowContext(cctx, roomID, userID).Scan(
		&row.ID,
		&row.RoomID,
		&row.Nickname,
		&row.IsAdmin,
		&row.JoinedAt,
		&row.LastSeenAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.User{}, nil
		}

		return nil, fmt.Errorf("error fetching user: %w", err)
	}

	user := row.toUser()

	return &user, nil
}

// prepareGetUsersInRoomStmt prepares the GetUsersInRoomStatement.
func (c *Client) prepareGetUsersInRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, room_id, nickname, is_admin, joined_at, last_seen_at
		FROM room_users
		WHERE room_id = ?
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

		err := row.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning user row: %w", err)
		}

		users = append(users, row.toUser())
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}

// prepareCountUsersInRoomStmt prepares the CountUsersInRoomStatement.
func (c *Client) prepareCountUsersInRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT COUNT(*) FROM room_users WHERE room_id = ?
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

	var count int

	err := c.CountUsersInRoomStatement.QueryRowContext(cctx, roomID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting users: %w", err)
	}

	return count, nil
}

// prepareCreateUserStmt prepares the CreateUserStatement.
func (c *Client) prepareCreateUserStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO room_users (id, room_id, nickname, is_admin, joined_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?)
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

	_, err := c.CreateUserStatement.ExecContext(cctx,
		user.ID,
		user.RoomID,
		user.Nickname,
		isAdmin,
		user.JoinedAt,
		user.LastSeenAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	return user, nil
}

// prepareUpdateUserLastSeenStmt prepares the UpdateUserLastSeenStatement.
func (c *Client) prepareUpdateUserLastSeenStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE room_users SET last_seen_at = ? WHERE room_id = ? AND id = ?
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
		DELETE FROM room_users WHERE room_id = ? AND id = ?
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
		DELETE FROM room_users WHERE room_id = ? AND last_seen_at < ?
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

// --- Internal types and helpers ---

type userRow struct {
	ID         string
	RoomID     string
	Nickname   sql.NullString
	IsAdmin    int
	JoinedAt   time.Time
	LastSeenAt time.Time
}

func (r *userRow) scan(rows *sql.Rows) error {
	return rows.Scan(
		&r.ID,
		&r.RoomID,
		&r.Nickname,
		&r.IsAdmin,
		&r.JoinedAt,
		&r.LastSeenAt,
	)
}

func (r *userRow) toUser() vibe.User {
	var nickname *string
	if r.Nickname.Valid {
		nickname = &r.Nickname.String
	}

	return vibe.User{
		ID:         r.ID,
		RoomID:     r.RoomID,
		Nickname:   nickname,
		IsAdmin:    r.IsAdmin == 1,
		JoinedAt:   r.JoinedAt,
		LastSeenAt: r.LastSeenAt,
	}
}
