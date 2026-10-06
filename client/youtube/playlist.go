package youtube

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) GetProviderPlaylist(
	ctx context.Context,
	id string,
	roomType vibe.RoomType,
) (*vibe.ProviderPlaylist, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetProviderPlaylist")
	defer span.End()

	if !roomType.AllowsSource(vibe.SourceTypeYouTube) {
		return nil, fmt.Errorf("error getting youtube playlist: invalid room type %q", roomType)
	}

	if c.apiKey == "" {
		return nil, fmt.Errorf(
			"error validating youtube API key in GetProviderPlaylist: not configured",
		)
	}

	videoIDs := make([]string, 0)
	nextPageToken := ""
	truncated := false
	for {
		params := url.Values{}
		params.Set("part", "contentDetails")
		params.Set("playlistId", id)
		params.Set("maxResults", fmt.Sprintf("%d", youtubePlaylistPageSize))
		params.Set("key", c.apiKey)
		if nextPageToken != "" {
			params.Set("pageToken", nextPageToken)
		}

		responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
			Method:  http.MethodGet,
			URL:     fmt.Sprintf("%s/playlistItems", c.Endpoint),
			Payload: &params,
		})
		if err != nil {
			return nil, fmt.Errorf(
				"error requesting youtube playlist items in GetProviderPlaylist: %w",
				err,
			)
		}

		var page youtubePlaylistItemsResponse
		err = json.Unmarshal(responseBody, &page)
		if err != nil {
			return nil, fmt.Errorf(
				"error decoding youtube playlist items in GetProviderPlaylist: %w",
				err,
			)
		}

		for _, item := range page.Items {
			if item.ContentDetails.VideoID == "" {
				continue
			}
			if len(videoIDs) == playlistItemLimit {
				truncated = true
				break
			}
			videoIDs = append(videoIDs, item.ContentDetails.VideoID)
		}
		if truncated || page.NextPageToken == "" {
			break
		}
		nextPageToken = page.NextPageToken
	}

	playlistItems := make([]vibe.ProviderItem, 0, len(videoIDs))
	skippedEmbeddingCount := 0
	skippedMadeForKidsCount := 0
	skippedRoomTypeCount := 0
	for start := 0; start < len(videoIDs); start += youtubePlaylistPageSize {
		end := start + youtubePlaylistPageSize
		if end > len(videoIDs) {
			end = len(videoIDs)
		}

		params := url.Values{}
		params.Set("part", "snippet,contentDetails,status")
		params.Set("id", strings.Join(videoIDs[start:end], ","))
		params.Set("key", c.apiKey)

		responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
			Method:  http.MethodGet,
			URL:     fmt.Sprintf("%s/videos", c.Endpoint),
			Payload: &params,
		})
		if err != nil {
			return nil, fmt.Errorf(
				"error requesting youtube playlist video details in GetProviderPlaylist: %w",
				err,
			)
		}

		var response videoResponse
		err = json.Unmarshal(responseBody, &response)
		if err != nil {
			return nil, fmt.Errorf(
				"error decoding youtube playlist video details in GetProviderPlaylist: %w",
				err,
			)
		}

		items := make(map[string]videoItem, len(response.Items))
		for _, item := range response.Items {
			items[item.ID] = item
		}
		for _, videoID := range videoIDs[start:end] {
			item, ok := items[videoID]
			if !ok || item.isLiveVideo() {
				continue
			}

			if roomType == vibe.RoomTypeMusic && item.Snippet.CategoryID != youtubeMusicCategoryID {
				skippedRoomTypeCount++
				continue
			}

			if item.Status.MadeForKids {
				skippedMadeForKidsCount++
				continue
			}

			if !item.Status.Embeddable {
				skippedEmbeddingCount++
				continue
			}

			durationSeconds, err := youtubeDurationSeconds(item.ContentDetails.Duration)
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

			playlistItems = append(playlistItems, vibe.ProviderItem{
				ID:                  item.ID,
				Source:              vibe.SourceTypeYouTube,
				CategoryID:          item.Snippet.CategoryID,
				ProviderURL:         fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.ID),
				Title:               html.UnescapeString(item.Snippet.Title),
				Publisher:           html.UnescapeString(item.Snippet.ChannelTitle),
				ThumbnailURL:        thumbnailURL,
				Duration:            item.ContentDetails.Duration,
				DurationSeconds:     durationSeconds,
				PlaybackRestriction: item.playbackRestriction(),
			})
		}
	}

	return &vibe.ProviderPlaylist{
		ID:                      id,
		Source:                  vibe.SourceTypeYouTube,
		Items:                   playlistItems,
		Truncated:               truncated,
		SkippedEmbeddingCount:   skippedEmbeddingCount,
		SkippedRoomTypeCount:    skippedRoomTypeCount,
		SkippedMadeForKidsCount: skippedMadeForKidsCount,
	}, nil
}

type youtubePlaylistItemsResponse struct {
	Items         []youtubePlaylistItem `json:"items"`
	NextPageToken string                `json:"nextPageToken"`
}

type youtubePlaylistItem struct {
	ContentDetails youtubePlaylistItemContentDetails `json:"contentDetails"`
}

type youtubePlaylistItemContentDetails struct {
	VideoID string `json:"videoId"`
}

const youtubePlaylistPageSize = 50

const playlistItemLimit = 500
