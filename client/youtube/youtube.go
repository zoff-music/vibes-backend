package youtube

import (
	"context"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/config"
)

// Client implements vibe.MusicSearcher
type Client struct {
	apiKey       string
	clientID     string
	clientSecret string
	redirectURI  string
	Endpoint     string
	HTTPClient   client.HTTPClient
}

// Init initializes the YouTube API client
func (c *Client) Init(_ context.Context, cfg *config.Config) error {
	if cfg.YouTubeAPIKey == "" {
		return fmt.Errorf("error youtube api key is required")
	}
	if cfg.YouTubeEndpoint == "" {
		return fmt.Errorf("error youtube endpoint is required")
	}
	c.apiKey = cfg.YouTubeAPIKey
	c.Endpoint = cfg.YouTubeEndpoint
	c.clientID = cfg.YouTubeClientID
	c.clientSecret = cfg.YouTubeClientSecret
	c.redirectURI = cfg.YouTubeRedirectURI
	timeout := 5 * time.Second
	c.HTTPClient = client.NewHTTPClient(client.Parameters{
		Timeout: &timeout,
	})
	return nil
}
