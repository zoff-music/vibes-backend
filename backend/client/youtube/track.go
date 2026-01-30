package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes/monitoring/opentracing"

	"github.com/zoff-music/vibes/client"
	"github.com/zoff-music/vibes/vibe"
)

// GetTrack fetches details for a specific video ID
func (c *Client) GetTrack(ctx context.Context, id string) (*vibe.MusicTrack, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetTrack")
	defer span.Finish()

	if c.apiKey == "" {
		return nil, fmt.Errorf("YouTube API key not configured")
	}

	params := url.Values{}
	params.Set("part", "snippet,contentDetails")
	params.Set("id", id)
	params.Set("key", c.apiKey)

	reqData := client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/videos", c.Endpoint),
		Payload: &params,
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error request failed: %w", err)
	}

	var result videoResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("error track not found")
	}

	item := result.Items[0]
	return &vibe.MusicTrack{
		ID:           item.ID,
		Source:       vibe.SourceTypeYouTube,
		Title:        html.UnescapeString(item.Snippet.Title),
		ChannelTitle: html.UnescapeString(item.Snippet.ChannelTitle),
		ThumbnailURL: item.Snippet.Thumbnails.Medium.URL,
		Duration:     item.ContentDetails.Duration,
	}, nil
}

type videoResponse struct {
	Items []videoItem `json:"items"`
}

type videoItem struct {
	ID             string              `json:"id"`
	Snippet        videoSnippet        `json:"snippet"`
	ContentDetails videoContentDetails `json:"contentDetails"`
}

type videoSnippet struct {
	Title        string           `json:"title"`
	ChannelTitle string           `json:"channelTitle"`
	Thumbnails   searchThumbnails `json:"thumbnails"`
}

type videoContentDetails struct {
	Duration string `json:"duration"`
}
