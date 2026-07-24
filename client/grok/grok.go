package grok

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
)

type Client struct {
	Enabled    bool
	Endpoint   string
	Model      string
	apiKey     string
	HTTPClient client.HTTPClient
}

func (c *Client) Init(ctx context.Context, cfg *config.Config) error {
	span, _ := tracing.StartSpanFromContext(ctx, "Init")
	defer span.End()

	c.Endpoint = strings.TrimRight(cfg.GrokEndpoint, "/")
	c.Model = cfg.AIModel
	c.apiKey = cfg.GrokAPIKey
	c.Enabled = c.apiKey != ""

	if c.Endpoint == "" {
		return fmt.Errorf("error grok endpoint is required")
	}
	if c.Model == "" {
		return fmt.Errorf("error grok model is required")
	}

	c.HTTPClient = client.HTTPClient{
		Client: &http.Client{
			Timeout:   2 * time.Minute,
			Transport: client.InstrumentedTransport(),
		},
	}

	return nil
}
