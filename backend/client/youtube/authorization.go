package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes/monitoring/opentracing"

	"github.com/zoff-music/vibes/client"
	"github.com/zoff-music/vibes/vibe"
)

// GetOAuthURL returns the URL to redirect the user to for YouTube (Google) authentication
func (c *Client) GetOAuthURL(state, codeVerifier string) string {
	u, _ := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", c.clientID)
	q.Set("scope", "https://www.googleapis.com/auth/youtube.readonly")
	q.Set("redirect_uri", c.redirectURI)
	q.Set("state", state)
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	u.RawQuery = q.Encode()

	return u.String()
}

// ExchangeCode exchanges an authorization code for an access token
func (c *Client) ExchangeCode(ctx context.Context, code, codeVerifier string) (*vibe.TokenResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ExchangeCode")
	defer span.Finish()

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("client_id", c.clientID)
	params.Set("client_secret", c.clientSecret)
	params.Set("redirect_uri", c.redirectURI)
	params.Set("code", code)

	reqData := client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    "https://oauth2.googleapis.com/token",
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "RefreshToken")
	defer span.Finish()

	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("client_id", c.clientID)
	params.Set("client_secret", c.clientSecret)
	params.Set("refresh_token", refreshToken)

	reqData := client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    "https://oauth2.googleapis.com/token",
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
