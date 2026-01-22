package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes/client"
	"github.com/zoff-music/vibes/vibe"
)

// Search searches for tracks on SoundCloud
func (c *Client) Search(ctx context.Context, query string) ([]vibe.MusicTrack, error) {
	if !c.Enabled {
		return nil, fmt.Errorf("SoundCloud client is not enabled")
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", "10")
	params.Set("client_id", c.apiKey)

	reqData := client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/tracks", c.Endpoint),
		Payload: &params,
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error request failed: %w", err)
	}

	var results []trackResponse
	err = json.Unmarshal(resp, &results)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	tracks := make([]vibe.MusicTrack, 0, len(results))
	for _, res := range results {
		tracks = append(tracks, vibe.MusicTrack{
			ID:           fmt.Sprintf("%d", res.ID),
			Source:       vibe.SourceTypeSoundCloud,
			Title:        res.Title,
			ChannelTitle: res.User.Username,
			ThumbnailURL: res.ArtworkURL,
			Duration:     fmt.Sprintf("PT%dM%dS", (res.Duration/1000)/60, (res.Duration/1000)%60),
		})
	}

	return tracks, nil
}

type trackResponse struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	User        user   `json:"user"`
	ArtworkURL  string `json:"artwork_url"`
	Duration    int    `json:"duration"`
	Permalink   string `json:"permalink"`
	StreamURL   string `json:"stream_url"`
	Description string `json:"description"`
}

type user struct {
	Username string `json:"username"`
}
