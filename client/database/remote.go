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

func (c *Client) prepareCreateRemoteControlV2Stmt() error {
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
			current_playlist_item_id = '',
			playback_position_ms = 0,
			playback_is_playing = FALSE,
			playback_observed_at = NOW(),
			pairing_expires_at = EXCLUDED.pairing_expires_at,
			last_seen_at = NOW(),
			updated_at = NOW()
		RETURNING id, owner_user_id, current_room_id, current_playlist_item_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateRemoteControlV2Statement: %w", err)
	}

	c.CreateRemoteControlV2Statement = stmt

	return nil
}

func (c *Client) CreateRemoteControlV2(ctx context.Context, remoteID, ownerUserID, pairingTokenHash, pairingCodeHash, roomID string, pairingExpiresAt time.Time) (*vibe.RemoteControlV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateRemoteControlV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.CreateRemoteControlV2Statement.QueryRowContext(
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

	remote, err := row.toRemoteControlV2()
	if err != nil {
		return nil, fmt.Errorf("error mapping remote: %w", err)
	}

	return remote, nil
}

func (c *Client) prepareGetRemoteControlByOwnerV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, owner_user_id, current_room_id, current_playlist_item_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
		FROM remote_controls
		WHERE owner_user_id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRemoteControlByOwnerV2Statement: %w", err)
	}

	c.GetRemoteControlByOwnerV2Statement = stmt

	return nil
}

func (c *Client) GetRemoteControlByOwnerV2(ctx context.Context, ownerUserID string) (*vibe.RemoteControlV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetRemoteControlByOwnerV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetRemoteControlByOwnerV2Statement.QueryRowContext(cctx, ownerUserID)

	var row remoteControlRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RemoteControlV2{}, nil
		}

		return nil, fmt.Errorf("error getting remote control by owner: %w", err)
	}

	remote, err := row.toRemoteControlV2()
	if err != nil {
		return nil, fmt.Errorf("error mapping remote: %w", err)
	}

	return remote, nil
}

func (c *Client) prepareGetRemoteControlV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, owner_user_id, current_room_id, current_playlist_item_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
		FROM remote_controls
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetRemoteControlV2Statement: %w", err)
	}

	c.GetRemoteControlV2Statement = stmt

	return nil
}

func (c *Client) GetRemoteControlV2(ctx context.Context, remoteID string) (*vibe.RemoteControlV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetRemoteControlV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetRemoteControlV2Statement.QueryRowContext(cctx, remoteID)

	var row remoteControlRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RemoteControlV2{}, nil
		}

		return nil, fmt.Errorf("error getting remote control: %w", err)
	}

	remote, err := row.toRemoteControlV2()
	if err != nil {
		return nil, fmt.Errorf("error mapping remote: %w", err)
	}

	return remote, nil
}

func (c *Client) preparePairRemoteControlV2Stmt() error {
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
		RETURNING id, owner_user_id, current_room_id, current_playlist_item_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing PairRemoteControlV2Statement: %w", err)
	}

	c.PairRemoteControlV2Statement = stmt

	return nil
}

func (c *Client) PairRemoteControlV2(ctx context.Context, remoteID, pairingTokenHash, pairingCodeHash, controllerTokenHash string) (*vibe.RemoteControlV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "PairRemoteControlV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.PairRemoteControlV2Statement.QueryRowContext(
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
			return &vibe.RemoteControlV2{}, nil
		}

		return nil, fmt.Errorf("error pairing remote control: %w", err)
	}

	remote, err := row.toRemoteControlV2()
	if err != nil {
		return nil, fmt.Errorf("error mapping remote: %w", err)
	}

	return remote, nil
}

func (c *Client) prepareAuthenticateRemoteControlV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, owner_user_id, current_room_id, current_playlist_item_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
		FROM remote_controls
		WHERE id = $1
		AND controller_token_hash = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing AuthenticateRemoteControlV2Statement: %w", err)
	}

	c.AuthenticateRemoteControlV2Statement = stmt

	return nil
}

func (c *Client) AuthenticateRemoteControlV2(ctx context.Context, remoteID, controllerTokenHash string) (*vibe.RemoteControlV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AuthenticateRemoteControlV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.AuthenticateRemoteControlV2Statement.QueryRowContext(
		cctx,
		remoteID,
		controllerTokenHash,
	)

	var row remoteControlRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RemoteControlV2{}, nil
		}

		return nil, fmt.Errorf("error authenticating remote control: %w", err)
	}

	remote, err := row.toRemoteControlV2()
	if err != nil {
		return nil, fmt.Errorf("error mapping remote: %w", err)
	}

	return remote, nil
}

func (c *Client) prepareUpdateOwnedRemoteControlV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE remote_controls
		SET current_room_id = $3,
			current_playlist_item_id = $4,
			playback_position_ms = $5,
			playback_is_playing = $6,
			playback_observed_at = NOW(),
			last_seen_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND owner_user_id = $2
		RETURNING id, owner_user_id, current_room_id, current_playlist_item_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdateOwnedRemoteControlV2Statement: %w", err)
	}

	c.UpdateOwnedRemoteControlV2Statement = stmt

	return nil
}

func (c *Client) UpdateOwnedRemoteControlV2(ctx context.Context, remoteID, ownerUserID string, request vibe.RemoteUpdateRequestV2) (*vibe.RemoteControlV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdateOwnedRemoteControlV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.UpdateOwnedRemoteControlV2Statement.QueryRowContext(
		cctx,
		remoteID,
		ownerUserID,
		request.RoomID,
		request.CurrentPlaylistItemID,
		request.PlaybackPositionMs,
		request.PlaybackIsPlaying,
	)

	var row remoteControlRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RemoteControlV2{}, nil
		}

		return nil, fmt.Errorf("error updating owned remote control: %w", err)
	}

	remote, err := row.toRemoteControlV2()
	if err != nil {
		return nil, fmt.Errorf("error mapping remote: %w", err)
	}

	return remote, nil
}

func (c *Client) prepareUpdatePairedRemoteControlV2Stmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE remote_controls
		SET current_room_id = CASE
				WHEN $2 = '' THEN current_room_id
				ELSE $2
			END,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, owner_user_id, current_room_id, current_playlist_item_id,
			playback_position_ms, playback_is_playing, playback_observed_at,
			controller_token_hash != '', pairing_expires_at, last_seen_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdatePairedRemoteControlV2Statement: %w", err)
	}

	c.UpdatePairedRemoteControlV2Statement = stmt

	return nil
}

func (c *Client) UpdatePairedRemoteControlV2(ctx context.Context, remoteID string, request vibe.RemoteUpdateRequestV2) (*vibe.RemoteControlV2, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdatePairedRemoteControlV2")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.UpdatePairedRemoteControlV2Statement.QueryRowContext(
		cctx,
		remoteID,
		request.RoomID,
	)

	var row remoteControlRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.RemoteControlV2{}, nil
		}

		return nil, fmt.Errorf("error updating paired remote control: %w", err)
	}

	remote, err := row.toRemoteControlV2()
	if err != nil {
		return nil, fmt.Errorf("error mapping remote: %w", err)
	}

	return remote, nil
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
	ID                    sql.NullString
	OwnerUserID           sql.NullString
	CurrentRoomID         sql.NullString
	CurrentPlaylistItemID sql.NullString
	PlaybackPositionMs    sql.NullInt64
	PlaybackIsPlaying     sql.NullBool
	PlaybackObservedAt    sql.NullTime
	Paired                sql.NullBool
	PairingExpiresAt      sql.NullTime
	LastSeenAt            sql.NullTime
}

func (r *remoteControlRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.ID,
		&r.OwnerUserID,
		&r.CurrentRoomID,
		&r.CurrentPlaylistItemID,
		&r.PlaybackPositionMs,
		&r.PlaybackIsPlaying,
		&r.PlaybackObservedAt,
		&r.Paired,
		&r.PairingExpiresAt,
		&r.LastSeenAt,
	)
	if err != nil {
		return fmt.Errorf("error scanning remote control row: %w", err)
	}

	return nil
}

func (r *remoteControlRow) toRemoteControlV2() (*vibe.RemoteControlV2, error) {
	return &vibe.RemoteControlV2{
		ID:                    r.ID.String,
		OwnerUserID:           r.OwnerUserID.String,
		CurrentRoomID:         r.CurrentRoomID.String,
		CurrentPlaylistItemID: r.CurrentPlaylistItemID.String,
		PlaybackPositionMs:    int(r.PlaybackPositionMs.Int64),
		PlaybackIsPlaying:     r.PlaybackIsPlaying.Bool,
		PlaybackObservedAt:    r.PlaybackObservedAt.Time,
		Paired:                r.Paired.Bool,
		PairingExpiresAt:      r.PairingExpiresAt.Time,
		LastSeenAt:            r.LastSeenAt.Time,
	}, nil
}
