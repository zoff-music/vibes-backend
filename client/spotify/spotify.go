package spotify

import (
	"context"
	"sync"
	"time"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/config"
)

// Client implements vibe.MusicSearcher
type Client struct {
	Enabled      bool
	clientID     string
	clientSecret string
	Endpoint     string
	tokenURL     string
	redirectURI  string
	HTTPClient   client.HTTPClient

	mu          sync.RWMutex
	accessToken string
	expiresAt   time.Time
}

// Init initializes the Spotify API client
func (c *Client) Init(_ context.Context, cfg *config.Config) error {
	if cfg.SpotifyClientID == "" || cfg.SpotifyClientSecret == "" || cfg.SpotifyEndpoint == "" || cfg.SpotifyTokenURL == "" {
		c.Enabled = false
		return nil
	}
	c.Enabled = true
	c.clientID = cfg.SpotifyClientID
	c.clientSecret = cfg.SpotifyClientSecret
	c.Endpoint = cfg.SpotifyEndpoint
	c.tokenURL = cfg.SpotifyTokenURL
	c.redirectURI = cfg.SpotifyRedirectURI
	timeout := 5 * time.Second
	c.HTTPClient = client.NewHTTPClient(client.Parameters{Timeout: &timeout})
	return nil
}
