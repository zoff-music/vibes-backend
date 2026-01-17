package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"
)

// prepareGetRoomStmt prepares the GetRoomStatement.
func (c *Client) prepareGetRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, name, admin_password_hash, settings_json, created_at
		FROM rooms
		WHERE id = ?
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRoomStatement: %w", err)
	}

	c.GetRoomStatement = stmt

	return nil
}

// GetRoom fetches a room by ID.
func (c *Client) GetRoom(ctx context.Context, id string) (*vibe.Room, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetRoom")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var row roomRow

	err := c.GetRoomStatement.QueryRowContext(cctx, id).Scan(
		&row.ID,
		&row.Name,
		&row.AdminPasswordHash,
		&row.SettingsJSON,
		&row.CreatedAt,
	)
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

// prepareCreateRoomStmt prepares the CreateRoomStatement.
func (c *Client) prepareCreateRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO rooms (id, name, admin_password_hash, settings_json, created_at)
		VALUES (?, ?, ?, ?, ?)
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

	settingsJSON, err := json.Marshal(room.Settings)
	if err != nil {
		return nil, fmt.Errorf("error marshaling room settings: %w", err)
	}

	_, err = c.CreateRoomStatement.ExecContext(cctx,
		room.ID,
		room.Name,
		room.AdminPasswordHash,
		string(settingsJSON),
		room.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating room: %w", err)
	}

	return room, nil
}

// prepareUpdateRoomStmt prepares the UpdateRoomStatement.
func (c *Client) prepareUpdateRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE rooms
		SET name = ?, admin_password_hash = ?, settings_json = ?
		WHERE id = ?
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

	settingsJSON, err := json.Marshal(room.Settings)
	if err != nil {
		return nil, fmt.Errorf("error marshaling room settings: %w", err)
	}

	_, err = c.UpdateRoomStatement.ExecContext(cctx,
		room.Name,
		room.AdminPasswordHash,
		string(settingsJSON),
		room.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("error updating room: %w", err)
	}

	return room, nil
}

// --- Internal types and helpers ---

type roomRow struct {
	ID                string
	Name              string
	AdminPasswordHash sql.NullString
	SettingsJSON      string
	CreatedAt         time.Time
}

func (r *roomRow) toRoom() (*vibe.Room, error) {
	var settings vibe.RoomSettings

	err := json.Unmarshal([]byte(r.SettingsJSON), &settings)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling room settings: %w", err)
	}

	return &vibe.Room{
		ID:                r.ID,
		Name:              r.Name,
		AdminPasswordHash: r.AdminPasswordHash.String,
		HasPassword:       r.AdminPasswordHash.Valid && r.AdminPasswordHash.String != "",
		Settings:          settings,
		CreatedAt:         r.CreatedAt,
	}, nil
}
