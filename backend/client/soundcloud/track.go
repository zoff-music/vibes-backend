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

// GetTrack fetches details for a specific track ID
func (c *Client) GetTrack(ctx context.Context, id string) (*vibe.MusicTrack, error) {
	if !c.Enabled {
		return nil, fmt.Errorf("SoundCloud client is not enabled")
	}

	params := url.Values{}
	params.Set("client_id", c.apiKey)

	reqData := client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/tracks/%s", c.Endpoint, id),
		Payload: &params,
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

	return &vibe.MusicTrack{
		ID:           fmt.Sprintf("%d", res.ID),
		Source:       vibe.SourceTypeSoundCloud,
		Title:        res.Title,
		ChannelTitle: res.User.Username,
		ThumbnailURL: res.ArtworkURL,
		Duration:     fmt.Sprintf("PT%dM%dS", (res.Duration/1000)/60, (res.Duration/1000)%60),
	}, nil
}
