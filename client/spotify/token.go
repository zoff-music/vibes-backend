package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/zoff-music/vibes/client"
	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"
)

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "getAccessToken")
	defer span.Finish()

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
		return c.accessToken, nil
	}

	params := url.Values{}
	params.Set("grant_type", "client_credentials")

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.clientID, c.clientSecret)))
	reqData := client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    c.tokenURL,
		Headers: map[string]string{
			"Authorization": "Basic " + auth,
			"Content-Type":  "application/x-www-form-urlencoded",
		},
		Body: []byte(params.Encode()),
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return "", fmt.Errorf("error token request failed: %w", err)
	}

	var res tokenResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return "", fmt.Errorf("error decoding token response: %w", err)
	}

	c.accessToken = res.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(res.ExpiresIn-60) * time.Second)

	return c.accessToken, nil
}

func (c *Client) ExchangeCode(ctx context.Context, code, codeVerifier string) (*vibe.TokenResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ExchangeCode")
	defer span.Finish()

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("redirect_uri", c.redirectURI)

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.clientID, c.clientSecret)))
	reqData := client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    c.tokenURL,
		Headers: map[string]string{
			"Authorization": "Basic " + auth,
			"Content-Type":  "application/x-www-form-urlencoded",
		},
		Body: []byte(params.Encode()),
	}

	log.Printf("code: %s", code)
	log.Printf("auth: %s", auth)
	log.Printf("url: %s", c.tokenURL)
	log.Printf("body: %s", string(reqData.Body))

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error token request failed: %w", err)
	}

	var res vibe.TokenResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf("error decoding token response: %w", err)
	}

	return &res, nil
}

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*vibe.TokenResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RefreshToken")
	defer span.Finish()

	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.clientID, c.clientSecret)))
	reqData := client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    c.tokenURL,
		Headers: map[string]string{
			"Authorization": "Basic " + auth,
			"Content-Type":  "application/x-www-form-urlencoded",
		},
		Body: []byte(params.Encode()),
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error token request failed: %w", err)
	}

	var res vibe.TokenResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf("error decoding token response: %w", err)
	}

	return &res, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
