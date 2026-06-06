package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes-backend/monitoring/opentracing"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/vibe"
)

// Search searches for tracks on SoundCloud
func (c *Client) Search(ctx context.Context, query string) ([]vibe.MusicTrack, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Search")
	defer span.Finish()

	if !c.Enabled {
		return nil, fmt.Errorf("SoundCloud client is not enabled")
	}

	// Ensure valid access token
	err := c.EnsureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure token: %w", err)
	}

	if len(c.accessToken) > 5 {
		fmt.Printf("DEBUG: Access key is valid: %s...\n", c.accessToken[:5])
	} else {
		fmt.Printf("DEBUG: Access key is invalid/short: %s\n", c.accessToken)
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", "25")
	params.Set("access", "playable,preview")

	reqData := client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/tracks", c.Endpoint),
		Payload: &params,
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("OAuth %s", c.accessToken),
		},
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error request failed: %w", err)
	}

	var results []trackResponse
	err = json.Unmarshal(resp, &results)
	if err != nil {
		// Log error and raw response for debugging
		fmt.Printf("ERROR: SoundCloud search failed to decode: %v\nResponse: %s\n", err, string(resp))
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	tracks := make([]vibe.MusicTrack, 0, len(results))
	for _, res := range results {
		username := "Unknown"
		if res.User != nil {
			username = res.User.Username
		}

		artworkURL := ""
		if res.ArtworkURL != nil {
			artworkURL = *res.ArtworkURL
		}

		tracks = append(tracks, vibe.MusicTrack{
			ID:           fmt.Sprintf("%d", res.ID),
			Source:       vibe.SourceTypeSoundCloud,
			Title:        res.Title,
			ChannelTitle: username,
			ThumbnailURL: artworkURL,
			Duration:     fmt.Sprintf("PT%dM%dS", (res.Duration/1000)/60, (res.Duration/1000)%60),
		})
	}

	return tracks, nil
}

type trackResponse struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	User        *user   `json:"user"`
	ArtworkURL  *string `json:"artwork_url"`
	Duration    int     `json:"duration"`
	Permalink   string  `json:"permalink_url"`
	StreamURL   string  `json:"stream_url"`
	Description string  `json:"description"`
}

type user struct {
	Username string `json:"username"`
}
