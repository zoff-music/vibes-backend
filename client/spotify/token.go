package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getAccessToken")
	defer span.End()

	c.mu.RLock()
	if c.accessToken != "" && time.Now().Before(c.expiresAt) {
		token := c.accessToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-check after acquiring lock
	if c.accessToken != "" && time.Now().Before(c.expiresAt) {
		token := c.accessToken

		return token, nil
	}

	params := url.Values{}
	params.Set("grant_type", "client_credentials")

	credentials := base64.StdEncoding.EncodeToString(
		[]byte(c.clientID + ":" + c.clientSecret),
	)
	reqData := client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    c.tokenURL,
		Headers: map[string]string{
			"Accept":        "application/json",
			"Authorization": "Basic " + credentials,
			"Content-Type":  "application/x-www-form-urlencoded",
		},
		Body: []byte(params.Encode()),
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return "", fmt.Errorf(
			"error requesting client credentials token in getAccessToken: %w",
			err,
		)
	}

	var res tokenResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return "", fmt.Errorf(
			"error decoding token response in getAccessToken: %w",
			err,
		)
	}

	c.accessToken = res.AccessToken
	expiresIn := max(res.ExpiresIn-60, 1)
	c.expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	token := c.accessToken

	return token, nil
}

func (c *Client) ExchangeCode(ctx context.Context, code, codeVerifier string) (*vibe.TokenResponse, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ExchangeCode")
	defer span.End()

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("redirect_uri", c.redirectURI)

	credentials := base64.StdEncoding.EncodeToString(
		[]byte(c.clientID + ":" + c.clientSecret),
	)
	reqData := client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    c.tokenURL,
		Headers: map[string]string{
			"Accept":        "application/json",
			"Authorization": "Basic " + credentials,
			"Content-Type":  "application/x-www-form-urlencoded",
		},
		Body: []byte(params.Encode()),
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error requesting token in ExchangeCode: %w", err)
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

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*vibe.TokenResponse, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "RefreshToken")
	defer span.End()

	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)

	credentials := base64.StdEncoding.EncodeToString(
		[]byte(c.clientID + ":" + c.clientSecret),
	)
	reqData := client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    c.tokenURL,
		Headers: map[string]string{
			"Accept":        "application/json",
			"Authorization": "Basic " + credentials,
			"Content-Type":  "application/x-www-form-urlencoded",
		},
		Body: []byte(params.Encode()),
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error requesting token in RefreshToken: %w", err)
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

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
