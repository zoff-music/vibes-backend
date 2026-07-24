package soundcloud

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
