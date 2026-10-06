// Package database contains a Postgres client for Vibes.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
)

// Client holds the database client and prepared statements.
type Client struct {
	DB *sql.DB

	maxNameLength                          int
	maxQueueLength                         int
	enabledProviders                       []string
	roomNameReservationTTL                 time.Duration
	roomGenerationMaxAttempts              int
	roomGenerationMaxDailyCount            int
	roomGenerationMaxExistingPlaylistItems int

	// Room statements
	GetRoomV2Statement                         *sql.Stmt
	GetRoomByNameV2Statement                   *sql.Stmt
	GetPublicRoomsStatement                    *sql.Stmt
	SearchPublicRoomsV3Statement               *sql.Stmt
	ReserveRoomNameStatement                   *sql.Stmt
	ReserveSuggestedRoomNameStatement          *sql.Stmt
	DeleteExpiredRoomNameReservationsStatement *sql.Stmt
	RoomExistsStatement                        *sql.Stmt
	CreateRoomV2Statement                      *sql.Stmt
	UpdateRoomV2Statement                      *sql.Stmt
	ProcessNextAbandonedHostStatement          *sql.Stmt

	// Room generation statements
	HasActiveRoomGenerationStatement      *sql.Stmt
	CreateRoomGenerationStatement         *sql.Stmt
	ProcessNextRoomGenerationStatement    *sql.Stmt
	CompleteRoomGenerationStatement       *sql.Stmt
	FailRoomGenerationStatement           *sql.Stmt
	DeleteExpiredRoomGenerationsStatement *sql.Stmt

	// PlaylistItem statements
	GetPlaylistItemsStatement                      *sql.Stmt
	GetPlaylistItemStatement                       *sql.Stmt
	AddPlaylistItemStatement                       *sql.Stmt
	RemovePlaylistItemStatement                    *sql.Stmt
	VotePlaylistItemStatement                      *sql.Stmt
	ClearVotesPlaylistItemStatement                *sql.Stmt
	UpdatePlaylistItemAddedAtStatement             *sql.Stmt
	ClaimPlaylistItemMetadataRefreshStatement      *sql.Stmt
	RefreshPlaylistItemMetadataStatement           *sql.Stmt
	DeferPlaylistItemMetadataRefreshStatement      *sql.Stmt
	ExpirePlaylistItemMetadataStatement            *sql.Stmt
	UpdatePlaylistItemPlaybackRestrictionStatement *sql.Stmt

	// Playlist import statements
	CreatePlaylistImportStatement               *sql.Stmt
	CreatePlaylistImportItemStatement           *sql.Stmt
	DeleteAbandonedPlaylistImportItemsStatement *sql.Stmt
	ProcessNextPlaylistImportStatement          *sql.Stmt
	CompletePlaylistImportItemStatement         *sql.Stmt
	DeletePlaylistImportStatement               *sql.Stmt

	// Playback statements
	GetPlaybackStateV2Statement           *sql.Stmt
	UpsertPlaybackStateV2Statement        *sql.Stmt
	SkipPlaylistItemStatement             *sql.Stmt
	ProcessNextExpiredPlaybackV2Statement *sql.Stmt
	StartPlaybackIfIdleV2Statement        *sql.Stmt

	// User statements
	GetUserStatement        *sql.Stmt
	CreateUserStatement     *sql.Stmt
	ClearRoomAdminStatement *sql.Stmt

	// Session statements
	GetSessionRoomsStatement           *sql.Stmt
	GetOrCreateSessionProfileStatement *sql.Stmt
	UpdateSessionProfileStatement      *sql.Stmt

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
	ConsumePendingOAuthStateStatement        *sql.Stmt
	DeleteExpiredPendingOAuthStatesStatement *sql.Stmt

	// Token cleanup statements
	ClaimAndGetExpiredTokenForRefreshStatement *sql.Stmt

	// Participant statements
	UpdateParticipantStatement         *sql.Stmt
	GetActiveParticipantsStatement     *sql.Stmt
	GetActiveListenerCountsStatement   *sql.Stmt
	SetRoomHostStatement               *sql.Stmt
	RemoveParticipantStatement         *sql.Stmt
	CleanInactiveParticipantsStatement *sql.Stmt
	GetStatsV2Statement                *sql.Stmt

	// Additional room statements
	GetActiveSourcesStatement   *sql.Stmt
	SearchAdminRoomsV2Statement *sql.Stmt
	UpdateAdminRoomStatement    *sql.Stmt
	DeleteAdminRoomStatement    *sql.Stmt

	// Admin user statements
	GetAdminUserStatement            *sql.Stmt
	GetAdminUserByUsernameStatement  *sql.Stmt
	ListAdminUsersStatement          *sql.Stmt
	CreateAdminUserStatement         *sql.Stmt
	UpdateAdminUserPasswordStatement *sql.Stmt
	DeleteAdminUserStatement         *sql.Stmt

	// Search usage statements
	CreateSearchUsagesStatement       *sql.Stmt
	ListAdminSearchUsageStatement     *sql.Stmt
	ListAdminRoomSearchUsageStatement *sql.Stmt

	// Message usage statements
	CreateMessageUsageStatement    *sql.Stmt
	ListAdminMessageUsageStatement *sql.Stmt

	// Listener usage statements
	CreateListenerUsageStatement        *sql.Stmt
	ListAdminListenerUsageStatement     *sql.Stmt
	ListAdminRoomListenerUsageStatement *sql.Stmt

	// Remote control statements
	CreateRemoteControlV2Statement       *sql.Stmt
	GetRemoteControlByOwnerV2Statement   *sql.Stmt
	GetRemoteControlV2Statement          *sql.Stmt
	PairRemoteControlV2Statement         *sql.Stmt
	AuthenticateRemoteControlV2Statement *sql.Stmt
	UpdateOwnedRemoteControlV2Statement  *sql.Stmt
	UpdatePairedRemoteControlV2Statement *sql.Stmt
	DeleteRemoteControlStatement         *sql.Stmt
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
	c.roomGenerationMaxExistingPlaylistItems = cfg.RoomGenerationMaxExistingPlaylistItems
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
		c.prepareGetRoomV2Stmt,
		c.prepareGetRoomByNameV2Stmt,
		c.prepareGetPublicRoomsStmt,
		c.prepareSearchPublicRoomsV3Stmt,
		c.prepareReserveRoomNameStmt,
		c.prepareReserveSuggestedRoomNameStmt,
		c.prepareDeleteExpiredRoomNameReservationsStmt,
		c.prepareRoomExistsStmt,
		c.prepareCreateRoomV2Stmt,
		c.prepareUpdateRoomV2Stmt,
		c.prepareProcessNextAbandonedHostStmt,
		c.prepareGetActiveSourcesStmt,
		c.prepareSearchAdminRoomsV2Stmt,
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
		c.prepareListAdminRoomSearchUsageStmt,
		// Message usage statements
		c.prepareCreateMessageUsageStmt,
		c.prepareListAdminMessageUsageStmt,
		// Listener usage statements
		c.prepareCreateListenerUsageStmt,
		c.prepareListAdminListenerUsageStmt,
		c.prepareListAdminRoomListenerUsageStmt,
		// Room generation statements
		c.prepareHasActiveRoomGenerationStmt,
		c.prepareCreateRoomGenerationStmt,
		c.prepareProcessNextRoomGenerationStmt,
		c.prepareCompleteRoomGenerationStmt,
		c.prepareFailRoomGenerationStmt,
		c.prepareDeleteExpiredRoomGenerationsStmt,
		// PlaylistItem statements
		c.prepareGetPlaylistItemsStmt,
		c.prepareGetPlaylistItemStmt,
		c.prepareAddPlaylistItemStmt,
		c.prepareRemovePlaylistItemStmt,
		c.prepareVotePlaylistItemStmt,
		c.prepareClearVotesPlaylistItemStmt,
		c.prepareUpdatePlaylistItemAddedAtStmt,
		c.prepareClaimPlaylistItemMetadataRefreshStmt,
		c.prepareRefreshPlaylistItemMetadataStmt,
		c.prepareDeferPlaylistItemMetadataRefreshStmt,
		c.prepareExpirePlaylistItemMetadataStmt,
		c.prepareUpdatePlaylistItemPlaybackRestrictionStmt,
		// Playlist import statements
		c.prepareCreatePlaylistImportStmt,
		c.prepareCreatePlaylistImportItemStmt,
		c.prepareDeleteAbandonedPlaylistImportItemsStmt,
		c.prepareProcessNextPlaylistImportStmt,
		c.prepareCompletePlaylistImportItemStmt,
		c.prepareDeletePlaylistImportStmt,
		// Playback statements
		c.prepareGetPlaybackStateV2Stmt,
		c.prepareUpsertPlaybackStateV2Stmt,
		c.prepareSkipPlaylistItemStmt,
		c.prepareProcessNextExpiredPlaybackV2Stmt,
		c.prepareStartPlaybackIfIdleV2Stmt,
		// User statements
		c.prepareGetUserStmt,
		c.prepareCreateUserStmt,
		c.prepareClearRoomAdminStmt,
		// Session statements
		c.prepareGetSessionRoomsStmt,
		c.prepareGetOrCreateSessionProfileStmt,
		c.prepareUpdateSessionProfileStmt,
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
		c.prepareConsumePendingOAuthStateStmt,
		c.prepareDeleteExpiredPendingOAuthStatesStmt,
		c.prepareClaimAndGetExpiredTokenForRefreshStmt,
		// Participant statements
		c.prepareUpdateParticipantStmt,
		c.prepareGetActiveParticipantsStmt,
		c.prepareGetActiveListenerCountsStmt,
		c.prepareSetRoomHostStmt,
		c.prepareRemoveParticipantStmt,
		c.prepareCleanInactiveParticipantsStmt,
		c.prepareGetStatsV2Stmt,
		// Remote control statements
		c.prepareCreateRemoteControlV2Stmt,
		c.prepareGetRemoteControlByOwnerV2Stmt,
		c.prepareGetRemoteControlV2Stmt,
		c.preparePairRemoteControlV2Stmt,
		c.prepareAuthenticateRemoteControlV2Stmt,
		c.prepareUpdateOwnedRemoteControlV2Stmt,
		c.prepareUpdatePairedRemoteControlV2Stmt,
		c.prepareDeleteRemoteControlStmt,
	}

	for _, prepareStmt := range prepareStatements {
		err := prepareStmt()
		if err != nil {
			closeErr := c.Close()
			if closeErr != nil {
				err = errors.Join(err, closeErr)
			}

			return fmt.Errorf("error preparing statements: %w", err)
		}
	}

	return nil
}

// Close closes the database connection and statements.
func (c *Client) Close() error {
	statements := []*sql.Stmt{
		c.GetRoomV2Statement,
		c.GetRoomByNameV2Statement,
		c.GetPublicRoomsStatement,
		c.SearchPublicRoomsV3Statement,
		c.ReserveRoomNameStatement,
		c.ReserveSuggestedRoomNameStatement,
		c.DeleteExpiredRoomNameReservationsStatement,
		c.RoomExistsStatement,
		c.CreateRoomV2Statement,
		c.UpdateRoomV2Statement,
		c.ProcessNextAbandonedHostStatement,
		c.HasActiveRoomGenerationStatement,
		c.CreateRoomGenerationStatement,
		c.ProcessNextRoomGenerationStatement,
		c.CompleteRoomGenerationStatement,
		c.FailRoomGenerationStatement,
		c.DeleteExpiredRoomGenerationsStatement,
		c.GetPlaylistItemsStatement,
		c.GetPlaylistItemStatement,
		c.AddPlaylistItemStatement,
		c.RemovePlaylistItemStatement,
		c.VotePlaylistItemStatement,
		c.ClearVotesPlaylistItemStatement,
		c.UpdatePlaylistItemAddedAtStatement,
		c.UpdatePlaylistItemPlaybackRestrictionStatement,
		c.ClaimPlaylistItemMetadataRefreshStatement,
		c.RefreshPlaylistItemMetadataStatement,
		c.DeferPlaylistItemMetadataRefreshStatement,
		c.ExpirePlaylistItemMetadataStatement,
		c.CreatePlaylistImportStatement,
		c.CreatePlaylistImportItemStatement,
		c.DeleteAbandonedPlaylistImportItemsStatement,
		c.ProcessNextPlaylistImportStatement,
		c.CompletePlaylistImportItemStatement,
		c.DeletePlaylistImportStatement,
		c.GetPlaybackStateV2Statement,
		c.UpsertPlaybackStateV2Statement,
		c.SkipPlaylistItemStatement,
		c.ProcessNextExpiredPlaybackV2Statement,
		c.StartPlaybackIfIdleV2Statement,
		c.GetUserStatement,
		c.CreateUserStatement,
		c.ClearRoomAdminStatement,
		c.GetSessionRoomsStatement,
		c.GetOrCreateSessionProfileStatement,
		c.UpdateSessionProfileStatement,
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
		c.ConsumePendingOAuthStateStatement,
		c.DeleteExpiredPendingOAuthStatesStatement,
		c.ClaimAndGetExpiredTokenForRefreshStatement,
		c.UpdateParticipantStatement,
		c.GetActiveParticipantsStatement,
		c.GetActiveListenerCountsStatement,
		c.SetRoomHostStatement,
		c.RemoveParticipantStatement,
		c.CleanInactiveParticipantsStatement,
		c.GetStatsV2Statement,
		c.GetActiveSourcesStatement,
		c.SearchAdminRoomsV2Statement,
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
		c.ListAdminRoomSearchUsageStatement,
		c.CreateMessageUsageStatement,
		c.ListAdminMessageUsageStatement,
		c.CreateListenerUsageStatement,
		c.ListAdminListenerUsageStatement,
		c.ListAdminRoomListenerUsageStatement,
		c.CreateRemoteControlV2Statement,
		c.GetRemoteControlByOwnerV2Statement,
		c.GetRemoteControlV2Statement,
		c.PairRemoteControlV2Statement,
		c.AuthenticateRemoteControlV2Statement,
		c.UpdateOwnedRemoteControlV2Statement,
		c.UpdatePairedRemoteControlV2Statement,
		c.DeleteRemoteControlStatement,
	}

	var closeErrors []error

	for _, stmt := range statements {
		if stmt == nil {
			continue
		}

		err := stmt.Close()
		if err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("error closing database statement: %w", err))
		}
	}

	if c.DB != nil {
		err := c.DB.Close()
		if err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("error closing database connection: %w", err))
		}
	}

	err := errors.Join(closeErrors...)
	if err != nil {
		return fmt.Errorf("error closing database client: %w", err)
	}

	return nil
}
