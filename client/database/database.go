// Package database contains a Postgres client for Vibes.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
)

// Client holds the database client and prepared statements.
type Client struct {
	DB *sql.DB

	maxNameLength                  int
	maxQueueLength                 int
	enabledProviders               []string
	roomNameReservationTTL         time.Duration
	roomGenerationMaxAttempts      int
	roomGenerationMaxDailyCount    int
	roomGenerationMaxExistingSongs int

	// Room statements
	GetRoomStatement                           *sql.Stmt
	GetRoomByNameStatement                     *sql.Stmt
	GetPublicRoomsStatement                    *sql.Stmt
	ReserveRoomNameStatement                   *sql.Stmt
	ReserveSuggestedRoomNameStatement          *sql.Stmt
	DeleteExpiredRoomNameReservationsStatement *sql.Stmt
	RoomExistsStatement                        *sql.Stmt
	CreateRoomStatement                        *sql.Stmt
	UpdateRoomStatement                        *sql.Stmt
	ProcessNextAbandonedHostStatement          *sql.Stmt

	// Room generation statements
	HasActiveRoomGenerationStatement      *sql.Stmt
	CreateRoomGenerationStatement         *sql.Stmt
	ProcessNextRoomGenerationStatement    *sql.Stmt
	CompleteRoomGenerationStatement       *sql.Stmt
	FailRoomGenerationStatement           *sql.Stmt
	DeleteExpiredRoomGenerationsStatement *sql.Stmt

	// Song statements
	GetSongsStatement                 *sql.Stmt
	GetSongStatement                  *sql.Stmt
	AddSongStatement                  *sql.Stmt
	RemoveSongStatement               *sql.Stmt
	VoteSongStatement                 *sql.Stmt
	ClearVotesSongStatement           *sql.Stmt
	UpdateSongAddedAtStatement        *sql.Stmt
	ClaimSongMetadataRefreshStatement *sql.Stmt
	RefreshSongMetadataStatement      *sql.Stmt
	DeferSongMetadataRefreshStatement *sql.Stmt

	// Playback statements
	GetPlaybackStateStatement           *sql.Stmt
	UpsertPlaybackStateStatement        *sql.Stmt
	SkipTrackStatement                  *sql.Stmt
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
	GetStatsStatement                   *sql.Stmt

	// Additional room statements
	GetActiveSourcesStatement *sql.Stmt
	GetAdminRoomsStatement    *sql.Stmt
	UpdateAdminRoomStatement  *sql.Stmt
	DeleteAdminRoomStatement  *sql.Stmt

	// Admin user statements
	GetAdminUserStatement            *sql.Stmt
	GetAdminUserByUsernameStatement  *sql.Stmt
	ListAdminUsersStatement          *sql.Stmt
	CreateAdminUserStatement         *sql.Stmt
	UpdateAdminUserPasswordStatement *sql.Stmt
	DeleteAdminUserStatement         *sql.Stmt

	// Search usage statements
	CreateSearchUsagesStatement   *sql.Stmt
	ListAdminSearchUsageStatement *sql.Stmt

	// Listener usage statements
	CreateListenerUsageStatement    *sql.Stmt
	ListAdminListenerUsageStatement *sql.Stmt
}

// Init sets up a new database client.
func (c *Client) Init(ctx context.Context, cfg *config.Config) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "Init")
	defer span.End()

	c.maxNameLength = cfg.MaxNameLength
	if c.maxNameLength == 0 {
		c.maxNameLength = 100
	}

	c.maxQueueLength = cfg.MaxQueueLength
	if c.maxQueueLength == 0 {
		c.maxQueueLength = 200
	}

	c.enabledProviders = cfg.EnabledProviders()
	c.roomNameReservationTTL = cfg.RoomNameReservationTTL
	c.roomGenerationMaxAttempts = cfg.RoomGenerationMaxAttempts
	c.roomGenerationMaxDailyCount = cfg.RoomGenerationMaxDailyCount
	c.roomGenerationMaxExistingSongs = cfg.RoomGenerationMaxExistingSongs
	if c.roomNameReservationTTL == 0 {
		c.roomNameReservationTTL = 2 * time.Minute
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
		c.prepareGetPublicRoomsStmt,
		c.prepareReserveRoomNameStmt,
		c.prepareReserveSuggestedRoomNameStmt,
		c.prepareDeleteExpiredRoomNameReservationsStmt,
		c.prepareRoomExistsStmt,
		c.prepareCreateRoomStmt,
		c.prepareUpdateRoomStmt,
		c.prepareProcessNextAbandonedHostStmt,
		c.prepareGetActiveSourcesStmt,
		c.prepareGetAdminRoomsStmt,
		c.prepareUpdateAdminRoomStmt,
		c.prepareDeleteAdminRoomStmt,
		// Admin user statements
		c.prepareGetAdminUserStmt,
		c.prepareGetAdminUserByUsernameStmt,
		c.prepareListAdminUsersStmt,
		c.prepareCreateAdminUserStmt,
		c.prepareUpdateAdminUserPasswordStmt,
		c.prepareDeleteAdminUserStmt,
		// Search usage statements
		c.prepareCreateSearchUsagesStmt,
		c.prepareListAdminSearchUsageStmt,
		// Listener usage statements
		c.prepareCreateListenerUsageStmt,
		c.prepareListAdminListenerUsageStmt,
		// Room generation statements
		c.prepareHasActiveRoomGenerationStmt,
		c.prepareCreateRoomGenerationStmt,
		c.prepareProcessNextRoomGenerationStmt,
		c.prepareCompleteRoomGenerationStmt,
		c.prepareFailRoomGenerationStmt,
		c.prepareDeleteExpiredRoomGenerationsStmt,
		// Song statements
		c.prepareGetSongsStmt,
		c.prepareGetSongStmt,
		c.prepareAddSongStmt,
		c.prepareRemoveSongStmt,
		c.prepareVoteSongStmt,
		c.prepareClearVotesSongStmt,
		c.prepareUpdateSongAddedAtStmt,
		c.prepareClaimSongMetadataRefreshStmt,
		c.prepareRefreshSongMetadataStmt,
		c.prepareDeferSongMetadataRefreshStmt,
		// Playback statements
		c.prepareGetPlaybackStateStmt,
		c.prepareUpsertPlaybackStateStmt,
		c.prepareSkipTrackStmt,
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
		c.prepareGetStatsStmt,
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
		c.GetPublicRoomsStatement,
		c.ReserveRoomNameStatement,
		c.ReserveSuggestedRoomNameStatement,
		c.DeleteExpiredRoomNameReservationsStatement,
		c.RoomExistsStatement,
		c.CreateRoomStatement,
		c.UpdateRoomStatement,
		c.ProcessNextAbandonedHostStatement,
		c.HasActiveRoomGenerationStatement,
		c.CreateRoomGenerationStatement,
		c.ProcessNextRoomGenerationStatement,
		c.CompleteRoomGenerationStatement,
		c.FailRoomGenerationStatement,
		c.DeleteExpiredRoomGenerationsStatement,
		c.GetSongsStatement,
		c.GetSongStatement,
		c.AddSongStatement,
		c.RemoveSongStatement,
		c.VoteSongStatement,
		c.ClearVotesSongStatement,
		c.UpdateSongAddedAtStatement,
		c.ClaimSongMetadataRefreshStatement,
		c.RefreshSongMetadataStatement,
		c.DeferSongMetadataRefreshStatement,
		c.GetPlaybackStateStatement,
		c.UpsertPlaybackStateStatement,
		c.SkipTrackStatement,
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
		c.GetStatsStatement,
		c.GetActiveSourcesStatement,
		c.GetAdminRoomsStatement,
		c.UpdateAdminRoomStatement,
		c.DeleteAdminRoomStatement,
		c.GetAdminUserStatement,
		c.GetAdminUserByUsernameStatement,
		c.ListAdminUsersStatement,
		c.CreateAdminUserStatement,
		c.UpdateAdminUserPasswordStatement,
		c.DeleteAdminUserStatement,
		c.CreateSearchUsagesStatement,
		c.ListAdminSearchUsageStatement,
		c.CreateListenerUsageStatement,
		c.ListAdminListenerUsageStatement,
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
