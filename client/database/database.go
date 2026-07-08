// Package database contains a Postgres client for Vibes.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/opentracing"
)

// Client holds the database client and prepared statements.
type Client struct {
	DB *sql.DB

	maxNameLength  int
	maxQueueLength int

	// Room statements
	GetRoomStatement                  *sql.Stmt
	GetRoomByNameStatement            *sql.Stmt
	CreateRoomStatement               *sql.Stmt
	UpdateRoomStatement               *sql.Stmt
	ProcessNextAbandonedHostStatement *sql.Stmt

	// Song statements
	GetSongsStatement             *sql.Stmt
	GetSongStatement              *sql.Stmt
	AddSongStatement              *sql.Stmt
	RemoveSongStatement           *sql.Stmt
	GetNextSongStatement          *sql.Stmt
	GetNextSongExcludingStatement *sql.Stmt
	VoteSongStatement             *sql.Stmt
	ClearVotesSongStatement       *sql.Stmt
	UpdateSongAddedAtStatement    *sql.Stmt

	// Playback statements
	GetPlaybackStateStatement           *sql.Stmt
	UpsertPlaybackStateStatement        *sql.Stmt
	ProcessNextExpiredPlaybackStatement *sql.Stmt
	StartPlaybackIfIdleStatement        *sql.Stmt

	// User statements
	GetUserStatement    *sql.Stmt
	CreateUserStatement *sql.Stmt

	// Skip vote statements
	GetSkipVotesStatement   *sql.Stmt
	HasUserVotedStatement   *sql.Stmt
	AddSkipVoteStatement    *sql.Stmt
	ClearSkipVotesStatement *sql.Stmt

	// Auth token statements
	UpsertAuthTokenStatement         *sql.Stmt
	GetAuthProvidersStatement        *sql.Stmt
	DeleteExpiredAuthTokensStatement *sql.Stmt

	// Access token statements
	UpsertAccessTokenStatement         *sql.Stmt
	GetAccessTokenStatement            *sql.Stmt
	DeleteExpiredAccessTokensStatement *sql.Stmt

	// Pending OAuth state statements
	SavePendingOAuthStateStatement           *sql.Stmt
	ValidatePendingOAuthStateStatement       *sql.Stmt
	DeletePendingOAuthStateStatement         *sql.Stmt
	DeleteExpiredPendingOAuthStatesStatement *sql.Stmt

	// Token cleanup statements
	ClaimAndGetExpiredTokenForRefreshStatement *sql.Stmt

	// Participant statements
	UpdateParticipantStatement          *sql.Stmt
	GetActiveParticipantsStatement      *sql.Stmt
	GetActiveListenerCountsStatement    *sql.Stmt
	SetRoomHostStatement                *sql.Stmt
	RemoveParticipantStatement          *sql.Stmt
	DeleteInactiveParticipantsStatement *sql.Stmt

	// Additional room statements
	ElectNewHostStatement     *sql.Stmt
	GetActiveSourcesStatement *sql.Stmt
	GetAdminRoomsStatement    *sql.Stmt
}

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

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("error in db: open postgres: %w", err)
	}

	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxOpenConns(cfg.DatabaseMaxConns)
	db.SetMaxIdleConns(cfg.DatabaseMaxIdleConns)

	c.DB = db

	prepareStatements := []func() error{
		// Room statements
		c.prepareGetRoomStmt,
		c.prepareGetRoomByNameStmt,
		c.prepareCreateRoomStmt,
		c.prepareUpdateRoomStmt,
		c.prepareProcessNextAbandonedHostStmt,
		c.prepareElectNewHostStmt,
		c.prepareGetActiveSourcesStmt,
		c.prepareGetAdminRoomsStmt,
		// Song statements
		c.prepareGetSongsStmt,
		c.prepareGetSongStmt,
		c.prepareAddSongStmt,
		c.prepareRemoveSongStmt,
		c.prepareGetNextSongStmt,
		c.prepareGetNextSongExcludingStmt,
		c.prepareVoteSongStmt,
		c.prepareClearVotesSongStmt,
		c.prepareUpdateSongAddedAtStmt,
		// Playback statements
		c.prepareGetPlaybackStateStmt,
		c.prepareUpsertPlaybackStateStmt,
		c.prepareProcessNextExpiredPlaybackStmt,
		c.prepareStartPlaybackIfIdleStmt,
		// User statements
		c.prepareGetUserStmt,
		c.prepareCreateUserStmt,
		// Skip vote statements
		c.prepareGetSkipVotesStmt,
		c.prepareHasUserVotedStmt,
		c.prepareAddSkipVoteStmt,
		c.prepareClearSkipVotesStmt,
		// Auth token statements
		c.prepareUpsertAuthTokenStmt,
		c.prepareGetAuthProvidersStmt,
		c.prepareDeleteExpiredAuthTokensStmt,
		// Access token statements
		c.prepareUpsertAccessTokenStmt,
		c.prepareGetAccessTokenStmt,
		c.prepareDeleteExpiredAccessTokensStmt,
		c.prepareSavePendingOAuthStateStmt,
		c.prepareValidatePendingOAuthStateStmt,
		c.prepareDeletePendingOAuthStateStmt,
		c.prepareDeleteExpiredPendingOAuthStatesStmt,
		c.prepareClaimAndGetExpiredTokenForRefreshStmt,
		// Participant statements
		c.prepareUpdateParticipantStmt,
		c.prepareGetActiveParticipantsStmt,
		c.prepareGetActiveListenerCountsStmt,
		c.prepareSetRoomHostStmt,
		c.prepareRemoveParticipantStmt,
		c.prepareDeleteInactiveParticipantsStmt,
	}

	for _, prepareStmt := range prepareStatements {
		err := prepareStmt()
		if err != nil {
			return fmt.Errorf("error preparing statements: %w", err)
		}
	}

	return nil
}

// Close closes the database connection and statements.
func (c *Client) Close() error {
	statements := []*sql.Stmt{
		c.GetRoomStatement,
		c.GetRoomByNameStatement,
		c.CreateRoomStatement,
		c.UpdateRoomStatement,
		c.ProcessNextAbandonedHostStatement,
		c.GetSongsStatement,
		c.GetSongStatement,
		c.AddSongStatement,
		c.RemoveSongStatement,
		c.GetNextSongStatement,
		c.GetNextSongExcludingStatement,
		c.VoteSongStatement,
		c.ClearVotesSongStatement,
		c.UpdateSongAddedAtStatement,
		c.GetPlaybackStateStatement,
		c.UpsertPlaybackStateStatement,
		c.ProcessNextExpiredPlaybackStatement,
		c.StartPlaybackIfIdleStatement,
		c.GetUserStatement,
		c.CreateUserStatement,
		c.GetSkipVotesStatement,
		c.HasUserVotedStatement,
		c.AddSkipVoteStatement,
		c.ClearSkipVotesStatement,
		c.UpsertAuthTokenStatement,
		c.GetAuthProvidersStatement,
		c.DeleteExpiredAuthTokensStatement,
		c.UpsertAccessTokenStatement,
		c.GetAccessTokenStatement,
		c.DeleteExpiredAccessTokensStatement,
		c.SavePendingOAuthStateStatement,
		c.ValidatePendingOAuthStateStatement,
		c.DeletePendingOAuthStateStatement,
		c.DeleteExpiredPendingOAuthStatesStatement,
		c.ClaimAndGetExpiredTokenForRefreshStatement,
		c.UpdateParticipantStatement,
		c.GetActiveParticipantsStatement,
		c.GetActiveListenerCountsStatement,
		c.SetRoomHostStatement,
		c.RemoveParticipantStatement,
		c.DeleteInactiveParticipantsStatement,
		c.ElectNewHostStatement,
		c.GetActiveSourcesStatement,
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
