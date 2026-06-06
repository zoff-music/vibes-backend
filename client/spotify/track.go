package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zoff-music/vibes-backend/monitoring/opentracing"
	"net/http"
	"strings"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetTrack fetches details for a specific track ID
func (c *Client) GetTrack(ctx context.Context, id string) (*vibe.MusicTrack, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetTrack")
	defer span.Finish()

	if !c.Enabled {
		return nil, fmt.Errorf("error Spotify client is not enabled")
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting access token: %w", err)
	}

	reqData := client.HTTPRequestData{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("%s/tracks/%s", c.Endpoint, id),
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error request failed: %w", err)
	}

	var item spotifyTrack
	err = json.Unmarshal(resp, &item)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	artists := make([]string, 0, len(item.Artists))
	for _, a := range item.Artists {
		artists = append(artists, a.Name)
	}

	thumbnail := ""
	if len(item.Album.Images) > 0 {
		thumbnail = item.Album.Images[0].URL
	}

	return &vibe.MusicTrack{
		ID:           item.ID,
		Source:       vibe.SourceTypeSpotify,
		Title:        item.Name,
		ChannelTitle: strings.Join(artists, ", "),
		ThumbnailURL: thumbnail,
		Duration:     fmt.Sprintf("PT%dM%dS", (item.DurationMS/1000)/60, (item.DurationMS/1000)%60),
	}, nil
}
