// Package database contains a SQLite client for Vibes.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zoff-music/vibes/config"
	"github.com/zoff-music/vibes/monitoring/opentracing"
	_ "modernc.org/sqlite"
)

// Client holds the database client and prepared statements.
type Client struct {
	DB *sql.DB

	maxNameLength  int
	maxQueueLength int

	// Room statements
	GetRoomStatement            *sql.Stmt
	GetRoomByNameStatement      *sql.Stmt
	CreateRoomStatement         *sql.Stmt
	CreateRoomSettingsStatement *sql.Stmt
	UpdateRoomStatement         *sql.Stmt
	UpdateRoomSettingsStatement *sql.Stmt

	// Song statements
	GetSongsStatement           *sql.Stmt
	GetSongStatement            *sql.Stmt
	AddSongStatement            *sql.Stmt
	RemoveSongStatement         *sql.Stmt
	GetMaxPositionStatement     *sql.Stmt
	UpdateSongPositionStatement *sql.Stmt
	GetNextSongStatement        *sql.Stmt
	ShiftPositionsDownStatement *sql.Stmt
	ShiftPositionsUpStatement   *sql.Stmt

	// Playback statements
	GetPlaybackStateStatement    *sql.Stmt
	UpsertPlaybackStateStatement *sql.Stmt

	// User statements
	GetUserStatement              *sql.Stmt
	GetUsersInRoomStatement       *sql.Stmt
	CountUsersInRoomStatement     *sql.Stmt
	CreateUserStatement           *sql.Stmt
	UpdateUserLastSeenStatement   *sql.Stmt
	RemoveUserStatement           *sql.Stmt
	CleanupInactiveUsersStatement *sql.Stmt

	// Skip vote statements
	GetSkipVotesStatement   *sql.Stmt
	HasUserVotedStatement   *sql.Stmt
	AddSkipVoteStatement    *sql.Stmt
	ClearSkipVotesStatement *sql.Stmt
}

// ... (Init function is unchanged, but I need to make sure I don't overwrite it in ReplaceContent)
// Wait, I can't just replace the struct and the method in one go if they are far apart.
// I will do two chunks using multi_replace_file_content since the tool call above is replace_file_content.
// Actually, I can just use multi_replace for this.

// Init sets up a new database client.
func (c *Client) Init(ctx context.Context, cfg *config.Config) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Init")
	defer span.Finish()

	c.maxNameLength = cfg.MaxNameLength
	if c.maxNameLength == 0 {
		c.maxNameLength = 100
	}

	c.maxQueueLength = cfg.MaxQueueLength
	if c.maxQueueLength == 0 {
		c.maxQueueLength = 200
	}

	// Ensure the directory exists
	dir := filepath.Dir(cfg.DatabasePath)
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("error in db: create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("error in db: open sqlite: %w", err)
	}

	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err = db.ExecContext(cctx, "PRAGMA journal_mode = WAL;")
	if err != nil {
		return fmt.Errorf("error in db: set journal_mode: %w", err)
	}

	cctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err = db.ExecContext(cctx, "PRAGMA synchronous = NORMAL;")
	if err != nil {
		return fmt.Errorf("error in db: set synchronous: %w", err)
	}

	cctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err = db.ExecContext(cctx, "PRAGMA foreign_keys = ON;")
	if err != nil {
		return fmt.Errorf("error in db: enable foreign keys: %w", err)
	}

	c.DB = db

	err = c.migrateSchema(ctx)
	if err != nil {
		return fmt.Errorf("error in db: migrate schema: %w", err)
	}

	err = c.prepareStatements()
	if err != nil {
		return fmt.Errorf("error in db: prepare statements: %w", err)
	}

	return nil
}

// Close closes the database connection and statements.
func (c *Client) Close() error {
	statements := []*sql.Stmt{
		c.GetRoomStatement,
		c.GetRoomByNameStatement,
		c.CreateRoomStatement,
		c.CreateRoomSettingsStatement,
		c.UpdateRoomStatement,
		c.UpdateRoomSettingsStatement,
		c.GetSongsStatement,
		c.GetSongStatement,
		c.AddSongStatement,
		c.RemoveSongStatement,
		c.GetMaxPositionStatement,
		c.UpdateSongPositionStatement,
		c.GetNextSongStatement,
		c.ShiftPositionsDownStatement,
		c.ShiftPositionsUpStatement,
		c.GetPlaybackStateStatement,
		c.UpsertPlaybackStateStatement,
		c.GetUserStatement,
		c.GetUsersInRoomStatement,
		c.CountUsersInRoomStatement,
		c.CreateUserStatement,
		c.UpdateUserLastSeenStatement,
		c.RemoveUserStatement,
		c.CleanupInactiveUsersStatement,
		c.GetSkipVotesStatement,
		c.HasUserVotedStatement,
		c.AddSkipVoteStatement,
		c.ClearSkipVotesStatement,
	}

	for _, stmt := range statements {
		if stmt == nil {
			continue
		}

		err := stmt.Close()
		if err != nil {
			return fmt.Errorf("error in db: close statement: %w", err)
		}
	}

	if c.DB == nil {
		return nil
	}

	err := c.DB.Close()
	if err != nil {
		return fmt.Errorf("error in db: close database: %w", err)
	}

	return nil
}

type prepareStatementFunc func() error

func (c *Client) prepareStatements() error {
	preparedStatements := []prepareStatementFunc{
		// Room statements
		c.prepareGetRoomStmt,
		c.prepareGetRoomByNameStmt,
		c.prepareCreateRoomStmt,
		c.prepareCreateRoomSettingsStmt,
		c.prepareUpdateRoomStmt,
		c.prepareUpdateRoomSettingsStmt,
		// Song statements
		c.prepareGetSongsStmt,
		c.prepareGetSongStmt,
		c.prepareAddSongStmt,
		c.prepareRemoveSongStmt,
		c.prepareGetMaxPositionStmt,
		c.prepareUpdateSongPositionStmt,
		c.prepareGetNextSongStmt,
		c.prepareShiftPositionsDownStmt,
		c.prepareShiftPositionsUpStmt,
		// Playback statements
		c.prepareGetPlaybackStateStmt,
		c.prepareUpsertPlaybackStateStmt,
		// User statements
		c.prepareGetUserStmt,
		c.prepareGetUsersInRoomStmt,
		c.prepareCountUsersInRoomStmt,
		c.prepareCreateUserStmt,
		c.prepareUpdateUserLastSeenStmt,
		c.prepareRemoveUserStmt,
		c.prepareCleanupInactiveUsersStmt,
		// Skip vote statements
		c.prepareGetSkipVotesStmt,
		c.prepareHasUserVotedStmt,
		c.prepareAddSkipVoteStmt,
		c.prepareClearSkipVotesStmt,
	}

	for _, stmt := range preparedStatements {
		err := stmt()
		if err != nil {
			return fmt.Errorf("error in db: prepare statement: %w", err)
		}
	}

	return nil
}
