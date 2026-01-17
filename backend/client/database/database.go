// Package database contains a SQLite client for lab.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zoff-music/vibes/config"
	"github.com/zoff-music/vibes/monitoring/opentracing"
	_ "modernc.org/sqlite"
)

// Client holds the database client and prepared statements.
type Client struct {
	DB *sql.DB

	maxNameLength int
}

// Init sets up a new database client.
func (c *Client) Init(ctx context.Context, cfg *config.Config) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Init")
	defer span.Finish()

	maxNameLength := cfg.MaxNameLength
	if maxNameLength == 0 {
		maxNameLength = 40
	}
	c.maxNameLength = maxNameLength

	db, err := sql.Open("sqlite", cfg.HighscoreDB)
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
	statements := []*sql.Stmt{}
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
	preparedStatements := []prepareStatementFunc{}

	for _, stmt := range preparedStatements {
		err := stmt()
		if err != nil {
			return fmt.Errorf("error in db: prepare statement: %w", err)
		}
	}
	return nil
}
