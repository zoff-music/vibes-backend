package soundcloud

import (
	"context"
	"time"

	"github.com/zoff-music/vibes/client"
	"github.com/zoff-music/vibes/config"
)

// Client implements vibe.MusicSearcher
type Client struct {
	Enabled      bool
	apiKey       string
	clientID     string
	clientSecret string
	redirectURI  string
	Endpoint     string
	HTTPClient   client.HTTPClient
}

// Init initializes the SoundCloud API client
func (c *Client) Init(ctx context.Context, cfg *config.Config) error {
	if cfg.SoundCloudAPIKey == "" || cfg.SoundCloudEndpoint == "" {
		c.Enabled = false
		return nil
	}
	c.Enabled = true
	c.apiKey = cfg.SoundCloudAPIKey
	c.Endpoint = cfg.SoundCloudEndpoint
	c.clientID = cfg.SoundCloudClientID
	c.clientSecret = cfg.SoundCloudClientSecret
	c.redirectURI = cfg.SoundCloudRedirectURI
	timeout := 5 * time.Second
	c.HTTPClient = client.NewHTTPClient(client.Parameters{Timeout: &timeout})
	return nil
}
