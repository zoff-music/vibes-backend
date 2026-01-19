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
