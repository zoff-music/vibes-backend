package vibe

import (
	"context"
	"time"
)

// OAuthAuthorizer handles OAuth authorization URL generation.
type OAuthAuthorizer interface {
	GetOAuthURL(state string) string
}

// ExternalAuthUpserter handles storing external authentication data.
type ExternalAuthUpserter interface {
	UpsertExternalAuth(ctx context.Context, userID, provider, code, state string, expiresAt time.Time) error
}

// ExternalAuthLister handles listing external authentication data.
type ExternalAuthLister interface {
	GetExternalAuths(ctx context.Context, userID string) ([]string, error)
}

// ExternalAuthGetter handles retrieving external authentication data.
type ExternalAuthGetter interface {
	GetExternalAuth(ctx context.Context, userID, provider string) (*ExternalAuth, error)
}

// ExternalAuthStorage handles storing external authentication data.
// Deprecated: Use ExternalAuthUpserter, ExternalAuthLister, or ExternalAuthGetter instead.
type ExternalAuthStorage interface {
	ExternalAuthUpserter
	ExternalAuthLister
	ExternalAuthGetter
}

// ExternalAuth represents a user's authorization with an external provider
type ExternalAuth struct {
	UserID    string    `json:"userId"`
	Provider  string    `json:"provider"`
	Code      string    `json:"-"`
	State     string    `json:"-"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// TokenResponse represents the token response from a provider
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// TokenExchanger handles exchanging auth codes for tokens
type TokenExchanger interface {
	ExchangeCode(ctx context.Context, code string) (*TokenResponse, error)
}
