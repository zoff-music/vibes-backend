package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/vibe"
)

// Search searches for tracks on SoundCloud
func (c *Client) Search(ctx context.Context, query string) ([]vibe.MusicTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "Search")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf(
			"error validating soundcloud client in Search: client is not enabled",
		)
	}

	// Ensure valid access token
	err := c.EnsureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error ensuring token in Search: %w", err)
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
			"Accept":        "application/json; charset=utf-8",
		},
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf(
			"error requesting soundcloud tracks in Search: %w",
			err,
		)
	}

	var results []trackResponse
	err = json.Unmarshal(resp, &results)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding soundcloud response in Search: %w",
			err,
		)
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
			ID:              fmt.Sprintf("%d", res.ID),
			Source:          vibe.SourceTypeSoundCloud,
			Title:           res.Title,
			ChannelTitle:    username,
			ThumbnailURL:    artworkURL,
			Duration:        fmt.Sprintf("PT%dM%dS", (res.Duration/1000)/60, (res.Duration/1000)%60),
			DurationSeconds: res.Duration / 1000,
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
