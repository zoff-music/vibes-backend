package soundcloud

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetOAuthURL returns the URL to redirect the user to for SoundCloud authentication
func (c *Client) GetOAuthURL(state, codeVerifier string) string {
	u, _ := url.Parse("https://soundcloud.com/connect")
	q := u.Query()
	q.Set("client_id", c.clientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", c.redirectURI)
	q.Set("state", state)

	// PKCE
	if codeVerifier != "" {
		h := sha256.New()
		h.Write([]byte(codeVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
		q.Set("code_challenge", codeChallenge)
		q.Set("code_challenge_method", "S256")
	}

	u.RawQuery = q.Encode()

	return u.String()
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
		URL:    "https://secure.soundcloud.com/oauth/token",
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		Body: []byte(params.Encode()),
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error exchanging code for token: %w", err)
	}

	var res vibe.TokenResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf("error decoding token response: %w", err)
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
		URL:    "https://secure.soundcloud.com/oauth/token",
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		Body: []byte(params.Encode()),
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error refreshing token: %w", err)
	}

	var res vibe.TokenResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf("error decoding token response: %w", err)
	}

	return &res, nil
}
