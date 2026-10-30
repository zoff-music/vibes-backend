package plausible

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

type Client struct {
	Endpoint   string
	HTTPClient client.HTTPClient
}

func (c *Client) Init(ctx context.Context, cfg *config.Config) error {
	span, _ := tracing.StartSpanFromContext(ctx, "Init")
	defer span.End()
	endpoint, err := url.Parse(cfg.PlausibleEndpoint)
	if err != nil {
		return fmt.Errorf("error parsing Plausible endpoint: %w", err)
	}
	if endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" {
		return fmt.Errorf("error validating Plausible endpoint")
	}
	c.Endpoint = endpoint.String()
	c.HTTPClient = client.HTTPClient{Client: &http.Client{
		Timeout:   5 * time.Second,
		Transport: client.InstrumentedTransport(),
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
	return nil
}

func (c *Client) CreateAnalyticsEvent(ctx context.Context, event vibe.AnalyticsEvent, visitor vibe.AnalyticsVisitor) (*vibe.AnalyticsResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateAnalyticsEvent")
	defer span.End()
	body, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("error marshaling analytics event: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating analytics request: %w", err)
	}
	// Only forward the headers needed for aggregate visitor counting, never credentials.
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", visitor.UserAgent)
	request.Header.Set("X-Forwarded-For", visitor.IP)
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error forwarding analytics event: %w", err)
	}
	defer response.Body.Close()
	_, err = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if err != nil {
		return nil, fmt.Errorf("error reading analytics response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("error forwarding analytics event: upstream status %d", response.StatusCode)
	}
	return &vibe.AnalyticsResult{Dropped: response.Header.Get("X-Plausible-Dropped") == "1"}, nil
}
