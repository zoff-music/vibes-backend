package vibe

import (
	"context"
	"time"
)

// OAuthAuthorizer handles OAuth authorization URL generation.
type OAuthAuthorizer interface {
	GetOAuthURL(state string) string
}

// AuthTokenUpserter handles storing auth token data (code/state).
type AuthTokenUpserter interface {
	UpsertAuthToken(ctx context.Context, userID, provider, code, state string, expiresAt time.Time) error
}

// AccessTokenUpserter handles storing access token data.
type AccessTokenUpserter interface {
	UpsertAccessToken(ctx context.Context, userID, provider, accessToken, refreshToken string, expiresAt, refreshExpiresAt time.Time) error
}

// AuthTokenLister handles listing auth providers.
type AuthTokenLister interface {
	GetAuthProviders(ctx context.Context, userID string) ([]string, error)
}

// AccessTokenGetter handles retrieving access token data.
type AccessTokenGetter interface {
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
	SavePendingOAuthState(ctx context.Context, userID, state string) error
}

// OAuthCallbackDB handles database operations for OAuth callbacks
type OAuthCallbackDB interface {
	ValidateAndDeletePendingOAuthState(ctx context.Context, state string) (string, error)
	UpsertAuthToken(ctx context.Context, userID, provider, code, state string, expiresAt time.Time) error
	UpsertAccessToken(ctx context.Context, userID, provider, accessToken, refreshToken string, expiresAt, refreshExpiresAt time.Time) error
}
