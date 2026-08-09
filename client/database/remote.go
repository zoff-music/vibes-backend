package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareCreateRemoteControlStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO remote_controls (
			id,
			owner_user_id,
			pairing_token_hash,
			pairing_code_hash,
			controller_token_hash,
			current_room_id,
			pairing_expires_at,
			last_seen_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, '', $5, $6, NOW(), NOW())
		ON CONFLICT (owner_user_id) DO UPDATE SET
			id = EXCLUDED.id,
			pairing_token_hash = EXCLUDED.pairing_token_hash,
			pairing_code_hash = EXCLUDED.pairing_code_hash,
			controller_token_hash = '',
			current_room_id = EXCLUDED.current_room_id,
			current_song_id = '',
			playback_position_ms = 0,
			playback_is_playing = FALSE,
			playback_observed_at = NOW(),
			pairing_expires_at = EXCLUDED.pairing_expires_at,
			last_seen_at = NOW(),
			updated_at = NOW()
		RETURNING id, owner_user_id, current_room_id, current_song_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateRemoteControlStatement: %w", err)
	}

	c.CreateRemoteControlStatement = stmt

	return nil
}

func (c *Client) CreateRemoteControl(ctx context.Context, remoteID, ownerUserID, pairingTokenHash, pairingCodeHash, roomID string, pairingExpiresAt time.Time) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateRemoteControl")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.CreateRemoteControlStatement.QueryRowContext(
		cctx,
		remoteID,
		ownerUserID,
		pairingTokenHash,
		pairingCodeHash,
		roomID,
		pairingExpiresAt,
	)

	var row remoteControlRow
	err := row.scan(r)
	if err != nil {
		return nil, fmt.Errorf("error creating remote control: %w", err)
	}

	return row.toRemoteControl(), nil
}

func (c *Client) prepareGetRemoteControlByOwnerStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, owner_user_id, current_room_id, current_song_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
		FROM remote_controls
		WHERE owner_user_id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRemoteControlByOwnerStatement: %w", err)
	}

	c.GetRemoteControlByOwnerStatement = stmt

	return nil
}

func (c *Client) GetRemoteControlByOwner(ctx context.Context, ownerUserID string) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetRemoteControlByOwner")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetRemoteControlByOwnerStatement.QueryRowContext(cctx, ownerUserID)

	var row remoteControlRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RemoteControl{}, nil
		}

		return nil, fmt.Errorf("error getting remote control by owner: %w", err)
	}

	return row.toRemoteControl(), nil
}

func (c *Client) prepareGetRemoteControlStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, owner_user_id, current_room_id, current_song_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
		FROM remote_controls
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRemoteControlStatement: %w", err)
	}

	c.GetRemoteControlStatement = stmt

	return nil
}

func (c *Client) GetRemoteControl(ctx context.Context, remoteID string) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetRemoteControl")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetRemoteControlStatement.QueryRowContext(cctx, remoteID)

	var row remoteControlRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RemoteControl{}, nil
		}

		return nil, fmt.Errorf("error getting remote control: %w", err)
	}

	return row.toRemoteControl(), nil
}

func (c *Client) preparePairRemoteControlStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE remote_controls
		SET controller_token_hash = $4,
			pairing_token_hash = '',
			pairing_code_hash = '',
			updated_at = NOW()
		WHERE id = $1
		AND pairing_expires_at > NOW()
		AND (
			($2 != '' AND pairing_token_hash = $2)
			OR ($3 != '' AND pairing_code_hash = $3)
		)
		RETURNING id, owner_user_id, current_room_id, current_song_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing PairRemoteControlStatement: %w", err)
	}

	c.PairRemoteControlStatement = stmt

	return nil
}

func (c *Client) PairRemoteControl(ctx context.Context, remoteID, pairingTokenHash, pairingCodeHash, controllerTokenHash string) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "PairRemoteControl")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.PairRemoteControlStatement.QueryRowContext(
		cctx,
		remoteID,
		pairingTokenHash,
		pairingCodeHash,
		controllerTokenHash,
	)

	var row remoteControlRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RemoteControl{}, nil
		}

		return nil, fmt.Errorf("error pairing remote control: %w", err)
	}

	return row.toRemoteControl(), nil
}

func (c *Client) prepareAuthenticateRemoteControlStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, owner_user_id, current_room_id, current_song_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
		FROM remote_controls
		WHERE id = $1
		AND controller_token_hash = $2
		AND last_seen_at > NOW() - ($3 * INTERVAL '1 millisecond')
	`)
	if err != nil {
		return fmt.Errorf("error preparing AuthenticateRemoteControlStatement: %w", err)
	}

	c.AuthenticateRemoteControlStatement = stmt

	return nil
}

func (c *Client) AuthenticateRemoteControl(ctx context.Context, remoteID, controllerTokenHash string, presenceTimeout time.Duration) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AuthenticateRemoteControl")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.AuthenticateRemoteControlStatement.QueryRowContext(
		cctx,
		remoteID,
		controllerTokenHash,
		presenceTimeout.Milliseconds(),
	)

	var row remoteControlRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RemoteControl{}, nil
		}

		return nil, fmt.Errorf("error authenticating remote control: %w", err)
	}

	return row.toRemoteControl(), nil
}

func (c *Client) prepareUpdateOwnedRemoteControlStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE remote_controls
		SET current_room_id = $3,
			current_song_id = $4,
			playback_position_ms = $5,
			playback_is_playing = $6,
			playback_observed_at = NOW(),
			last_seen_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND owner_user_id = $2
		RETURNING id, owner_user_id, current_room_id, current_song_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdateOwnedRemoteControlStatement: %w", err)
	}

	c.UpdateOwnedRemoteControlStatement = stmt

	return nil
}

func (c *Client) UpdateOwnedRemoteControl(ctx context.Context, remoteID, ownerUserID string, request vibe.RemoteUpdateRequest) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdateOwnedRemoteControl")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.UpdateOwnedRemoteControlStatement.QueryRowContext(
		cctx,
		remoteID,
		ownerUserID,
		request.RoomID,
		request.CurrentSongID,
		request.PlaybackPositionMs,
		request.PlaybackIsPlaying,
	)

	var row remoteControlRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RemoteControl{}, nil
		}

		return nil, fmt.Errorf("error updating owned remote control: %w", err)
	}

	return row.toRemoteControl(), nil
}

func (c *Client) prepareUpdatePairedRemoteControlStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE remote_controls
		SET current_room_id = CASE
				WHEN $2 = '' THEN current_room_id
				ELSE $2
			END,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, owner_user_id, current_room_id, current_song_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdatePairedRemoteControlStatement: %w", err)
	}

	c.UpdatePairedRemoteControlStatement = stmt

	return nil
}

func (c *Client) UpdatePairedRemoteControl(ctx context.Context, remoteID string, request vibe.RemoteUpdateRequest) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdatePairedRemoteControl")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.UpdatePairedRemoteControlStatement.QueryRowContext(
		cctx,
		remoteID,
		request.RoomID,
	)

	var row remoteControlRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RemoteControl{}, nil
		}

		return nil, fmt.Errorf("error updating paired remote control: %w", err)
	}

	return row.toRemoteControl(), nil
}

func (c *Client) prepareDeleteRemoteControlStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM remote_controls
		WHERE id = $1 AND owner_user_id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing DeleteRemoteControlStatement: %w", err)
	}

	c.DeleteRemoteControlStatement = stmt

	return nil
}

func (c *Client) DeleteRemoteControl(ctx context.Context, remoteID, ownerUserID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "DeleteRemoteControl")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.DeleteRemoteControlStatement.ExecContext(cctx, remoteID, ownerUserID)
	if err != nil {
		return fmt.Errorf("error deleting remote control: %w", err)
	}

	return nil
}

type remoteControlRow struct {
	ID                 sql.NullString
	OwnerUserID        sql.NullString
	CurrentRoomID      sql.NullString
	CurrentSongID      sql.NullString
	PlaybackPositionMs sql.NullInt64
	PlaybackIsPlaying  sql.NullBool
	PlaybackObservedAt sql.NullTime
	Paired             sql.NullBool
	PairingExpiresAt   sql.NullTime
	LastSeenAt         sql.NullTime
}

func (r *remoteControlRow) scan(row *sql.Row) error {
	return row.Scan(
		&r.ID,
		&r.OwnerUserID,
		&r.CurrentRoomID,
		&r.CurrentSongID,
		&r.PlaybackPositionMs,
		&r.PlaybackIsPlaying,
		&r.PlaybackObservedAt,
		&r.Paired,
		&r.PairingExpiresAt,
		&r.LastSeenAt,
	)
}

func (r *remoteControlRow) toRemoteControl() *vibe.RemoteControl {
	return &vibe.RemoteControl{
		ID:                 r.ID.String,
		OwnerUserID:        r.OwnerUserID.String,
		CurrentRoomID:      r.CurrentRoomID.String,
		CurrentSongID:      r.CurrentSongID.String,
		PlaybackPositionMs: r.PlaybackPositionMs.Int64,
		PlaybackIsPlaying:  r.PlaybackIsPlaying.Bool,
		PlaybackObservedAt: r.PlaybackObservedAt.Time,
		Paired:             r.Paired.Bool,
		PairingExpiresAt:   r.PairingExpiresAt.Time,
		LastSeenAt:         r.LastSeenAt.Time,
	}
}
