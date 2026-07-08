package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetTrack fetches details for a specific track ID
func (c *Client) GetTrack(ctx context.Context, id string) (*vibe.MusicTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetTrack")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf("error soundcloud client is not enabled")
	}

	// Ensure valid access token
	err := c.EnsureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error ensuring token in GetTrack: %w", err)
	}

	reqData := client.HTTPRequestData{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("%s/tracks/%s", c.Endpoint, id),
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("OAuth %s", c.accessToken),
		},
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error request failed: %w", err)
	}

	var res trackResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	username := "Unknown"
	if res.User != nil {
		username = res.User.Username
	}

	artworkURL := ""
	if res.ArtworkURL != nil {
		artworkURL = *res.ArtworkURL
	}

	return &vibe.MusicTrack{
		ID:           fmt.Sprintf("%d", res.ID),
		Source:       vibe.SourceTypeSoundCloud,
		Title:        res.Title,
		ChannelTitle: username,
		ThumbnailURL: artworkURL,
		Duration:     fmt.Sprintf("PT%dM%dS", (res.Duration/1000)/60, (res.Duration/1000)%60),
	}, nil
}
