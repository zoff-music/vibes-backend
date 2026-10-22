package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
	"golang.org/x/crypto/bcrypt"
)

// prepareGetUserStmt prepares the GetUserStatement.
func (c *Client) prepareGetUserStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT a.id, a.room_id,
			COALESCE(a.is_admin AND a.admin_password_hash = b.admin_password_hash AND b.admin_password_hash != '', FALSE),
			a.joined_at, a.last_seen_at
		FROM room_users a
		JOIN rooms b ON b.id = a.room_id
		WHERE a.room_id = $1 AND a.id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetUserStatement: %w", err)
	}

	c.GetUserStatement = stmt

	return nil
}

// GetUser fetches a user by ID.
func (c *Client) GetUser(ctx context.Context, roomID, userID string) (*vibe.User, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetUser")
	defer span.End()

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
	IsAdmin    sql.NullBool
	JoinedAt   sql.NullTime
	LastSeenAt sql.NullTime
}

func (r *userRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.ID,
		&r.RoomID,
		&r.IsAdmin,
		&r.JoinedAt,
		&r.LastSeenAt,
	)
	if err != nil {
		return fmt.Errorf("error scanning user row: %w", err)
	}

	return nil
}

func (r *userRow) toUser() (*vibe.User, error) {
	return &vibe.User{
		ID:         r.ID.String,
		RoomID:     r.RoomID.String,
		IsAdmin:    r.IsAdmin.Bool,
		JoinedAt:   r.JoinedAt.Time,
		LastSeenAt: r.LastSeenAt.Time,
	}, nil
}

func (c *Client) prepareGrantRoomAdminStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO room_users (id, room_id, is_admin, admin_password_hash, is_active_listener, joined_at, last_seen_at)
		SELECT $2, a.id, TRUE, $3, FALSE, NOW(), NOW()
		FROM rooms a
		WHERE a.id = $1 AND a.admin_password_hash = $3 AND $3 != ''
		FOR UPDATE OF a
		ON CONFLICT(id, room_id) DO UPDATE SET
		is_admin = EXCLUDED.is_admin,
		admin_password_hash = EXCLUDED.admin_password_hash,
		last_seen_at = EXCLUDED.last_seen_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing GrantRoomAdminStatement: %w", err)
	}

	c.GrantRoomAdminStatement = stmt

	return nil
}

// GrantRoomAdmin binds access to the password that was actually verified.
func (c *Client) GrantRoomAdmin(ctx context.Context, roomID, userID, hash string) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GrantRoomAdmin")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.GrantRoomAdminStatement.ExecContext(cctx, roomID, userID, hash)
	if err != nil {
		return false, fmt.Errorf("error granting room admin: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error counting room admin grants: %w", err)
	}

	return count == 1, nil
}

func (c *Client) prepareClearRoomAdminStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE room_users
		SET is_admin = FALSE, admin_password_hash = NULL, last_seen_at = $3
		WHERE room_id = $1 AND id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing ClearRoomAdminStatement: %w", err)
	}

	c.ClearRoomAdminStatement = stmt

	return nil
}

// ClearRoomAdmin removes a user's admin access in one room.
func (c *Client) ClearRoomAdmin(ctx context.Context, roomID string, userID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "ClearRoomAdmin")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.ClearRoomAdminStatement.ExecContext(cctx, roomID, userID, time.Now())
	if err != nil {
		return fmt.Errorf("error clearing room admin in ClearRoomAdmin: %w", err)
	}

	return nil
}

func (c *Client) prepareClaimRoomPasswordStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE rooms
		SET admin_password_hash = $2
		WHERE id = $1 AND COALESCE(admin_password_hash, '') = ''
	`)
	if err != nil {
		return fmt.Errorf("error preparing ClaimRoomPasswordStatement: %w", err)
	}

	c.ClaimRoomPasswordStatement = stmt

	return nil
}

func (c *Client) ClaimRoomPassword(ctx context.Context, roomID, hash string) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ClaimRoomPassword")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.ClaimRoomPasswordStatement.ExecContext(cctx, roomID, hash)
	if err != nil {
		return false, fmt.Errorf("error claiming room password: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error counting claimed room password: %w", err)
	}

	return count == 1, nil
}

// AuthenticateAdmin handles password verification and admin elevation.
func (c *Client) AuthenticateAdmin(ctx context.Context, roomID, userID, password string) (*vibe.AdminAuthResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AuthenticateAdmin")
	defer span.End()

	room, err := c.GetRoomV2(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting room in authenticate admin: %w", err)
	}

	if room.IsEmpty() {
		return nil, fmt.Errorf("error room not found in authenticate admin")
	}

	isFirstTimeSetup := false

	if !room.HasPassword {
		isFirstTimeSetup = true
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("error hashing password: %w", err)
		}
		claimed, err := c.ClaimRoomPassword(ctx, roomID, string(hash))
		if err != nil {
			return nil, fmt.Errorf("error claiming room password: %w", err)
		}

		if !claimed {
			return &vibe.AdminAuthResult{}, nil
		}

		room.AdminPasswordHash = string(hash)
	}

	err = bcrypt.CompareHashAndPassword([]byte(room.AdminPasswordHash), []byte(password))
	if err != nil {
		return &vibe.AdminAuthResult{
			IsAdmin:          false,
			IsFirstTimeSetup: isFirstTimeSetup,
		}, nil
	}

	granted, err := c.GrantRoomAdmin(ctx, roomID, userID, room.AdminPasswordHash)
	if err != nil {
		return nil, fmt.Errorf("error granting room admin in authenticate admin: %w", err)
	}

	return &vibe.AdminAuthResult{
		IsAdmin:          granted,
		IsFirstTimeSetup: isFirstTimeSetup,
	}, nil
}
