package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/vibe"
)

// Search searches for videos on YouTube
func (c *Client) Search(ctx context.Context, query string) ([]vibe.MusicTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "Search")
	defer span.End()

	if c.apiKey == "" {
		return nil, fmt.Errorf("error youtube api key not configured")
	}

	const targetResults = 15

	fetchSearch := func(maxResults int, categoryID string) (searchResponse, error) {
		params := url.Values{}
		params.Set("part", "snippet")
		params.Set("q", query)
		params.Set("type", "video")
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
		params.Set("key", c.apiKey)
		if categoryID != "" {
			params.Set("videoCategoryId", categoryID)
		}

		reqData := client.HTTPRequestData{
			Method:  http.MethodGet,
			URL:     fmt.Sprintf("%s/search", c.Endpoint),
			Payload: &params,
		}

		resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
		if err != nil {
			return searchResponse{}, fmt.Errorf("error requesting youtube search in Search: %w", err)
		}

		var result searchResponse
		err = json.Unmarshal(resp, &result)
		if err != nil {
			return searchResponse{}, fmt.Errorf("error unmarshaling youtube search response in Search: %w", err)
		}
		return result, nil
	}

	fetchTracksWithDetails := func(items []searchItem) ([]vibe.MusicTrack, error) {
		if len(items) == 0 {
			return []vibe.MusicTrack{}, nil
		}

		ids := ""
		for i, item := range items {
			if i > 0 {
				ids += ","
			}
			ids += item.ID.VideoID
		}

		vparams := url.Values{}
		vparams.Set("part", "contentDetails,snippet")
		vparams.Set("id", ids)
		vparams.Set("key", c.apiKey)

		vreqData := client.HTTPRequestData{
			Method:  http.MethodGet,
			URL:     fmt.Sprintf("%s/videos", c.Endpoint),
			Payload: &vparams,
		}

		vresp, err := c.HTTPClient.RequestBytes(ctx, vreqData)
		if err != nil {
			return nil, fmt.Errorf("error requesting youtube video details in Search: %w", err)
		}

		var vresult videoResponse
		err = json.Unmarshal(vresp, &vresult)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling youtube video details in Search: %w", err)
		}

		durations := make(map[string]string)
		for _, vitem := range vresult.Items {
			durations[vitem.ID] = vitem.ContentDetails.Duration
		}

		tracks := make([]vibe.MusicTrack, 0, len(items))
		for _, item := range items {
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

	// First: hard filter to music category (id 10)
	musicResult, err := fetchSearch(targetResults, "10")
	if err != nil {
		return nil, fmt.Errorf("error request failed: %w", err)
	}

	musicTracks, err := fetchTracksWithDetails(musicResult.Items)
	if err != nil {
		return nil, fmt.Errorf("error request failed: %w", err)
	}

	if len(musicTracks) >= targetResults {
		return musicTracks[:targetResults], nil
	}

	// Second: normal search, fill remaining slots, dedupe
	remaining := targetResults - len(musicTracks)
	mixedResult, err := fetchSearch(remaining, "")
	if err != nil {
		return nil, fmt.Errorf("error request failed: %w", err)
	}

	mixedTracks, err := fetchTracksWithDetails(mixedResult.Items)
	if err != nil {
		return nil, fmt.Errorf("error request failed: %w", err)
	}

	seen := make(map[string]struct{}, len(musicTracks))
	for _, track := range musicTracks {
		seen[track.ID] = struct{}{}
	}

	tracks := make([]vibe.MusicTrack, 0, targetResults)
	tracks = append(tracks, musicTracks...)
	for _, track := range mixedTracks {
		if len(tracks) >= targetResults {
			break
		}
		if _, exists := seen[track.ID]; exists {
			continue
		}
		tracks = append(tracks, track)
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
