// Package database contains a Postgres client and methods for communicating with the database.
package database

import (
	"context"
	"fmt"
	"net"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/zoff-music/vibes/config"
)

// Client holds the database client and prepared statements.
type Client struct {
	DB                         *sqlx.DB
	CloudSQLDialer             *cloudsqlconn.Dialer
	PgxPool                    *pgxpool.Pool
	GetExampleDataStatement    *sqlx.Stmt
	RecordExampleDataStatement *sqlx.Stmt
}

type prepareStatementFunc func() error

// Init sets up a new database client.
func (c *Client) Init(ctx context.Context, config *config.Config) error {
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s %s",
		config.DatabaseUser, config.DatabasePassword,
		config.DatabaseDB, config.DatabaseOptions)
	pgxConf, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse PGX DSN: %w", err)
	}

	pgxConf.MaxConns = int32(config.DatabaseMaxConnections) // #nosec G115 - There's no way this setting will ever go beyond an int32 max value
	pgxConf.MaxConnIdleTime = time.Minute * time.Duration(config.DatabaseMaxIdleTimeMinutes)

	dialer, err := cloudsqlconn.NewDialer(ctx, cloudsqlconn.WithDefaultDialOptions(
		cloudsqlconn.WithPrivateIP(), // default to private IP, since we're using Private Services Access
	))
	if err != nil {
		return fmt.Errorf("failed to create cloudsql dialer: %w", err)
	}
	c.CloudSQLDialer = dialer

	pgxConf.ConnConfig.DialFunc = func(ctx context.Context, _ string, _ string) (net.Conn, error) {
		return dialer.Dial(ctx, config.CloudSQLInstance)
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxConf)
	if err != nil {
		return fmt.Errorf("failed to create PGX pool: %w", err)
	}
	c.PgxPool = pool

	sqlDB := stdlib.OpenDBFromPool(pool)
	db := sqlx.NewDb(sqlDB, "pgx")

	err = db.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(config.DatabaseMaxConnections)
	db.SetMaxIdleConns(config.DatabaseMaxIdleConnections)
	db.SetConnMaxIdleTime(time.Minute * time.Duration(config.DatabaseMaxIdleTimeMinutes))

	c.DB = db

	preparedStatements := []prepareStatementFunc{
		c.prepareGetExampleDataStmt,
		c.prepareRecordExampleDataStmt,
	}

	for _, stmt := range preparedStatements {
		err = stmt()
		if err != nil {
			return fmt.Errorf("failed to prepare statements: %w", err)
		}
	}

	return nil
}

// Close closes the database connection and statements.
func (c *Client) Close() error {
	statements := []*sqlx.Stmt{
		c.GetExampleDataStatement,
		c.RecordExampleDataStatement,
	}

	for _, stmt := range statements {
		err := stmt.Close()
		if err != nil {
			return fmt.Errorf("error closing statements: %w", err)
		}
	}

	err := c.DB.Close()
	if err != nil {
		return fmt.Errorf("error closing database: %w", err)
	}

	err = c.CloudSQLDialer.Close()
	if err != nil {
		return fmt.Errorf("error closing cloudsql dialer: %w", err)
	}

	return nil
}
