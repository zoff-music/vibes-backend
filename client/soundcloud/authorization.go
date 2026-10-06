package soundcloud

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetOAuthURL returns the URL to redirect the user to for SoundCloud authentication
func (c *Client) GetOAuthURL(state, codeVerifier string) string {
	u := url.URL{
		Scheme: "https",
		Host:   "secure.soundcloud.com",
		Path:   "/authorize",
	}
	q := u.Query()
	q.Set("client_id", c.clientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", c.redirectURI)
	q.Set("state", state)

	// PKCE
	if codeVerifier != "" {
		hash := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
		q.Set("code_challenge", codeChallenge)
		q.Set("code_challenge_method", "S256")
	}

	u.RawQuery = q.Encode()
	authorizationURL := u.String()

	return authorizationURL
}

// ExchangeCode exchanges an authorization code for an access token
func (c *Client) ExchangeCode(ctx context.Context, code, codeVerifier string) (*vibe.TokenResponse, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ExchangeCode")
	defer span.End()

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("client_id", c.clientID)
	params.Set("client_secret", c.clientSecret)
	params.Set("redirect_uri", c.redirectURI)
	params.Set("code", code)
	if codeVerifier != "" {
		params.Set("code_verifier", codeVerifier)
	}

	reqData := client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    soundCloudTokenURL,
		Headers: map[string]string{
			"Accept":       "application/json; charset=utf-8",
			"Content-Type": "application/x-www-form-urlencoded",
		},
		Body: []byte(params.Encode()),
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error exchanging code in ExchangeCode: %w", err)
	}

	var res vibe.TokenResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding token response in ExchangeCode: %w",
			err,
		)
	}

	return &res, nil
}

// RefreshToken refreshes the access token
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*vibe.TokenResponse, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "RefreshToken")
	defer span.End()

	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("client_id", c.clientID)
	params.Set("client_secret", c.clientSecret)
	params.Set("refresh_token", refreshToken)

	reqData := client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    soundCloudTokenURL,
		Headers: map[string]string{
			"Accept":       "application/json; charset=utf-8",
			"Content-Type": "application/x-www-form-urlencoded",
		},
		Body: []byte(params.Encode()),
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error refreshing token in RefreshToken: %w", err)
	}

	var res vibe.TokenResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding token response in RefreshToken: %w",
			err,
		)
	}

	return &res, nil
}

// #nosec G101 -- this is SoundCloud's public OAuth endpoint, not a credential.
const soundCloudTokenURL = "https://secure.soundcloud.com/oauth/token"

// EnsureToken checks if the current token is valid and refreshes it if necessary
func (c *Client) EnsureToken(ctx context.Context) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "EnsureToken")
	defer span.End()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if token is valid (with 1 minute buffer)
	if c.accessToken != "" && time.Now().Add(1*time.Minute).Before(c.tokenExpiresAt) {
		return nil
	}

	token, err := c.GetClientCredentialsToken(ctx)
	if err != nil {
		return fmt.Errorf(
			"error getting client credentials token in EnsureToken: %w",
			err,
		)
	}

	c.accessToken = token.AccessToken
	// Default to 1 hour if not specified, typically SoundCloud returns 3600 seconds
	expiresIn := token.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 3600
	}
	c.tokenExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)

	return nil
}

// GetClientCredentialsToken authenticates using client_credentials grant type
func (c *Client) GetClientCredentialsToken(ctx context.Context) (*vibe.TokenResponse, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetClientCredentialsToken")
	defer span.End()

	params := url.Values{}
	params.Set("grant_type", "client_credentials")
	credentials := base64.StdEncoding.EncodeToString(
		[]byte(c.clientID + ":" + c.clientSecret),
	)

	reqData := client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    soundCloudTokenURL,
		Headers: map[string]string{
			"Accept":        "application/json; charset=utf-8",
			"Authorization": "Basic " + credentials,
			"Content-Type":  "application/x-www-form-urlencoded",
		},
		Body: []byte(params.Encode()),
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf(
			"error requesting token in GetClientCredentialsToken: %w",
			err,
		)
	}

	var res vibe.TokenResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding token response in GetClientCredentialsToken: %w",
			err,
		)
	}

	return &res, nil
}
