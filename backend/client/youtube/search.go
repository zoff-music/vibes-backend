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

// Search searches for videos on YouTube
func (c *Client) Search(ctx context.Context, query string) ([]vibe.MusicTrack, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Search")
	defer span.Finish()

	if c.apiKey == "" {
		return nil, fmt.Errorf("YouTube API key not configured")
	}

	params := url.Values{}
	params.Set("part", "snippet")
	params.Set("q", query)
	params.Set("type", "video")
	params.Set("maxResults", "10")
	params.Set("key", c.apiKey)

	reqData := client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/search", c.Endpoint),
		Payload: &params,
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error request failed: %w", err)
	}

	var result searchResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	tracks := make([]vibe.MusicTrack, 0, len(result.Items))
	if len(result.Items) == 0 {
		return tracks, nil
	}

	// Fetch durations for all videos in one call
	ids := ""
	for i, item := range result.Items {
		if i > 0 {
			ids += ","
		}
		ids += item.ID.VideoID
	}

	vparams := url.Values{}
	vparams.Set("part", "contentDetails")
	vparams.Set("id", ids)
	vparams.Set("key", c.apiKey)

	vreqData := client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/videos", c.Endpoint),
		Payload: &vparams,
	}

	vresp, err := c.HTTPClient.RequestBytes(ctx, vreqData)
	if err == nil {
		var vresult videoResponse
		err := json.Unmarshal(vresp, &vresult)
		if err == nil {
			durations := make(map[string]string)
			for _, vitem := range vresult.Items {
				durations[vitem.ID] = vitem.ContentDetails.Duration
			}

			// Map durations back to search items
			for _, item := range result.Items {
				if item.ID.VideoID == "" {
					continue
				}

				thmb := item.Snippet.Thumbnails.High.URL
				if thmb == "" {
					thmb = item.Snippet.Thumbnails.Medium.URL
				}
				if thmb == "" {
					thmb = item.Snippet.Thumbnails.Default.URL
				}

				tracks = append(tracks, vibe.MusicTrack{
					ID:           item.ID.VideoID,
					Source:       vibe.SourceTypeYouTube,
					Title:        html.UnescapeString(item.Snippet.Title),
					ChannelTitle: html.UnescapeString(item.Snippet.ChannelTitle),
					ThumbnailURL: thmb,
					Duration:     durations[item.ID.VideoID],
				})
			}
			return tracks, nil
		}
	}

	// Fallback if video details fetch fails
	for _, item := range result.Items {
		if item.ID.VideoID == "" {
			continue
		}

		thmb := item.Snippet.Thumbnails.High.URL
		if thmb == "" {
			thmb = item.Snippet.Thumbnails.Medium.URL
		}
		if thmb == "" {
			thmb = item.Snippet.Thumbnails.Default.URL
		}

		tracks = append(tracks, vibe.MusicTrack{
			ID:           item.ID.VideoID,
			Source:       vibe.SourceTypeYouTube,
			Title:        html.UnescapeString(item.Snippet.Title),
			ChannelTitle: html.UnescapeString(item.Snippet.ChannelTitle),
			ThumbnailURL: thmb,
		})
	}

	return tracks, nil
}

type searchResponse struct {
	Items []searchItem `json:"items"`
}

type searchItem struct {
	ID      searchID      `json:"id"`
	Snippet searchSnippet `json:"snippet"`
}

type searchID struct {
	VideoID string `json:"videoId"`
}

type searchSnippet struct {
	Title        string           `json:"title"`
	ChannelTitle string           `json:"channelTitle"`
	Thumbnails   searchThumbnails `json:"thumbnails"`
}

type searchThumbnails struct {
	Default thumbnail `json:"default"`
	Medium  thumbnail `json:"medium"`
	High    thumbnail `json:"high"`
}

type thumbnail struct {
	URL string `json:"url"`
}
