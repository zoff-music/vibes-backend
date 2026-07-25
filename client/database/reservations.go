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

	reservation, err := scanRoomNameReservation(row)
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

	reservation, err := scanRoomNameReservation(row)
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

	return reservation, nil
}

func scanRoomNameReservation(row *sql.Row) (*vibe.RoomNameReservation, error) {
	var reservation vibe.RoomNameReservation

	err := row.Scan(
		&reservation.Name,
		&reservation.Token,
		&reservation.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error scanning room name reservation row: %w", err)
	}

	return &reservation, nil
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
) (int64, error) {
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

	return deleted, nil
}
