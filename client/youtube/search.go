package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
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

	result, err := c.searchVideos(ctx, query, youtubeSearchResultCount)
	if err != nil {
		return nil, fmt.Errorf("error searching youtube videos: %w", err)
	}

	youtubeIDs := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		if item.ID.VideoID != "" {
			youtubeIDs = append(youtubeIDs, item.ID.VideoID)
		}
	}
	if len(youtubeIDs) == 0 {
		return []vibe.MusicTrack{}, nil
	}

	params := url.Values{}
	params.Set("part", "snippet,contentDetails,statistics")
	params.Set("id", strings.Join(youtubeIDs, ","))
	params.Set(
		"fields",
		"items(id,snippet(categoryId),contentDetails/duration,statistics(viewCount,likeCount))",
	)
	params.Set("key", c.apiKey)

	responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/videos", c.Endpoint),
		Payload: &params,
	})
	if err != nil {
		return nil, fmt.Errorf("error requesting youtube video details: %w", err)
	}

	var videoResult videoResponse
	err = json.Unmarshal(responseBody, &videoResult)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling youtube video details: %w", err)
	}

	videoItems := make(map[string]videoItem, len(videoResult.Items))
	for _, item := range videoResult.Items {
		videoItems[item.ID] = item
	}

	tracks := make([]vibe.MusicTrack, 0, len(result.Items))
	for _, item := range result.Items {
		if item.ID.VideoID == "" {
			continue
		}

		videoItem, ok := videoItems[item.ID.VideoID]
		if !ok || videoItem.Snippet.CategoryID != youtubeMusicCategoryID {
			continue
		}
		durationSeconds, err := youtubeDurationSeconds(
			videoItem.ContentDetails.Duration,
		)
		if err != nil {
			continue
		}

		thumbnailURL := item.Snippet.Thumbnails.High.URL
		if thumbnailURL == "" {
			thumbnailURL = item.Snippet.Thumbnails.Medium.URL
		}
		if thumbnailURL == "" {
			thumbnailURL = item.Snippet.Thumbnails.Default.URL
		}

		viewCount, _ := strconv.ParseUint(
			videoItem.Statistics.ViewCount,
			10,
			64,
		)
		likeCount, _ := strconv.ParseUint(
			videoItem.Statistics.LikeCount,
			10,
			64,
		)
		tracks = append(tracks, vibe.MusicTrack{
			ID:              item.ID.VideoID,
			Source:          vibe.SourceTypeYouTube,
			Title:           html.UnescapeString(item.Snippet.Title),
			ChannelTitle:    html.UnescapeString(item.Snippet.ChannelTitle),
			ThumbnailURL:    thumbnailURL,
			Duration:        videoItem.ContentDetails.Duration,
			DurationSeconds: durationSeconds,
			ViewCount:       viewCount,
			LikeCount:       likeCount,
		})
		if len(tracks) == youtubeSearchDisplayCount {
			break
		}
	}

	return tracks, nil
}

func (c *Client) searchVideos(
	ctx context.Context,
	query string,
	maxResults int,
) (*searchResponse, error) {
	c.searchQuotaMu.RLock()
	searchQuotaReset := c.searchQuotaReset
	c.searchQuotaMu.RUnlock()
	if time.Now().Before(searchQuotaReset) {
		return nil, internalerror.ErrProviderQuotaExceeded{
			Err:      fmt.Errorf("error youtube search quota cached until %s", searchQuotaReset.Format(time.RFC3339)),
			Provider: youtubeProvider,
		}
	}

	params := url.Values{}
	params.Set("part", "snippet")
	params.Set("q", query)
	params.Set("type", "video")
	params.Set("videoCategoryId", youtubeMusicCategoryID)
	params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	params.Set("fields", "items(id/videoId,snippet(title,channelTitle,thumbnails))")
	params.Set("key", c.apiKey)

	responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/search", c.Endpoint),
		Payload: &params,
	})
	if err != nil {
		var statusCodeError client.HTTPStatusCodeError
		if errors.As(err, &statusCodeError) &&
			(statusCodeError.StatusCode == http.StatusForbidden ||
				statusCodeError.StatusCode == http.StatusTooManyRequests) &&
			(strings.Contains(statusCodeError.Message, "youtube.googleapis.com/search_list") ||
				strings.Contains(statusCodeError.Message, "Search Queries") ||
				strings.Contains(statusCodeError.Message, "quotaExceeded")) {
			now := time.Now().In(c.searchQuotaZone)
			reset := time.Date(
				now.Year(),
				now.Month(),
				now.Day()+1,
				0,
				0,
				0,
				0,
				c.searchQuotaZone,
			)
			c.searchQuotaMu.Lock()
			c.searchQuotaReset = reset
			c.searchQuotaMu.Unlock()

			return nil, internalerror.ErrProviderQuotaExceeded{
				Err:      fmt.Errorf("error requesting youtube search: %w", err),
				Provider: youtubeProvider,
			}
		}

		return nil, fmt.Errorf("error requesting youtube search: %w", err)
	}

	var result searchResponse
	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling youtube search response: %w", err)
	}

	return &result, nil
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

const youtubeProvider = "youtube"

const youtubeQuotaLocation = "America/Los_Angeles"

const youtubeSearchDisplayCount = 15

const youtubeSearchResultCount = 25
