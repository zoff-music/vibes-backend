package vibe

import (
	"context"
	"time"
)

// TokenResponse represents the token response from a provider
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// ProviderTokenResponse represents the response containing the access token and its expiration sent to the frontend
type ProviderTokenResponse struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// TokenExchanger handles exchanging auth codes for tokens
type TokenExchanger interface {
	ExchangeCode(ctx context.Context, code string) (*TokenResponse, error)
}

// TokenRefresher handles refreshing access tokens
type TokenRefresher interface {
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
}

// OAuthProvider combines all OAuth capabilities
type OAuthProvider interface {
	OAuthAuthorizer
	TokenExchanger
	TokenRefresher
}
