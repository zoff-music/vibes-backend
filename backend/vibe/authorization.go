package vibe

import (
	"context"
	"time"
)

// OAuthAuthorizer handles OAuth authorization URL generation.
type OAuthAuthorizer interface {
	GetOAuthURL(state, codeVerifier string) string
}

// AccessTokenUpserterGetter handles storing and retrieving access token data.
type AccessTokenUpserterGetter interface {
	UpsertAccessToken(ctx context.Context, userID, provider, accessToken, refreshToken string, expiresAt, refreshExpiresAt time.Time) error
	GetAccessToken(ctx context.Context, userID, provider string) (*AccessToken, error)
}

// AuthTokenCleaner handles cleaning up expired auth and access tokens.
type AuthTokenCleaner interface {
	DeleteExpiredAuthTokens(ctx context.Context) (int64, error)
	DeleteExpiredAccessTokens(ctx context.Context) (int64, error)
}

// AuthToken represents a user's initial auth code/state
type AuthToken struct {
	UserID    string    `json:"userId"`
	Provider  string    `json:"provider"`
	Code      string    `json:"-"`
	State     string    `json:"-"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// AccessToken represents a user's OAuth tokens
type AccessToken struct {
	UserID           string    `json:"userId"`
	Provider         string    `json:"provider"`
	AccessToken      string    `json:"-"`
	RefreshToken     string    `json:"-"`
	ExpiresAt        time.Time `json:"expiresAt"`
	RefreshExpiresAt time.Time `json:"-"`
}

// PendingOAuthStateSaver handles saving pending OAuth state
type PendingOAuthStateSaver interface {
	SavePendingOAuthState(ctx context.Context, userID, state, codeVerifier string) error
}

// CodeValidatorUpserter handles database operations for OAuth callbacks
type CodeValidatorUpserter interface {
	ValidateAndDeletePendingOAuthState(ctx context.Context, state string) (string, string, error)
	UpsertAuthToken(ctx context.Context, userID, provider, code, state string, expiresAt time.Time) error
	UpsertAccessToken(ctx context.Context, userID, provider, accessToken, refreshToken string, expiresAt, refreshExpiresAt time.Time) error
}

// OAuthExchanger handles exchanging auth codes for tokens
type OAuthExchanger interface {
	ExchangeCode(ctx context.Context, code, codeVerifier string) (*TokenResponse, error)
}

// ExpiredTokenClaimUpdater claims and returns an expired token for refresh
type ExpiredTokenClaimUpdater interface {
	ClaimAndGetExpiredTokenForRefresh(ctx context.Context, provider string) (*AccessToken, error)
	UpsertAccessToken(ctx context.Context, userID, provider, accessToken, refreshToken string, expiresAt, refreshExpiresAt time.Time) error
}

// ExpiredPendingOAuthStateCleaner deletes expired pending OAuth states
type ExpiredPendingOAuthStateCleaner interface {
	DeleteExpiredPendingOAuthStates(ctx context.Context) (int64, error)
}
