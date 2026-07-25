// Package sse contains the low-level server-sent events transport client.
package sse

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

// Client writes server-sent event streams.
type Client struct {
	HeartbeatInterval time.Duration
}

// Init sets up the SSE client.
func (c *Client) Init(ctx context.Context) error {
	span, _ := tracing.StartSpanFromContext(ctx, "Init")
	defer span.End()

	c.HeartbeatInterval = defaultHeartbeatRate

	return nil
}

// Close closes the SSE client.
func (c *Client) Close() error {
	c.HeartbeatInterval = 0

	return nil
}

// Write writes and flushes an SSE event.
func (c *Client) Write(w http.ResponseWriter, flusher http.Flusher, event vibe.StreamEvent) error {
	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Payload)
	if err != nil {
		return fmt.Errorf("error writing event in Write: %w", err)
	}

	flusher.Flush()

	return nil
}

// Heartbeat writes and flushes an SSE heartbeat.
func (c *Client) Heartbeat(w http.ResponseWriter, flusher http.Flusher) {
	_, _ = fmt.Fprint(w, ": heartbeat\n\n")
	flusher.Flush()
}

// HeartbeatRate returns how often an SSE connection should be kept alive.
func (c *Client) HeartbeatRate() time.Duration {
	return c.HeartbeatInterval
}

const defaultHeartbeatRate = 5 * time.Second
