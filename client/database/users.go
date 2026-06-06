package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/opentracing"
	"github.com/zoff-music/vibes-backend/vibe"
	"golang.org/x/crypto/bcrypt"
)

// prepareGetUserStmt prepares the GetUserStatement.
func (c *Client) prepareGetUserStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, room_id, is_admin, joined_at, last_seen_at
		FROM room_users
		WHERE room_id = $1 AND id = $2
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

	r := c.GetUserStatement.QueryRowContext(cctx, roomID, userID)

	var row userRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.User{}, nil
		}

		return nil, fmt.Errorf("error fetching user: %w", err)
	}

	user, err := row.toUser()
	if err != nil {
		return nil, fmt.Errorf("error converting user row: %w", err)
	}

	return user, nil
}

type userRow struct {
	ID         sql.NullString
	RoomID     sql.NullString
	IsAdmin    sql.NullInt64
	JoinedAt   sql.NullTime
	LastSeenAt sql.NullTime
}

func (r *userRow) scan(row *sql.Row) error {
	return row.Scan(
		&r.ID,
		&r.RoomID,
		&r.IsAdmin,
		&r.JoinedAt,
		&r.LastSeenAt,
	)
}

func (r *userRow) toUser() (*vibe.User, error) {
	return &vibe.User{
		ID:         r.ID.String,
		RoomID:     r.RoomID.String,
		IsAdmin:    r.IsAdmin.Valid && r.IsAdmin.Int64 == 1,
		JoinedAt:   r.JoinedAt.Time,
		LastSeenAt: r.LastSeenAt.Time,
	}, nil
}

// prepareCreateUserStmt prepares the CreateUserStatement.
func (c *Client) prepareCreateUserStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO room_users (id, room_id, is_admin, is_active_listener, joined_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT(id, room_id) DO UPDATE SET
		is_admin = EXCLUDED.is_admin,
		is_active_listener = EXCLUDED.is_active_listener,
		last_seen_at = EXCLUDED.last_seen_at
		RETURNING id
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateUserStatement: %w", err)
	}

	c.CreateUserStatement = stmt

	return nil
}

// CreateUser creates or updates a user session in a room.
func (c *Client) CreateUser(ctx context.Context, user *vibe.User) (*vibe.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CreateUser")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.CreateUserStatement.QueryRowContext(cctx,
		user.ID,
		user.RoomID,
		boolToInt(user.IsAdmin),
		0, // is_active_listener (default to 0 for now)
		user.JoinedAt,
		user.LastSeenAt,
	)

	var returnedID sql.NullString
	err := r.Scan(&returnedID)
	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	user.ID = returnedID.String

	return user, nil
}

// AuthenticateAdmin handles password verification and admin elevation.
func (c *Client) AuthenticateAdmin(ctx context.Context, roomID, userID, password string) (*vibe.AdminAuthResult, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "AuthenticateAdmin")
	defer span.Finish()

	room, err := c.GetRoom(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting room in authenticate admin: %w", err)
	}

	if room.IsEmpty() {
		return nil, fmt.Errorf("room not found in authenticate admin")
	}

	isFirstTimeSetup := false

	// Handle initial password setup
	if !room.HasPassword {
		isFirstTimeSetup = true
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		room.AdminPasswordHash = string(hash)
		_, err = c.UpdateRoom(ctx, room)
		if err != nil {
			return nil, fmt.Errorf("failed to update room password: %w", err)
		}
	}

	// Verify existing or newly set password
	err = bcrypt.CompareHashAndPassword([]byte(room.AdminPasswordHash), []byte(password))
	if err != nil {
		return &vibe.AdminAuthResult{
			IsAdmin:          false,
			IsFirstTimeSetup: isFirstTimeSetup,
		}, nil
	}

	// Elevate user to admin
	log.Printf("AuthenticateAdmin: successfully verified password for userID=%s", userID)
	user, err := c.GetUser(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting user in authenticate admin: %w", err)
	}

	if user.IsEmpty() {
		user = &vibe.User{
			ID:       userID,
			RoomID:   roomID,
			JoinedAt: time.Now(),
		}
	}

	user.IsAdmin = true
	user.LastSeenAt = time.Now()

	log.Printf("AuthenticateAdmin: elevating userID=%s in roomID=%s", userID, roomID)
	_, err = c.CreateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("error creating user in authenticate admin: %w", err)
	}

	return &vibe.AdminAuthResult{
		IsAdmin:          true,
		IsFirstTimeSetup: isFirstTimeSetup,
	}, nil
}
