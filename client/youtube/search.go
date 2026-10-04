package youtube

import (
	"context"
	"encoding/json/v2"
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
		return nil, fmt.Errorf(
			"error validating youtube API key in Search: not configured",
		)
	}

	result, err := c.searchVideos(ctx, query, youtubeSearchResultCount)
	if err != nil {
		return nil, fmt.Errorf(
			"error searching youtube videos in Search: %w",
			err,
		)
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
	params.Set("part", "snippet,contentDetails,statistics,status")
	params.Set("id", strings.Join(youtubeIDs, ","))
	params.Set(
		"fields",
		"items(id,snippet(categoryId,liveBroadcastContent),contentDetails(duration,contentRating/ytRating,regionRestriction),statistics(viewCount,likeCount),status(embeddable,madeForKids))",
	)
	params.Set("key", c.apiKey)

	responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/videos", c.Endpoint),
		Payload: &params,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"error requesting youtube video details in Search: %w",
			err,
		)
	}

	var videoResult videoResponse
	err = json.Unmarshal(responseBody, &videoResult)
	if err != nil {
		return nil, fmt.Errorf(
			"error unmarshaling youtube video details in Search: %w",
			err,
		)
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
		if !ok ||
			videoItem.Status.MadeForKids ||
			!videoItem.Status.Embeddable ||
			videoItem.Snippet.CategoryID != youtubeMusicCategoryID ||
			videoItem.isLiveVideo() {
			continue
		}
		durationSeconds, err := youtubeDurationSeconds(
			videoItem.ContentDetails.Duration,
		)
		if err != nil {
			continue
		}
		if vibe.IsLiveVideo(vibe.SourceTypeYouTube, durationSeconds) {
			continue
		}

		thumbnailURL := item.Snippet.Thumbnails.High.URL
		if thumbnailURL == "" {
			thumbnailURL = item.Snippet.Thumbnails.Medium.URL
		}
		if thumbnailURL == "" {
			thumbnailURL = item.Snippet.Thumbnails.Default.URL
		}

		viewCount, err := strconv.ParseUint(
			videoItem.Statistics.ViewCount,
			10,
			64,
		)
		if err != nil {
			viewCount = 0
		}
		likeCount, err := strconv.ParseUint(
			videoItem.Statistics.LikeCount,
			10,
			64,
		)
		if err != nil {
			likeCount = 0
		}
		tracks = append(tracks, vibe.MusicTrack{
			ID:                  item.ID.VideoID,
			Source:              vibe.SourceTypeYouTube,
			ProviderURL:         fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.ID.VideoID),
			Title:               html.UnescapeString(item.Snippet.Title),
			ChannelTitle:        html.UnescapeString(item.Snippet.ChannelTitle),
			ThumbnailURL:        thumbnailURL,
			Duration:            videoItem.ContentDetails.Duration,
			DurationSeconds:     durationSeconds,
			ViewCount:           viewCount,
			LikeCount:           likeCount,
			PlaybackRestriction: videoItem.playbackRestriction(),
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
	params := url.Values{}
	params.Set("part", "snippet")
	params.Set("q", query)
	params.Set("type", "video")
	params.Set("videoCategoryId", youtubeMusicCategoryID)
	params.Set("videoEmbeddable", "true")
	params.Set("videoSyndicated", "true")
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
		isStatusError := errors.As(err, &statusCodeError)
		var quotaExceeded bool

		if isStatusError &&
			(statusCodeError.StatusCode == http.StatusForbidden ||
				statusCodeError.StatusCode == http.StatusTooManyRequests) {
			var failure searchErrorResponse
			decodeErr := json.Unmarshal(statusCodeError.ResponseBody, &failure)

			if decodeErr == nil {
				for _, reason := range failure.Error.Errors {
					if reason.Reason == "quotaExceeded" || reason.Reason == "dailyLimitExceeded" {
						quotaExceeded = true
					}
				}

				for _, detail := range failure.Error.Details {
					if detail.Metadata.QuotaMetric == "youtube.googleapis.com/search_list" &&
						detail.Metadata.QuotaLimit == "defaultSearchListPerDayPerProject" {
						quotaExceeded = true
					}
				}
			}
		}

		if quotaExceeded {
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
			return nil, internalerror.ErrProviderQuotaExceeded{
				Err: fmt.Errorf(
					"error requesting youtube search in searchVideos: %w",
					err,
				),
				Provider: youtubeProvider,
				ResetAt:  reset,
			}
		}

		return nil, fmt.Errorf(
			"error requesting youtube search in searchVideos: %w",
			err,
		)
	}

	var result searchResponse
	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		return nil, fmt.Errorf(
			"error unmarshaling youtube search response in searchVideos: %w",
			err,
		)
	}

	return &result, nil
}

type searchErrorResponse struct {
	Error searchError `json:"error"`
}

type searchError struct {
	Errors  []searchErrorReason `json:"errors"`
	Details []searchErrorDetail `json:"details"`
}

type searchErrorReason struct {
	Reason string `json:"reason"`
}

type searchErrorDetail struct {
	Metadata searchErrorMetadata `json:"metadata"`
}

type searchErrorMetadata struct {
	QuotaMetric string `json:"quota_metric"`
	QuotaLimit  string `json:"quota_limit"`
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
