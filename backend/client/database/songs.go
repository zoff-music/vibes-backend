package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"
)

// prepareGetSongsStmt prepares the GetSongsStatement.
func (c *Client) prepareGetSongsStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, room_id, source_type, source_id, title, artist, thumbnail_url, duration, added_by, added_by_nickname, added_at, position
		FROM songs
		WHERE room_id = ?1
		ORDER BY position ASC
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

		song, err := row.toSong()
		if err != nil {
			return nil, fmt.Errorf("error converting song row: %w", err)
		}

		songs = append(songs, *song)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating song rows: %w", err)
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
	)
}

func (r *songRow) toSong() (*vibe.Song, error) {
	var artist *string
	if r.Artist.Valid {
		artist = &r.Artist.String
	}

	var addedByNickname *string
	if r.AddedByNickname.Valid {
		addedByNickname = &r.AddedByNickname.String
	}

	if !r.AddedAt.Valid {
		return nil, fmt.Errorf("error missing song added_at")
	}

	duration := 0
	if r.Duration.Valid {
		duration = int(r.Duration.Int64)
	}

	position := 0
	if r.Position.Valid {
		position = int(r.Position.Int64)
	}

	return &vibe.Song{
		ID:              r.ID.String,
		RoomID:          r.RoomID.String,
		SourceType:      vibe.SourceType(r.SourceType.String),
		SourceID:        r.SourceID.String,
		Title:           r.Title.String,
		Artist:          artist,
		ThumbnailURL:    r.ThumbnailURL.String,
		Duration:        duration,
		AddedBy:         r.AddedBy.String,
		AddedByNickname: addedByNickname,
		AddedAt:         r.AddedAt.Time,
		Position:        position,
	}, nil
}

// prepareGetSongStmt prepares the GetSongStatement.
func (c *Client) prepareGetSongStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, room_id, source_type, source_id, title, artist, thumbnail_url, duration, added_by, added_by_nickname, added_at, position
		FROM songs
		WHERE room_id = ?1 AND id = ?2
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

	row := c.GetSongStatement.QueryRowContext(cctx, roomID, songID)

	var scanned songRow

	err := scanned.scan(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.Song{}, nil
		}

		return nil, fmt.Errorf("error fetching song: %w", err)
	}

	song, err := scanned.toSong()
	if err != nil {
		return nil, fmt.Errorf("error converting song row: %w", err)
	}

	return song, nil
}

// prepareAddSongStmt prepares the AddSongStatement.
func (c *Client) prepareAddSongStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO songs (room_id, source_type, source_id, title, artist, thumbnail_url, duration, added_by, added_by_nickname, added_at, position)
		VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11)
		RETURNING id
	`)
	if err != nil {
		return fmt.Errorf("error preparing AddSongStatement: %w", err)
	}

	c.AddSongStatement = stmt

	return nil
}

// AddSong adds a song to the queue.
func (c *Client) AddSong(ctx context.Context, song *vibe.Song) (*vibe.Song, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "AddSong")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var returnedID string
	err := c.AddSongStatement.QueryRowContext(cctx,
		// song.ID is removed
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
	).Scan(&returnedID)
	if err != nil {
		return nil, fmt.Errorf("error adding song: %w", err)
	}

	song.ID = returnedID

	return song, nil
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
		SELECT COALESCE(MAX(position), -1) FROM songs WHERE room_id = ?1
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

	row := c.GetMaxPositionStatement.QueryRowContext(cctx, roomID)

	var scanned maxPositionRow

	err := scanned.scan(row)
	if err != nil {
		return 0, fmt.Errorf("error getting max position: %w", err)
	}

	return scanned.toMaxPosition(), nil
}

type maxPositionRow struct {
	MaxPosition sql.NullInt64
}

func (r *maxPositionRow) scan(row *sql.Row) error {
	return row.Scan(&r.MaxPosition)
}

func (r *maxPositionRow) toMaxPosition() int {
	if !r.MaxPosition.Valid {
		return 0
	}

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

// prepareShiftPositionsDownStmt prepares the ShiftPositionsDownStatement.
func (c *Client) prepareShiftPositionsDownStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE songs SET position = position - 1
		WHERE room_id = ?1 AND position > ?2
	`)
	if err != nil {
		return fmt.Errorf("error preparing ShiftPositionsDownStatement: %w", err)
	}

	c.ShiftPositionsDownStatement = stmt

	return nil
}

// prepareShiftPositionsUpStmt prepares the ShiftPositionsUpStatement.
func (c *Client) prepareShiftPositionsUpStmt() error {
	stmt, err := c.DB.Prepare(`
		UPDATE songs SET position = position + 1
		WHERE room_id = ?1 AND position >= ?2 AND position < ?3
	`)
	if err != nil {
		return fmt.Errorf("error preparing ShiftPositionsUpStatement: %w", err)
	}

	c.ShiftPositionsUpStatement = stmt

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

	tx, err := c.DB.BeginTx(cctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer tx.Rollback()

	if newPosition < oldPosition {
		// Moving up: shift songs between newPosition and oldPosition down
		_, err = tx.StmtContext(cctx, c.ShiftPositionsUpStatement).ExecContext(cctx, roomID, newPosition, oldPosition)
		if err != nil {
			return fmt.Errorf("error shifting positions up: %w", err)
		}
	} else {
		// Moving down: shift songs between oldPosition and newPosition up
		_, err = tx.ExecContext(cctx, `
			UPDATE songs SET position = position - 1
			WHERE room_id = ?1 AND position > ?2 AND position <= ?3
		`, roomID, oldPosition, newPosition)
		if err != nil {
			return fmt.Errorf("error shifting positions down: %w", err)
		}
	}

	// Update the song's position
	_, err = tx.StmtContext(cctx, c.UpdateSongPositionStatement).ExecContext(cctx, newPosition, roomID, songID)
	if err != nil {
		return fmt.Errorf("error updating song position: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// prepareGetNextSongStmt prepares the GetNextSongStatement.
func (c *Client) prepareGetNextSongStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT id, room_id, source_type, source_id, title, artist, thumbnail_url, duration, added_by, added_by_nickname, added_at, position
		FROM songs
		WHERE room_id = ?1 AND position > ?2
		ORDER BY position ASC
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

	row := c.GetNextSongStatement.QueryRowContext(cctx, roomID, currentPosition)

	var scanned songRow

	err := scanned.scan(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &vibe.Song{}, nil
		}

		return nil, fmt.Errorf("error getting next song: %w", err)
	}

	song, err := scanned.toSong()
	if err != nil {
		return nil, fmt.Errorf("error converting song row: %w", err)
	}

	return song, nil
}
