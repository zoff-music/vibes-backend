package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

// prepareGetSongsStmt prepares the GetSongsStatement.
func (c *Client) prepareGetSongsStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			a.id,
			a.room_id,
			a.source_type,
			a.source_id,
			a.title,
			a.artist,
			a.thumbnail_url,
			a.duration,
			a.added_by,
			a.added_by_nickname,
			a.added_at,
			COUNT(b.user_id) as vote_count
		FROM songs a
		LEFT JOIN song_votes b
		ON a.id = b.song_id
		AND a.room_id = b.room_id
		WHERE a.room_id = $1
		GROUP BY a.id, a.room_id, a.source_type, a.source_id, a.title, a.artist, a.thumbnail_url, a.duration, a.added_by, a.added_by_nickname, a.added_at
		ORDER BY vote_count DESC, MAX(b.created_at) ASC, a.added_at ASC
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetSongsStatement: %w", err)
	}

	c.GetSongsStatement = stmt

	return nil
}

// GetSongs fetches all songs in a room's queue.
func (c *Client) GetSongs(ctx context.Context, roomID string) ([]vibe.Song, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetSongs")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.GetSongsStatement.QueryContext(cctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error fetching songs: %w", err)
	}
	defer rows.Close()

	songs := []vibe.Song{}
	for rows.Next() {
		var row songRow

		err := row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning song row: %w", err)
		}

		songs = append(songs, row.toSong())
	}

	return songs, nil
}

type songRow struct {
	ID              sql.NullString
	RoomID          sql.NullString
	SourceType      sql.NullString
	SourceID        sql.NullString
	Title           sql.NullString
	Artist          sql.NullString
	ThumbnailURL    sql.NullString
	Duration        sql.NullInt64
	AddedBy         sql.NullString
	AddedByNickname sql.NullString
	AddedAt         sql.NullTime
	VoteCount       sql.NullInt64
}

func (r *songRow) scanRows(rows *sql.Rows) error {
	return rows.Scan(
		&r.ID,
		&r.RoomID,
		&r.SourceType,
		&r.SourceID,
		&r.Title,
		&r.Artist,
		&r.ThumbnailURL,
		&r.Duration,
		&r.AddedBy,
		&r.AddedByNickname,
		&r.AddedAt,
		&r.VoteCount,
	)
}

func (r *songRow) scan(row *sql.Row) error {
	return row.Scan(
		&r.ID,
		&r.RoomID,
		&r.SourceType,
		&r.SourceID,
		&r.Title,
		&r.Artist,
		&r.ThumbnailURL,
		&r.Duration,
		&r.AddedBy,
		&r.AddedByNickname,
		&r.AddedAt,
		&r.VoteCount,
	)
}

func (r *songRow) toSong() vibe.Song {
	return vibe.Song{
		ID:              r.ID.String,
		RoomID:          r.RoomID.String,
		SourceType:      vibe.SourceType(r.SourceType.String),
		SourceID:        r.SourceID.String,
		Title:           r.Title.String,
		Artist:          r.Artist.String,
		ThumbnailURL:    r.ThumbnailURL.String,
		Duration:        int(r.Duration.Int64),
		AddedBy:         r.AddedBy.String,
		AddedByNickname: r.AddedByNickname.String,
		AddedAt:         r.AddedAt.Time,
		VoteCount:       int(r.VoteCount.Int64),
	}
}

// prepareGetSongStmt prepares the GetSongStatement.
func (c *Client) prepareGetSongStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT
			a.id,
			a.room_id,
			a.source_type,
			a.source_id,
			a.title,
			a.artist,
			a.thumbnail_url,
			a.duration,
			a.added_by,
			a.added_by_nickname,
			a.added_at,
			COUNT(b.user_id) as vote_count
		FROM songs a
		LEFT JOIN song_votes b
		ON a.id = b.song_id
		AND a.room_id = b.room_id
		WHERE a.room_id = $1
		AND a.id = $2
		GROUP BY a.id, a.room_id, a.source_type, a.source_id, a.title, a.artist, a.thumbnail_url, a.duration, a.added_by, a.added_by_nickname, a.added_at
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetSongStatement: %w", err)
	}

	c.GetSongStatement = stmt

	return nil
}

// GetSong fetches a single song by ID.
func (c *Client) GetSong(ctx context.Context, roomID, songID string) (*vibe.Song, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetSong")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetSongStatement.QueryRowContext(cctx, roomID, songID)

	var row songRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.Song{}, nil
		}

		return nil, fmt.Errorf("error fetching song: %w", err)
	}

	song := row.toSong()
	return &song, nil
}

// prepareAddSongStmt prepares the AddSongStatement.
func (c *Client) prepareAddSongStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH room_config_q AS (
			SELECT
				a.id AS room_id,
				COALESCE(b.allow_duplicates, FALSE) AS allow_duplicates
			FROM rooms a
			LEFT JOIN room_settings b ON b.room_id = a.id
			WHERE a.id = $1
		),
		existing_song_q AS (
			SELECT 1
			FROM songs a
			WHERE a.room_id = $1
			AND a.source_type = $2
			AND a.source_id = $3
			LIMIT 1
		),
		inserted_song_q AS (
			INSERT INTO songs (
				room_id,
				source_type,
				source_id,
				title,
				artist,
				thumbnail_url,
				duration,
				added_by,
				added_by_nickname,
				added_at,
				duplicate_guard
			)
			SELECT
				a.room_id,
				$2,
				$3,
				$4,
				$5,
				$6,
				$7,
				$8,
				$9,
				NOW(),
				NOT a.allow_duplicates
			FROM room_config_q a
			WHERE a.allow_duplicates
			OR NOT EXISTS (SELECT 1 FROM existing_song_q)
			ON CONFLICT (room_id, source_type, source_id)
			WHERE duplicate_guard
			DO NOTHING
			RETURNING
				id,
				room_id,
				source_type,
				source_id,
				title,
				artist,
				thumbnail_url,
				duration,
				added_by,
				added_by_nickname,
				added_at
		),
		inserted_vote_q AS (
			INSERT INTO song_votes (room_id, song_id, user_id)
			SELECT a.room_id, a.id, a.added_by
			FROM inserted_song_q a
			ON CONFLICT (room_id, song_id, user_id) DO NOTHING
			RETURNING 1
		)
		SELECT
			'inserted' AS result,
			a.id,
			a.room_id,
			a.source_type,
			a.source_id,
			a.title,
			a.artist,
			a.thumbnail_url,
			a.duration,
			a.added_by,
			a.added_by_nickname,
			a.added_at,
			1::BIGINT AS vote_count
		FROM inserted_song_q a
		UNION ALL
		SELECT
			CASE
				WHEN EXISTS (SELECT 1 FROM room_config_q) THEN 'duplicate'
				ELSE 'room_not_found'
			END AS result,
			NULL::TEXT AS id,
			NULL::TEXT AS room_id,
			NULL::TEXT AS source_type,
			NULL::TEXT AS source_id,
			NULL::TEXT AS title,
			NULL::TEXT AS artist,
			NULL::TEXT AS thumbnail_url,
			NULL::INTEGER AS duration,
			NULL::TEXT AS added_by,
			NULL::TEXT AS added_by_nickname,
			NULL::TIMESTAMP AS added_at,
			0::BIGINT AS vote_count
		WHERE NOT EXISTS (SELECT 1 FROM inserted_song_q)
	`)
	if err != nil {
		return fmt.Errorf("error preparing AddSongStatement: %w", err)
	}

	c.AddSongStatement = stmt

	return nil
}

// AddSong adds a song to the queue.
func (c *Client) AddSong(ctx context.Context, song *vibe.Song) (*vibe.Song, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AddSong")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.AddSongStatement.QueryRowContext(cctx,
		song.RoomID,
		string(song.SourceType),
		song.SourceID,
		song.Title,
		song.Artist,
		song.ThumbnailURL,
		song.Duration,
		song.AddedBy,
		song.AddedByNickname,
	)

	var row addSongRow
	err := row.scan(r)
	if err != nil {
		return nil, fmt.Errorf("error adding song atomically in AddSong: %w", err)
	}

	if row.Result.String == addSongResultDuplicate {
		return nil, internalerror.ErrDuplicateSong{
			Err: fmt.Errorf("song already exists"),
		}
	}

	if row.Result.String == addSongResultRoomNotFound {
		return nil, fmt.Errorf("error adding song in AddSong: room %s not found", song.RoomID)
	}

	addedSong := row.toSong()
	return &addedSong, nil
}

type addSongRow struct {
	Result          sql.NullString
	ID              sql.NullString
	RoomID          sql.NullString
	SourceType      sql.NullString
	SourceID        sql.NullString
	Title           sql.NullString
	Artist          sql.NullString
	ThumbnailURL    sql.NullString
	Duration        sql.NullInt64
	AddedBy         sql.NullString
	AddedByNickname sql.NullString
	AddedAt         sql.NullTime
	VoteCount       sql.NullInt64
}

func (r *addSongRow) scan(row *sql.Row) error {
	return row.Scan(
		&r.Result,
		&r.ID,
		&r.RoomID,
		&r.SourceType,
		&r.SourceID,
		&r.Title,
		&r.Artist,
		&r.ThumbnailURL,
		&r.Duration,
		&r.AddedBy,
		&r.AddedByNickname,
		&r.AddedAt,
		&r.VoteCount,
	)
}

func (r *addSongRow) toSong() vibe.Song {
	return vibe.Song{
		ID:              r.ID.String,
		RoomID:          r.RoomID.String,
		SourceType:      vibe.SourceType(r.SourceType.String),
		SourceID:        r.SourceID.String,
		Title:           r.Title.String,
		Artist:          r.Artist.String,
		ThumbnailURL:    r.ThumbnailURL.String,
		Duration:        int(r.Duration.Int64),
		AddedBy:         r.AddedBy.String,
		AddedByNickname: r.AddedByNickname.String,
		AddedAt:         r.AddedAt.Time,
		VoteCount:       int(r.VoteCount.Int64),
	}
}

// prepareRemoveSongStmt prepares the RemoveSongStatement.
func (c *Client) prepareRemoveSongStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM songs
		WHERE room_id = $1 AND id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing RemoveSongStatement: %w", err)
	}

	c.RemoveSongStatement = stmt

	return nil
}

// RemoveSong removes a song from the queue.
func (c *Client) RemoveSong(ctx context.Context, roomID, songID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "RemoveSong")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.RemoveSongStatement.ExecContext(cctx, roomID, songID)
	if err != nil {
		return fmt.Errorf("error removing song: %w", err)
	}

	return nil
}

// prepareVoteSongStmt prepares the VoteSongStatement.
func (c *Client) prepareVoteSongStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO song_votes (room_id, song_id, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT(room_id, song_id, user_id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("error preparing VoteSongStatement: %w", err)
	}

	c.VoteSongStatement = stmt

	return nil
}

// VoteSong adds a vote for a song.
func (c *Client) VoteSong(ctx context.Context, roomID, songID, userID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "VoteSong")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := c.VoteSongStatement.ExecContext(cctx, roomID, songID, userID)
	if err != nil {
		return fmt.Errorf("error voting for song: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rows == 0 {
		return internalerror.ErrAlreadyVoted{
			Err: fmt.Errorf("error voting for song: %w", err),
		}
	}

	return nil
}

func (c *Client) prepareClearVotesSongStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM song_votes WHERE room_id = $1 AND song_id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing ClearVotesSongStatement: %w", err)
	}

	c.ClearVotesSongStatement = stmt

	return nil
}

// clearVotesSong clears all votes for a song.
func (c *Client) clearVotesSong(ctx context.Context, roomID, songID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "clearVotesSong")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.Printf("[DEBUG-VOTES] Clearing votes for song %s in room %s", songID, roomID)

	result, err := c.ClearVotesSongStatement.ExecContext(cctx, roomID, songID)
	if err != nil {
		log.Printf("[DEBUG-VOTES] Error clearing votes for song %s in room %s: %v", songID, roomID, err)
		return fmt.Errorf("error clearing votes for song: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[DEBUG-VOTES] Error getting rows affected for song %s in room %s: %v", songID, roomID, err)
		return nil
	}
	log.Printf("[DEBUG-VOTES] Cleared %d votes for song %s in room %s", rowsAffected, songID, roomID)

	return nil
}

func (c *Client) prepareUpdateSongAddedAtStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE songs
		SET added_at = NOW()
		WHERE room_id = $1 AND id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdateSongAddedAtStatement: %w", err)
	}

	c.UpdateSongAddedAtStatement = stmt

	return nil
}

// updateSongAddedAt updates the added_at timestamp for a song to treat it as "new"
func (c *Client) updateSongAddedAt(ctx context.Context, roomID, songID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "updateSongAddedAt")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.Printf("[DEBUG-VOTES] Updating added_at timestamp for song %s in room %s", songID, roomID)

	result, err := c.UpdateSongAddedAtStatement.ExecContext(cctx, roomID, songID)
	if err != nil {
		log.Printf("[DEBUG-VOTES] Error updating added_at for song %s in room %s: %v", songID, roomID, err)
		return fmt.Errorf("error updating song added_at: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[DEBUG-VOTES] Error getting rows affected for song %s in room %s: %v", songID, roomID, err)
		return nil
	}
	log.Printf("[DEBUG-VOTES] Updated added_at for %d song(s) %s in room %s", rowsAffected, songID, roomID)

	return nil
}

const addSongResultDuplicate = "duplicate"
const addSongResultRoomNotFound = "room_not_found"
