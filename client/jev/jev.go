package jev

import (
	"context"
	"fmt"
	"math"
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
	threshold  float64
	HTTPClient client.HTTPClient
}

func (c *Client) Init(ctx context.Context, cfg *config.Config) error {
	span, _ := tracing.StartSpanFromContext(ctx, "Init")
	defer span.End()

	c.Endpoint = strings.TrimRight(cfg.JEVEndpoint, "/")
	c.Model = strings.TrimSpace(cfg.ModerationModel)
	c.apiKey = cfg.JEVAPIKey
	if c.Endpoint == "" {
		return fmt.Errorf("error JEV endpoint is required")
	}
	if c.Model == "" || strings.Contains(c.Model, ":") {
		return fmt.Errorf("error validating moderation model in Init: an unprefixed model name is required")
	}
	if c.apiKey == "" {
		return fmt.Errorf("error JEV API key is required")
	}
	if math.IsNaN(cfg.JEVModerationThreshold) || math.IsInf(cfg.JEVModerationThreshold, 0) || cfg.JEVModerationThreshold <= 0 || cfg.JEVModerationThreshold > 100 {
		return fmt.Errorf("error validating JEV moderation threshold in Init: percentage must be greater than zero and at most 100")
	}

	c.threshold = cfg.JEVModerationThreshold / 100
	c.HTTPClient = client.HTTPClient{
		Client: &http.Client{
			Timeout:   2 * time.Minute,
			Transport: client.InstrumentedTransport(),
		},
	}
	c.Enabled = true

	return nil
}
