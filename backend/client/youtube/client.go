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

// Client implements vibe.YouTubeFetcher
type Client struct {
	apiKey string
	client *http.Client
}

// NewClient creates a new YouTube API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

type searchResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
		Snippet struct {
			Title        string `json:"title"`
			ChannelTitle string `json:"channelTitle"`
			Thumbnails   struct {
				Default struct {
					URL string `json:"url"`
				} `json:"default"`
				Medium struct {
					URL string `json:"url"`
				} `json:"medium"`
				High struct {
					URL string `json:"url"`
				} `json:"high"`
			} `json:"thumbnails"`
		} `json:"snippet"`
	} `json:"items"`
}

type videoResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title        string `json:"title"`
			ChannelTitle string `json:"channelTitle"`
			Thumbnails   struct {
				Medium struct {
					URL string `json:"url"`
				} `json:"medium"`
			} `json:"thumbnails"`
		} `json:"snippet"`
		ContentDetails struct {
			Duration string `json:"duration"`
		} `json:"contentDetails"`
	} `json:"items"`
}

// Search searches for videos on YouTube
func (c *Client) Search(ctx context.Context, query string) ([]vibe.YouTubeVideo, error) {
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

	videos := make([]vibe.YouTubeVideo, 0, len(result.Items))
	for _, item := range result.Items {
		// Only include items with a video ID (safety check, though type=video should ensure this)
		if item.ID.VideoID == "" {
			continue
		}

		thumbnail := item.Snippet.Thumbnails.High.URL
		if thumbnail == "" {
			thumbnail = item.Snippet.Thumbnails.Medium.URL
		}
		if thumbnail == "" {
			thumbnail = item.Snippet.Thumbnails.Default.URL
		}

		videos = append(videos, vibe.YouTubeVideo{
			ID:           item.ID.VideoID,
			Title:        item.Snippet.Title,
			ChannelTitle: item.Snippet.ChannelTitle,
			ThumbnailURL: thumbnail,
		})
	}

	return videos, nil
}

// GetVideo fetches details for a specific video ID
func (c *Client) GetVideo(ctx context.Context, id string) (*vibe.YouTubeVideo, error) {
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
		return nil, fmt.Errorf("video not found")
	}

	item := result.Items[0]
	return &vibe.YouTubeVideo{
		ID:           item.ID,
		Title:        item.Snippet.Title,
		ChannelTitle: item.Snippet.ChannelTitle,
		ThumbnailURL: item.Snippet.Thumbnails.Medium.URL,
		Duration:     item.ContentDetails.Duration,
	}, nil
}
