package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/zoff-music/vibes-backend/internalerror"
	"log"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

// prepareGetPlaylistItemsStmt prepares the GetPlaylistItemsStatement.
func (c *Client) prepareGetPlaylistItemsStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			a.id,
			a.room_id,
			a.source_type,
			a.source_id,
			a.provider_url,
			a.playback_restriction,
			a.title,
			a.publisher,
			a.thumbnail_url,
			a.duration,
			a.added_by,
			COALESCE(
				(SELECT c.name FROM sessions c WHERE c.id = a.added_by),
				a.added_by_nickname
			) AS added_by_nickname,
			a.added_at,
			COUNT(b.user_id) as vote_count
		FROM playlist_items a
		LEFT JOIN playlist_item_votes b
		ON a.id = b.playlist_item_id
		AND a.room_id = b.room_id
		WHERE a.room_id = $1
		AND a.source_type = ANY($2::text[])
		AND NOT (a.source_type = 'youtube' AND a.duration <= 0)
		GROUP BY a.id, a.room_id, a.source_type, a.source_id, a.provider_url, a.playback_restriction, a.title, a.publisher, a.thumbnail_url, a.duration, a.added_by, a.added_by_nickname, a.added_at
		ORDER BY vote_count DESC, MAX(b.created_at) ASC, a.added_at ASC
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetPlaylistItemsStatement: %w", err)
	}

	c.GetPlaylistItemsStatement = stmt

	return nil
}

// GetPlaylistItems fetches all playlistItems in a room's queue.
func (c *Client) GetPlaylistItems(ctx context.Context, roomID string) ([]vibe.PlaylistItem, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetPlaylistItems")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.GetPlaylistItemsStatement.QueryContext(
		cctx,
		roomID,
		c.enabledProviders,
	)
	if err != nil {
		return nil, fmt.Errorf("error fetching playlistItems: %w", err)
	}
	defer rows.Close()

	playlistItems := []vibe.PlaylistItem{}
	for rows.Next() {
		var row playlistItemRow

		err := row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning playlistItem row: %w", err)
		}

		playlistItem, err := row.toPlaylistItem()
		if err != nil {
			return nil, fmt.Errorf("error mapping playlistItem: %w", err)
		}

		playlistItems = append(playlistItems, *playlistItem)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating playlistItem rows: %w", err)
	}

	return playlistItems, nil
}

type playlistItemRow struct {
	ID                  sql.NullString
	RoomID              sql.NullString
	SourceType          sql.NullString
	SourceID            sql.NullString
	ProviderURL         sql.NullString
	PlaybackRestriction sql.NullString
	Title               sql.NullString
	Publisher           sql.NullString
	ThumbnailURL        sql.NullString
	Duration            sql.NullInt64
	AddedBySessionID    sql.NullString
	AddedBy             sql.NullString
	AddedAt             sql.NullTime
	VoteCount           sql.NullInt64
}

func (r *playlistItemRow) scanRows(rows *sql.Rows) error {
	err := rows.Scan(
		&r.ID,
		&r.RoomID,
		&r.SourceType,
		&r.SourceID,
		&r.ProviderURL,
		&r.PlaybackRestriction,
		&r.Title,
		&r.Publisher,
		&r.ThumbnailURL,
		&r.Duration,
		&r.AddedBySessionID,
		&r.AddedBy,
		&r.AddedAt,
		&r.VoteCount,
	)
	if err != nil {
		return fmt.Errorf("error scanning playlistItem row: %w", err)
	}

	return nil
}

func (r *playlistItemRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.ID,
		&r.RoomID,
		&r.SourceType,
		&r.SourceID,
		&r.ProviderURL,
		&r.PlaybackRestriction,
		&r.Title,
		&r.Publisher,
		&r.ThumbnailURL,
		&r.Duration,
		&r.AddedBySessionID,
		&r.AddedBy,
		&r.AddedAt,
		&r.VoteCount,
	)
	if err != nil {
		return fmt.Errorf("error scanning playlistItem row: %w", err)
	}

	return nil
}

func (r *playlistItemRow) toPlaylistItem() (*vibe.PlaylistItem, error) {
	return &vibe.PlaylistItem{
		ID:                  r.ID.String,
		RoomID:              r.RoomID.String,
		SourceType:          r.SourceType.String,
		SourceID:            r.SourceID.String,
		ProviderURL:         r.ProviderURL.String,
		PlaybackRestriction: r.PlaybackRestriction.String,
		Title:               r.Title.String,
		Publisher:           r.Publisher.String,
		ThumbnailURL:        r.ThumbnailURL.String,
		Duration:            int(r.Duration.Int64),
		AddedBySessionID:    r.AddedBySessionID.String,
		AddedBy:             r.AddedBy.String,
		AddedAt:             r.AddedAt.Time,
		VoteCount:           int(r.VoteCount.Int64),
	}, nil
}

// prepareGetPlaylistItemStmt prepares the GetPlaylistItemStatement.
func (c *Client) prepareGetPlaylistItemStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			a.id,
			a.room_id,
			a.source_type,
			a.source_id,
			a.provider_url,
			a.playback_restriction,
			a.title,
			a.publisher,
			a.thumbnail_url,
			a.duration,
			a.added_by,
			COALESCE(
				(SELECT c.name FROM sessions c WHERE c.id = a.added_by),
				a.added_by_nickname
			) AS added_by_nickname,
			a.added_at,
			COUNT(b.user_id) as vote_count
		FROM playlist_items a
		LEFT JOIN playlist_item_votes b
		ON a.id = b.playlist_item_id
		AND a.room_id = b.room_id
		WHERE a.room_id = $1
		AND a.id = $2
		AND a.source_type = ANY($3::text[])
		AND NOT (a.source_type = 'youtube' AND a.duration <= 0)
		GROUP BY a.id, a.room_id, a.source_type, a.source_id, a.provider_url, a.playback_restriction, a.title, a.publisher, a.thumbnail_url, a.duration, a.added_by, a.added_by_nickname, a.added_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetPlaylistItemStatement: %w", err)
	}

	c.GetPlaylistItemStatement = stmt

	return nil
}

// GetPlaylistItem fetches a single playlistItem by ID.
func (c *Client) GetPlaylistItem(ctx context.Context, roomID, playlistItemID string) (*vibe.PlaylistItem, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetPlaylistItem")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetPlaylistItemStatement.QueryRowContext(
		cctx,
		roomID,
		playlistItemID,
		c.enabledProviders,
	)

	var row playlistItemRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.PlaylistItem{}, nil
		}

		return nil, fmt.Errorf("error fetching playlistItem: %w", err)
	}

	playlistItem, err := row.toPlaylistItem()
	if err != nil {
		return nil, fmt.Errorf("error mapping playlistItem: %w", err)
	}

	return playlistItem, nil
}

func (c *Client) prepareAddPlaylistItemStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH room_config_q AS (
			SELECT
				a.id AS room_id,
				COALESCE(b.allow_duplicates, FALSE) AS allow_duplicates
			FROM rooms a
			LEFT JOIN room_settings b ON b.room_id = a.id
			WHERE a.id = $1
			AND $2 = ANY($12::text[])
			FOR KEY SHARE OF a
		),
		existing_session_q AS MATERIALIZED (
			SELECT name
			FROM sessions
			WHERE id = $8
		),
		generated_range_q AS MATERIALIZED (
			SELECT id AS max_id
			FROM room_name_pool
			WHERE generated
			AND NOT EXISTS (SELECT 1 FROM existing_session_q)
			ORDER BY id DESC
			LIMIT 1
		),
		target_q AS MATERIALIZED (
			SELECT
				MOD(
					hashtextextended($8, 0) & 9223372036854775807,
					max_id
				) + 1 AS id
			FROM generated_range_q
		),
		generated_name_q AS MATERIALIZED (
			SELECT COALESCE(
				(
					SELECT name
					FROM room_name_pool
					WHERE generated
					AND id >= (SELECT id FROM target_q)
					ORDER BY id
					LIMIT 1
				),
				(
					SELECT name
					FROM room_name_pool
					WHERE generated
					AND EXISTS (SELECT 1 FROM target_q)
					ORDER BY id
					LIMIT 1
				)
			) AS name
		),
		inserted_session_q AS (
			INSERT INTO sessions (id, name)
			SELECT
				$8,
				SPLIT_PART(a.name, '-', 1) || '-' || SPLIT_PART(a.name, '-', 2)
			FROM generated_name_q a
			WHERE a.name IS NOT NULL
			ON CONFLICT (id) DO UPDATE
			SET name = sessions.name
			RETURNING name
		),
		session_profile_q AS (
			SELECT name
			FROM existing_session_q
			UNION ALL
			SELECT name
			FROM inserted_session_q
			LIMIT 1
		),
		upserted_playlist_item_q AS (
			INSERT INTO playlist_items (
				id,
				room_id,
				source_type,
				source_id,
				provider_url,
				playback_restriction,
				title,
				publisher,
				thumbnail_url,
				duration,
				added_by,
				added_by_nickname,
				added_at,
				duplicate_guard
			)
			SELECT
				$10,
				a.room_id,
				$2,
				$3,
				$13,
				$14,
				$4,
				$5,
				$6,
				$7,
				$8,
				COALESCE((SELECT name FROM session_profile_q), $9),
				NOW(),
				NOT a.allow_duplicates
			FROM room_config_q a
			ON CONFLICT (room_id, source_type, source_id)
			WHERE duplicate_guard
			DO UPDATE SET source_id = EXCLUDED.source_id
			RETURNING
				id,
				room_id,
				source_type,
				source_id,
				provider_url,
				playback_restriction,
				title,
				publisher,
				thumbnail_url,
				duration,
				added_by,
				added_by_nickname,
				added_at,
				id = $10 AS inserted
		),
		inserted_vote_q AS (
			INSERT INTO playlist_item_votes (room_id, playlist_item_id, user_id)
			SELECT a.room_id, a.id, $8
			FROM upserted_playlist_item_q a
			WHERE $11
			ON CONFLICT (room_id, playlist_item_id, user_id) DO NOTHING
			RETURNING 1
		)
		SELECT
			CASE
				WHEN a.inserted THEN 'added'
				WHEN EXISTS (SELECT 1 FROM inserted_vote_q) THEN 'duplicate_voted'
				ELSE 'duplicate_already_voted'
			END AS result,
			a.id,
			a.room_id,
			a.source_type,
			a.source_id,
			a.provider_url,
			a.playback_restriction,
			a.title,
			a.publisher,
			a.thumbnail_url,
			a.duration,
			a.added_by,
			COALESCE(
				(SELECT b.name FROM sessions b WHERE b.id = a.added_by),
				a.added_by_nickname
			) AS added_by_nickname,
			a.added_at,
			(
				SELECT COUNT(*)
				FROM playlist_item_votes b
				WHERE b.room_id = a.room_id
				AND b.playlist_item_id = a.id
			) + (SELECT COUNT(*) FROM inserted_vote_q) AS vote_count
		FROM upserted_playlist_item_q a
		UNION ALL
		SELECT
			CASE
				WHEN $2 = ANY($12::text[]) THEN 'room_not_found'
				ELSE 'provider_disabled'
			END AS result,
			NULL::TEXT AS id,
			NULL::TEXT AS room_id,
			NULL::TEXT AS source_type,
			NULL::TEXT AS source_id,
			NULL::TEXT AS provider_url,
			NULL::TEXT AS playback_restriction,
			NULL::TEXT AS title,
			NULL::TEXT AS publisher,
			NULL::TEXT AS thumbnail_url,
			NULL::INTEGER AS duration,
			NULL::TEXT AS added_by,
			NULL::TEXT AS added_by_nickname,
			NULL::TIMESTAMP AS added_at,
			0::BIGINT AS vote_count
		WHERE NOT EXISTS (SELECT 1 FROM upserted_playlist_item_q)
	`)
	if err != nil {
		return fmt.Errorf("error preparing AddPlaylistItemStatement: %w", err)
	}

	c.AddPlaylistItemStatement = stmt

	return nil
}

// AddPlaylistItem adds a playlistItem to the queue.
func (c *Client) AddPlaylistItem(ctx context.Context, playlistItem *vibe.PlaylistItem) (*vibe.AddPlaylistItemResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AddPlaylistItem")
	defer span.End()

	result, err := c.addPlaylistItem(ctx, playlistItem, true)
	if err != nil {
		return nil, fmt.Errorf("error adding playlistItem in AddPlaylistItem: %w", err)
	}

	return result, nil
}

func (c *Client) AddImportedPlaylistItem(
	ctx context.Context,
	playlistItem *vibe.PlaylistItem,
) (*vibe.AddPlaylistItemResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AddImportedPlaylistItem")
	defer span.End()

	result, err := c.addPlaylistItem(ctx, playlistItem, false)
	if err != nil {
		return nil, fmt.Errorf("error adding imported playlist item: %w", err)
	}

	return result, nil
}

func (c *Client) addPlaylistItem(
	ctx context.Context,
	playlistItem *vibe.PlaylistItem,
	addVote bool,
) (*vibe.AddPlaylistItemResult, error) {
	if vibe.IsLiveVideo(playlistItem.SourceType, playlistItem.Duration) {
		return nil, internalerror.ErrLiveVideo{
			Err: fmt.Errorf("error adding playlistItem in addPlaylistItem: live videos are not supported"),
		}
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.AddPlaylistItemStatement.QueryRowContext(cctx,
		playlistItem.RoomID,
		playlistItem.SourceType,
		playlistItem.SourceID,
		playlistItem.Title,
		playlistItem.Publisher,
		playlistItem.ThumbnailURL,
		playlistItem.Duration,
		playlistItem.AddedBySessionID,
		playlistItem.AddedBy,
		playlistItem.ID,
		addVote,
		c.enabledProviders,
		playlistItem.ProviderURL,
		playlistItem.PlaybackRestriction,
	)

	var row addPlaylistItemRow
	err := row.scan(r)
	if err != nil {
		return nil, fmt.Errorf("error adding playlistItem atomically in addPlaylistItem: %w", err)
	}

	if row.Result.String == addPlaylistItemResultRoomNotFound {
		return nil, fmt.Errorf("error adding playlistItem in addPlaylistItem: room %s not found", playlistItem.RoomID)
	}
	if row.Result.String == addPlaylistItemResultProviderDisabled {
		return nil, fmt.Errorf("error adding playlistItem in addPlaylistItem: provider %s is disabled", playlistItem.SourceType)
	}

	outcome := row.Result.String
	if outcome != vibe.AddPlaylistItemOutcomeAdded &&
		outcome != vibe.AddPlaylistItemOutcomeDuplicateVoted &&
		outcome != vibe.AddPlaylistItemOutcomeDuplicateAlreadyVoted {
		return nil, fmt.Errorf("error adding playlistItem in addPlaylistItem: unknown outcome %s", outcome)
	}

	convertedPlaylistItem, err := row.toPlaylistItem()
	if err != nil {
		return nil, fmt.Errorf("error mapping added playlistItem: %w", err)
	}

	return &vibe.AddPlaylistItemResult{
		PlaylistItem: *convertedPlaylistItem,
		Outcome:      outcome,
	}, nil
}

type addPlaylistItemRow struct {
	Result              sql.NullString
	ID                  sql.NullString
	RoomID              sql.NullString
	SourceType          sql.NullString
	SourceID            sql.NullString
	ProviderURL         sql.NullString
	PlaybackRestriction sql.NullString
	Title               sql.NullString
	Publisher           sql.NullString
	ThumbnailURL        sql.NullString
	Duration            sql.NullInt64
	AddedBySessionID    sql.NullString
	AddedBy             sql.NullString
	AddedAt             sql.NullTime
	VoteCount           sql.NullInt64
}

func (r *addPlaylistItemRow) scan(row *sql.Row) error {
	err := row.Scan(
		&r.Result,
		&r.ID,
		&r.RoomID,
		&r.SourceType,
		&r.SourceID,
		&r.ProviderURL,
		&r.PlaybackRestriction,
		&r.Title,
		&r.Publisher,
		&r.ThumbnailURL,
		&r.Duration,
		&r.AddedBySessionID,
		&r.AddedBy,
		&r.AddedAt,
		&r.VoteCount,
	)
	if err != nil {
		return fmt.Errorf("error scanning added playlistItem row: %w", err)
	}

	return nil
}

func (r *addPlaylistItemRow) toPlaylistItem() (*vibe.PlaylistItem, error) {
	return &vibe.PlaylistItem{
		ID:                  r.ID.String,
		RoomID:              r.RoomID.String,
		SourceType:          r.SourceType.String,
		SourceID:            r.SourceID.String,
		ProviderURL:         r.ProviderURL.String,
		PlaybackRestriction: r.PlaybackRestriction.String,
		Title:               r.Title.String,
		Publisher:           r.Publisher.String,
		ThumbnailURL:        r.ThumbnailURL.String,
		Duration:            int(r.Duration.Int64),
		AddedBySessionID:    r.AddedBySessionID.String,
		AddedBy:             r.AddedBy.String,
		AddedAt:             r.AddedAt.Time,
		VoteCount:           int(r.VoteCount.Int64),
	}, nil
}

// prepareRemovePlaylistItemStmt prepares the RemovePlaylistItemStatement.
func (c *Client) prepareRemovePlaylistItemStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH deleted_skip_votes_q AS (
			DELETE FROM skip_votes
			WHERE room_id = $1
			AND playlist_item_id = $2
		),
		deleted_playlist_item_votes_q AS (
			DELETE FROM playlist_item_votes
			WHERE room_id = $1
			AND playlist_item_id = $2
		),
		stopped_playback_q AS (
			UPDATE playback_state
			SET current_playlist_item_id = NULL, is_playing = FALSE,
				position_ms = 0, updated_at = NOW()
			WHERE room_id = $1 AND current_playlist_item_id = $2
		)
		DELETE FROM playlist_items
		WHERE room_id = $1
		AND id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing RemovePlaylistItemStatement: %w", err)
	}

	c.RemovePlaylistItemStatement = stmt

	return nil
}

// RemovePlaylistItem removes a playlistItem from the queue.
func (c *Client) RemovePlaylistItem(ctx context.Context, roomID, playlistItemID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "RemovePlaylistItem")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.RemovePlaylistItemStatement.ExecContext(cctx, roomID, playlistItemID)
	if err != nil {
		return fmt.Errorf("error removing playlistItem: %w", err)
	}

	return nil
}

// prepareVotePlaylistItemStmt prepares the VotePlaylistItemStatement.
func (c *Client) prepareVotePlaylistItemStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH playlist_item_q AS (
			SELECT a.room_id, a.id
			FROM playlist_items a
			WHERE a.room_id = $1
			AND a.id = $2
			FOR KEY SHARE OF a
		),
		inserted_vote_q AS (
			INSERT INTO playlist_item_votes (room_id, playlist_item_id, user_id)
			SELECT a.room_id, a.id, $3
			FROM playlist_item_q a
			ON CONFLICT(room_id, playlist_item_id, user_id) DO NOTHING
			RETURNING 1
		)
		SELECT CASE
			WHEN NOT EXISTS (SELECT 1 FROM playlist_item_q)
				THEN 'playlistItem_not_found'
			WHEN EXISTS (SELECT 1 FROM inserted_vote_q)
				THEN 'created'
			ELSE 'already_voted'
		END
	`)
	if err != nil {
		return fmt.Errorf("error preparing VotePlaylistItemStatement: %w", err)
	}

	c.VotePlaylistItemStatement = stmt

	return nil
}

// VotePlaylistItem adds a vote for a playlistItem.
func (c *Client) VotePlaylistItem(ctx context.Context, roomID, playlistItemID, userID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "VotePlaylistItem")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.VotePlaylistItemStatement.QueryRowContext(cctx, roomID, playlistItemID, userID)

	var vote votePlaylistItemRow
	err := vote.scan(row)
	if err != nil {
		return fmt.Errorf("error scanning vote playlistItem outcome: %w", err)
	}

	if vote.Outcome.String == votePlaylistItemNotFound {
		return fmt.Errorf("error voting for playlistItem: playlistItem not found")
	}
	if vote.Outcome.String == votePlaylistItemAlreadyVoted {
		return internalerror.ErrAlreadyVoted{
			Err: fmt.Errorf("error voting for playlistItem: vote already exists"),
		}
	}
	if vote.Outcome.String != votePlaylistItemCreated {
		return fmt.Errorf(
			"error voting for playlistItem: unknown outcome %s",
			vote.Outcome.String,
		)
	}

	return nil
}

type votePlaylistItemRow struct {
	Outcome sql.NullString
}

func (r *votePlaylistItemRow) scan(row *sql.Row) error {
	err := row.Scan(&r.Outcome)
	if err != nil {
		return fmt.Errorf("error scanning playlistItem vote row: %w", err)
	}

	return nil
}

func (c *Client) prepareClearVotesPlaylistItemStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM playlist_item_votes WHERE room_id = $1 AND playlist_item_id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing ClearVotesPlaylistItemStatement: %w", err)
	}

	c.ClearVotesPlaylistItemStatement = stmt

	return nil
}

// clearVotesPlaylistItem clears all votes for a playlistItem.
func (c *Client) clearVotesPlaylistItem(ctx context.Context, roomID, playlistItemID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "clearVotesPlaylistItem")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.Printf("[DEBUG-VOTES] Clearing votes for playlistItem %s in room %s", playlistItemID, roomID)

	result, err := c.ClearVotesPlaylistItemStatement.ExecContext(cctx, roomID, playlistItemID)
	if err != nil {
		log.Printf("[DEBUG-VOTES] Error clearing votes for playlistItem %s in room %s: %v", playlistItemID, roomID, err)
		return fmt.Errorf("error clearing votes for playlistItem: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[DEBUG-VOTES] Error getting rows affected for playlistItem %s in room %s: %v", playlistItemID, roomID, err)
		return nil
	}
	log.Printf("[DEBUG-VOTES] Cleared %d votes for playlistItem %s in room %s", rowsAffected, playlistItemID, roomID)

	return nil
}

func (c *Client) prepareUpdatePlaylistItemAddedAtStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE playlist_items
		SET added_at = NOW()
		WHERE room_id = $1 AND id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdatePlaylistItemAddedAtStatement: %w", err)
	}

	c.UpdatePlaylistItemAddedAtStatement = stmt

	return nil
}

// updatePlaylistItemAddedAt updates the added_at timestamp for a playlistItem to treat it as "new"
func (c *Client) updatePlaylistItemAddedAt(ctx context.Context, roomID, playlistItemID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "updatePlaylistItemAddedAt")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.Printf("[DEBUG-VOTES] Updating added_at timestamp for playlistItem %s in room %s", playlistItemID, roomID)

	result, err := c.UpdatePlaylistItemAddedAtStatement.ExecContext(cctx, roomID, playlistItemID)
	if err != nil {
		log.Printf("[DEBUG-VOTES] Error updating added_at for playlistItem %s in room %s: %v", playlistItemID, roomID, err)
		return fmt.Errorf("error updating playlistItem added_at: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[DEBUG-VOTES] Error getting rows affected for playlistItem %s in room %s: %v", playlistItemID, roomID, err)
		return nil
	}
	log.Printf("[DEBUG-VOTES] Updated added_at for %d playlistItem(s) %s in room %s", rowsAffected, playlistItemID, roomID)

	return nil
}

const votePlaylistItemAlreadyVoted = "already_voted"
const votePlaylistItemCreated = "created"
const votePlaylistItemNotFound = "playlistItem_not_found"
const addPlaylistItemResultRoomNotFound = "room_not_found"
const addPlaylistItemResultProviderDisabled = "provider_disabled"
