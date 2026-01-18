package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes/vibe"
)

const (
	baseURL = "https://www.googleapis.com/youtube/v3"
)

// Client implements vibe.MusicSearcher
type Client struct {
	apiKey string
	client *http.Client
}

// Init initializes the YouTube API client
func (c *Client) Init(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("YouTube API key is required")
	}
	c.apiKey = apiKey
	c.client = &http.Client{}
	return nil
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

// Search searches for videos on YouTube
func (c *Client) Search(ctx context.Context, query string) ([]vibe.MusicTrack, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("YouTube API key not configured")
	}

	u, _ := url.Parse(baseURL + "/search")
	q := u.Query()
	q.Set("part", "snippet")
	q.Set("q", query)
	q.Set("type", "video")
	q.Set("maxResults", "10")
	q.Set("key", c.apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("YouTube API returned status: %d", resp.StatusCode)
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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

	vidUrl, _ := url.Parse(baseURL + "/videos")
	vq := vidUrl.Query()
	vq.Set("part", "contentDetails")
	vq.Set("id", ids)
	vq.Set("key", c.apiKey)
	vidUrl.RawQuery = vq.Encode()

	vreq, err := http.NewRequestWithContext(ctx, "GET", vidUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create videos request: %w", err)
	}

	vresp, err := c.client.Do(vreq)
	if err != nil {
		return nil, fmt.Errorf("videos request failed: %w", err)
	}
	defer vresp.Body.Close()

	if vresp.StatusCode == http.StatusOK {
		var vresult videoResponse
		if err := json.NewDecoder(vresp.Body).Decode(&vresult); err == nil {
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
					Title:        item.Snippet.Title,
					ChannelTitle: item.Snippet.ChannelTitle,
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
			Title:        item.Snippet.Title,
			ChannelTitle: item.Snippet.ChannelTitle,
			ThumbnailURL: thmb,
		})
	}

	return tracks, nil
}

// GetTrack fetches details for a specific video ID
func (c *Client) GetTrack(ctx context.Context, id string) (*vibe.MusicTrack, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("YouTube API key not configured")
	}

	u, _ := url.Parse(baseURL + "/videos")
	q := u.Query()
	q.Set("part", "snippet,contentDetails")
	q.Set("id", id)
	q.Set("key", c.apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("YouTube API returned status: %d", resp.StatusCode)
	}

	var result videoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("track not found")
	}

	item := result.Items[0]
	return &vibe.MusicTrack{
		ID:           item.ID,
		Source:       vibe.SourceTypeYouTube,
		Title:        item.Snippet.Title,
		ChannelTitle: item.Snippet.ChannelTitle,
		ThumbnailURL: item.Snippet.Thumbnails.Medium.URL,
		Duration:     item.ContentDetails.Duration,
	}, nil
}
