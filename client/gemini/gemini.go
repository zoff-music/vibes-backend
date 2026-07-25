package gemini

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
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

	aiModel, err := vibe.ParseAIModel(cfg.AIModel)
	if err != nil {
		return fmt.Errorf("error parsing configured AI model in Init: %w", err)
	}
	if aiModel.Provider != vibe.AIProviderGemini {
		c.Enabled = false
		return nil
	}

	c.Endpoint = strings.TrimRight(cfg.GeminiEndpoint, "/")
	c.Model = aiModel.Name
	c.apiKey = cfg.GeminiAPIKey
	if c.Endpoint == "" {
		return fmt.Errorf("error gemini endpoint is required")
	}
	if c.apiKey == "" {
		return fmt.Errorf("error gemini API key is required")
	}
	c.Enabled = true

	c.HTTPClient = client.HTTPClient{
		Client: &http.Client{
			Timeout:   2 * time.Minute,
			Transport: client.InstrumentedTransport(),
		},
	}

	return nil
}
