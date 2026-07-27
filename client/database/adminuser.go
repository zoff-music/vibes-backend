package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareGetAdminUserStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			id,
			username,
			password_hash,
			session_version,
			created_at,
			updated_at
		FROM admin_users
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetAdminUserStatement: %w", err)
	}

	c.GetAdminUserStatement = stmt

	return nil
}

func (c *Client) GetAdminUser(
	ctx context.Context,
	adminID string,
) (*vibe.AdminUser, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetAdminUser")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetAdminUserStatement.QueryRowContext(cctx, adminID)
	var rowData adminUserRow
	err := rowData.scanRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.AdminUser{}, nil
		}

		return nil, fmt.Errorf(
			"error scanning admin user in GetAdminUser: %w",
			err,
		)
	}

	admin := rowData.toAdminUser()

	return &admin, nil
}

type adminUserRow struct {
	ID             string
	Username       string
	PasswordHash   string
	SessionVersion int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (r *adminUserRow) scanRow(row *sql.Row) error {
	err := row.Scan(
		&r.ID,
		&r.Username,
		&r.PasswordHash,
		&r.SessionVersion,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error scanning admin user row: %w", err)
	}

	return nil
}

func (r *adminUserRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(
		&r.ID,
		&r.Username,
		&r.PasswordHash,
		&r.SessionVersion,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error scanning admin user rows: %w", err)
	}

	return nil
}

func (r *adminUserRow) toAdminUser() vibe.AdminUser {
	return vibe.AdminUser{
		ID:             r.ID,
		Username:       r.Username,
		PasswordHash:   r.PasswordHash,
		SessionVersion: r.SessionVersion,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func (c *Client) prepareGetAdminUserByUsernameStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			id,
			username,
			password_hash,
			session_version,
			created_at,
			updated_at
		FROM admin_users
		WHERE username = $1
	`)
	if err != nil {
		return fmt.Errorf(
			"error preparing GetAdminUserByUsernameStatement: %w",
			err,
		)
	}

	c.GetAdminUserByUsernameStatement = stmt

	return nil
}

func (c *Client) GetAdminUserByUsername(
	ctx context.Context,
	username string,
) (*vibe.AdminUser, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetAdminUserByUsername")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.GetAdminUserByUsernameStatement.QueryRowContext(cctx, username)
	var rowData adminUserRow
	err := rowData.scanRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.AdminUser{}, nil
		}

		return nil, fmt.Errorf(
			"error scanning admin user in GetAdminUserByUsername: %w",
			err,
		)
	}

	admin := rowData.toAdminUser()

	return &admin, nil
}

func (c *Client) prepareListAdminUsersStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			id,
			username,
			password_hash,
			session_version,
			created_at,
			updated_at
		FROM admin_users
		ORDER BY username
	`)
	if err != nil {
		return fmt.Errorf("error preparing ListAdminUsersStatement: %w", err)
	}

	c.ListAdminUsersStatement = stmt

	return nil
}

func (c *Client) ListAdminUsers(
	ctx context.Context,
) ([]vibe.AdminUser, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ListAdminUsers")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.ListAdminUsersStatement.QueryContext(cctx)
	if err != nil {
		return nil, fmt.Errorf(
			"error querying admin users in ListAdminUsers: %w",
			err,
		)
	}
	defer rows.Close()

	users := make([]vibe.AdminUser, 0)
	for rows.Next() {
		var rowData adminUserRow
		err = rowData.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"error scanning admin users in ListAdminUsers: %w",
				err,
			)
		}

		admin := rowData.toAdminUser()
		users = append(users, admin)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf(
			"error iterating admin users in ListAdminUsers: %w",
			err,
		)
	}

	return users, nil
}

func (c *Client) prepareCreateAdminUserStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO admin_users (
			id,
			username,
			password_hash
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			username,
			password_hash,
			session_version,
			created_at,
			updated_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateAdminUserStatement: %w", err)
	}

	c.CreateAdminUserStatement = stmt

	return nil
}

func (c *Client) CreateAdminUser(
	ctx context.Context,
	user vibe.AdminUser,
) (*vibe.AdminUser, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateAdminUser")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.CreateAdminUserStatement.QueryRowContext(
		cctx,
		user.ID,
		user.Username,
		user.PasswordHash,
	)
	var rowData adminUserRow
	err := rowData.scanRow(row)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.ConstraintName == adminUsernameConstraint {
			return nil, internalerror.ErrAdminUsernameUnavailable{
				Err: fmt.Errorf(
					"error creating admin user in CreateAdminUser: %w",
					err,
				),
			}
		}

		return nil, fmt.Errorf(
			"error scanning admin user in CreateAdminUser: %w",
			err,
		)
	}

	admin := rowData.toAdminUser()

	return &admin, nil
}

func (c *Client) prepareUpdateAdminUserPasswordStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE admin_users
		SET
			password_hash = $2,
			session_version = session_version + 1,
			updated_at = NOW()
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf(
			"error preparing UpdateAdminUserPasswordStatement: %w",
			err,
		)
	}

	c.UpdateAdminUserPasswordStatement = stmt

	return nil
}

func (c *Client) UpdateAdminUserPassword(
	ctx context.Context,
	adminID string,
	passwordHash string,
) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdateAdminUserPassword")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.UpdateAdminUserPasswordStatement.ExecContext(
		cctx,
		adminID,
		passwordHash,
	)
	if err != nil {
		return false, fmt.Errorf(
			"error updating admin password in UpdateAdminUserPassword: %w",
			err,
		)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf(
			"error getting affected rows in UpdateAdminUserPassword: %w",
			err,
		)
	}

	return affected > 0, nil
}

func (c *Client) prepareDeleteAdminUserStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM admin_users
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeleteAdminUserStatement: %w", err)
	}

	c.DeleteAdminUserStatement = stmt

	return nil
}

func (c *Client) DeleteAdminUser(
	ctx context.Context,
	adminID string,
) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "DeleteAdminUser")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.DeleteAdminUserStatement.ExecContext(cctx, adminID)
	if err != nil {
		return false, fmt.Errorf(
			"error deleting admin user in DeleteAdminUser: %w",
			err,
		)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf(
			"error getting affected rows in DeleteAdminUser: %w",
			err,
		)
	}

	return affected > 0, nil
}

const adminUsernameConstraint = "admin_users_username_key"
