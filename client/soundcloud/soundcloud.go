package soundcloud

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
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
func (c *Client) Init(ctx context.Context, cfg *config.Config) error {
	span, _ := tracing.StartSpanFromContext(ctx, "Init")
	defer span.End()

	if cfg.SoundCloudClientID == "" || cfg.SoundCloudEndpoint == "" {
		c.Enabled = false
		return nil
	}
	c.Enabled = true
	c.Endpoint = cfg.SoundCloudEndpoint
	c.clientID = cfg.SoundCloudClientID
	c.clientSecret = cfg.SoundCloudClientSecret
	c.redirectURI = cfg.SoundCloudRedirectURI
	c.HTTPClient = client.HTTPClient{
		Client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: client.InstrumentedTransport(),
		},
	}
	return nil
}
