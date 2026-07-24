package youtube

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
)

// Client implements vibe.MusicSearcher
type Client struct {
	apiKey       string
	clientID     string
	clientSecret string
	redirectURI  string
	Endpoint     string
	HTTPClient   client.HTTPClient

	searchQuotaMu    sync.RWMutex
	searchQuotaZone  *time.Location
	searchQuotaReset time.Time
}

// Init initializes the YouTube API client
func (c *Client) Init(ctx context.Context, cfg *config.Config) error {
	span, _ := tracing.StartSpanFromContext(ctx, "Init")
	defer span.End()

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
	searchQuotaZone, err := time.LoadLocation(youtubeQuotaLocation)
	if err != nil {
		return fmt.Errorf("error loading youtube quota location: %w", err)
	}
	c.searchQuotaZone = searchQuotaZone
	c.HTTPClient = client.HTTPClient{
		Client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: client.InstrumentedTransport(),
		},
	}
	return nil
}
