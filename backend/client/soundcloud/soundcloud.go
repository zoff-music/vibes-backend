package soundcloud

import (
	"context"
	"sync"
	"time"

	"github.com/zoff-music/vibes/client"
	"github.com/zoff-music/vibes/config"
)

// Client implements vibe.MusicSearcher
type Client struct {
	Enabled        bool
	clientID       string
	clientSecret   string
	redirectURI    string
	Endpoint       string
	HTTPClient     client.HTTPClient
	accessToken    string
	tokenExpiresAt time.Time
	mu             sync.Mutex
}

// Init initializes the SoundCloud API client
func (c *Client) Init(_ context.Context, cfg *config.Config) error {
	if cfg.SoundCloudClientID == "" || cfg.SoundCloudEndpoint == "" {
		c.Enabled = false
		return nil
	}
	c.Enabled = true
	c.Endpoint = cfg.SoundCloudEndpoint
	c.clientID = cfg.SoundCloudClientID
	c.clientSecret = cfg.SoundCloudClientSecret
	c.redirectURI = cfg.SoundCloudRedirectURI
	timeout := 5 * time.Second
	c.HTTPClient = client.NewHTTPClient(client.Parameters{Timeout: &timeout})
	return nil
}
