package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/zoff-music/vibes/internalerror"
	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"
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
			a.position, 
			COUNT(b.user_id) as vote_count
		FROM songs a
		LEFT JOIN song_votes b 
		ON a.id = b.song_id 
		AND a.room_id = b.room_id
		WHERE a.room_id = ?1
		GROUP BY a.id
		ORDER BY vote_count DESC, a.position ASC
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetSongsStatement: %w", err)
	}

	c.GetSongsStatement = stmt

	return nil
}

// GetSongs fetches all songs in a room's queue.
func (c *Client) GetSongs(ctx context.Context, roomID string) ([]vibe.Song, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetSongs")
	defer span.Finish()

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
	Position        sql.NullInt64
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
		&r.Position,
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
		&r.Position,
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
		Position:        int(r.Position.Int64),
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
			a.position, 
			COUNT(b.user_id) as vote_count
		FROM songs a
		LEFT JOIN song_votes b 
		ON a.id = b.song_id 
		AND a.room_id = b.room_id
		WHERE a.room_id = ?1 
		AND a.id = ?2
		GROUP BY a.id
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetSongStatement: %w", err)
	}

	c.GetSongStatement = stmt

	return nil
}

// GetSong fetches a single song by ID.
func (c *Client) GetSong(ctx context.Context, roomID, songID string) (*vibe.Song, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetSong")
	defer span.Finish()

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
		INSERT INTO songs (room_id, source_type, source_id, title, artist, thumbnail_url, duration, added_by, added_by_nickname, added_at, position)
		VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11)
		ON CONFLICT(room_id, source_type, source_id) DO UPDATE SET
			room_id = excluded.room_id
		RETURNING id
	`)
	if err != nil {
		return fmt.Errorf("error preparing AddSongStatement: %w", err)
	}

	c.AddSongStatement = stmt

	return nil
}

func (c *Client) prepareInsertSongVoteStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO song_votes (room_id, song_id, user_id)
		VALUES (?1, ?2, ?3)
		ON CONFLICT(room_id, song_id, user_id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("error preparing InsertSongVoteStatement: %w", err)
	}
	c.InsertSongVoteStatement = stmt
	return nil
}

// AddSong adds a song to the queue.
func (c *Client) AddSong(ctx context.Context, song *vibe.Song) (*vibe.Song, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "AddSong")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 1. Insert Song (idempotent, returns ID)

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
		song.AddedAt,
		song.Position,
	)

	var returnedID string
	err := r.Scan(&returnedID)
	if err != nil {
		return nil, fmt.Errorf("error inserting song: %w", err)
	}

	song.ID = returnedID

	// 2. Insert Vote
	_, err = c.InsertSongVoteStatement.ExecContext(cctx, song.RoomID, song.ID, song.AddedBy)
	if err != nil {
		return nil, fmt.Errorf("error inserting vote: %w", err)
	}

	// 3. Fetch full song details to get accurate vote count
	updatedSong, err := c.GetSong(ctx, song.RoomID, song.ID)
	if err != nil {
		return nil, fmt.Errorf("error fetching updated song: %w", err)
	}

	return updatedSong, nil
}

// prepareRemoveSongStmt prepares the RemoveSongStatement.
func (c *Client) prepareRemoveSongStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM songs
		WHERE room_id = ?1 AND id = ?2
	`)
	if err != nil {
		return fmt.Errorf("error preparing RemoveSongStatement: %w", err)
	}

	c.RemoveSongStatement = stmt

	return nil
}

// RemoveSong removes a song from the queue.
func (c *Client) RemoveSong(ctx context.Context, roomID, songID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RemoveSong")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.RemoveSongStatement.ExecContext(cctx, roomID, songID)
	if err != nil {
		return fmt.Errorf("error removing song: %w", err)
	}

	return nil
}

// prepareGetMaxPositionStmt prepares the GetMaxPositionStatement.
func (c *Client) prepareGetMaxPositionStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT COALESCE(MAX(position), 0) FROM songs WHERE room_id = ?1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetMaxPositionStatement: %w", err)
	}

	c.GetMaxPositionStatement = stmt

	return nil
}

// GetMaxPosition gets the maximum position in a room's queue.
func (c *Client) GetMaxPosition(ctx context.Context, roomID string) (int, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetMaxPosition")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetMaxPositionStatement.QueryRowContext(cctx, roomID)

	var row maxPositionRow
	err := row.scan(r)
	if err != nil {
		return 0, fmt.Errorf("error getting max position: %w", err)
	}

	return row.toMaxPosition(), nil
}

type maxPositionRow struct {
	MaxPosition sql.NullInt64
}

func (r *maxPositionRow) scan(row *sql.Row) error {
	return row.Scan(
		&r.MaxPosition,
	)
}

func (r *maxPositionRow) toMaxPosition() int {
	return int(r.MaxPosition.Int64)
}

// prepareUpdateSongPositionStmt prepares the UpdateSongPositionStatement.
func (c *Client) prepareUpdateSongPositionStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE songs SET position = ?1 WHERE room_id = ?2 AND id = ?3
	`)
	if err != nil {
		return fmt.Errorf("error preparing UpdateSongPositionStatement: %w", err)
	}

	c.UpdateSongPositionStatement = stmt

	return nil
}

// prepareReorderSongsStmt prepares the ReorderSongsStatement.
func (c *Client) prepareReorderSongsStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE songs
		SET position = CASE
			WHEN id = ?1 THEN ?2
			WHEN ?2 < ?3 AND position >= ?2 AND position < ?3 THEN position + 1
			WHEN ?2 > ?3 AND position > ?3 AND position <= ?2 THEN position - 1
			ELSE position
		END
		WHERE room_id = ?4
		AND (
			id = ?1
			OR (?2 < ?3 AND position >= ?2 AND position < ?3)
			OR (?2 > ?3 AND position > ?3 AND position <= ?2)
		)
	`)
	if err != nil {
		return fmt.Errorf("error preparing ReorderSongsStatement: %w", err)
	}

	c.ReorderSongsStatement = stmt

	return nil
}

// ReorderSongs moves a song to a new position.
func (c *Client) ReorderSongs(ctx context.Context, roomID, songID string, newPosition int) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ReorderSongs")
	defer span.Finish()

	// Get current song position
	song, err := c.GetSong(ctx, roomID, songID)
	if err != nil {
		return fmt.Errorf("error getting song for reorder: %w", err)
	}

	if song.IsEmpty() {
		return fmt.Errorf("error song not found")
	}

	oldPosition := song.Position
	if oldPosition == newPosition {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = c.ReorderSongsStatement.ExecContext(cctx,
		songID,
		newPosition,
		oldPosition,
		roomID,
	)
	if err != nil {
		return fmt.Errorf("error reordering songs: %w", err)
	}

	return nil
}

// prepareVoteSongStmt prepares the VoteSongStatement.
func (c *Client) prepareVoteSongStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO song_votes (room_id, song_id, user_id)
		VALUES (?1, ?2, ?3)
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "VoteSong")
	defer span.Finish()

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
		DELETE FROM song_votes WHERE room_id = ?1 AND song_id = ?2
	`)
	if err != nil {
		return fmt.Errorf("error preparing ClearVotesSongStatement: %w", err)
	}

	c.ClearVotesSongStatement = stmt

	return nil
}

// clearVotesSong clears all votes for a song.
func (c *Client) clearVotesSong(ctx context.Context, roomID, songID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "clearVotesSong")
	defer span.Finish()

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
	} else {
		log.Printf("[DEBUG-VOTES] Cleared %d votes for song %s in room %s", rowsAffected, songID, roomID)
	}

	return nil
}

// prepareGetNextSongStmt prepares the GetNextSongStatement.
func (c *Client) prepareGetNextSongStmt() error {
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
			a.position,
			COUNT(b.user_id) as vote_count
		FROM songs a
		LEFT JOIN song_votes b 
		ON a.id = b.song_id 
		AND a.room_id = b.room_id
		WHERE a.room_id = ?1 
		AND a.position > ?2
		GROUP BY a.id
		ORDER BY a.position ASC
		LIMIT 1
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetNextSongStatement: %w", err)
	}

	c.GetNextSongStatement = stmt

	return nil
}

// GetNextSong gets the next song in the queue after the given position.
func (c *Client) GetNextSong(ctx context.Context, roomID string, currentPosition int) (*vibe.Song, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetNextSong")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.GetNextSongStatement.QueryRowContext(cctx, roomID, currentPosition)

	var row songRow
	err := row.scan(r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.Song{}, nil
		}

		return nil, fmt.Errorf("error getting next song: %w", err)
	}

	song := row.toSong()
	return &song, nil
}
