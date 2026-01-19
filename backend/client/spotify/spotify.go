package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/zoff-music/vibes/client"
	"github.com/zoff-music/vibes/config"
)

// Client implements vibe.MusicSearcher
type Client struct {
	Enabled      bool
	clientID     string
	clientSecret string
	Endpoint     string
	tokenURL     string
	HTTPClient   client.HTTPClient

	mu          sync.RWMutex
	accessToken string
	expiresAt   time.Time
}

// Init initializes the Spotify API client
func (c *Client) Init(ctx context.Context, cfg *config.Config) error {
	if cfg.SpotifyClientID == "" || cfg.SpotifyClientSecret == "" || cfg.SpotifyEndpoint == "" || cfg.SpotifyTokenURL == "" {
		c.Enabled = false
		return nil
	}
	c.Enabled = true
	c.clientID = cfg.SpotifyClientID
	c.clientSecret = cfg.SpotifyClientSecret
	c.Endpoint = cfg.SpotifyEndpoint
	c.tokenURL = cfg.SpotifyTokenURL
	timeout := 5 * time.Second
	c.HTTPClient = client.NewHTTPClient(client.Parameters{Timeout: &timeout})
	return nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
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
