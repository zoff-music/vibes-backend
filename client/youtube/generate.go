package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) VerifyGeneratedPlaylist(
	ctx context.Context,
	playlist *vibe.GeneratedPlaylist,
) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "VerifyGeneratedPlaylist")
	defer span.End()

	if c.apiKey == "" {
		return fmt.Errorf("error youtube api key not configured")
	}

	ids := make([]string, 0, len(playlist.Tracks))
	for index := range playlist.Tracks {
		ids = append(ids, playlist.Tracks[index].YouTubeVideoID)
		playlist.Tracks[index].YouTubeVideoID = ""
	}

	params := url.Values{}
	params.Set("part", "snippet,contentDetails")
	params.Set("id", strings.Join(ids, ","))
	params.Set("key", c.apiKey)

	responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/videos", c.Endpoint),
		Payload: &params,
	})
	if err != nil {
		return fmt.Errorf("error requesting youtube playlist verification: %w", err)
	}

	var response videoResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return fmt.Errorf("error unmarshaling youtube playlist verification: %w", err)
	}

	validTracks := make(map[string]vibe.GeneratedTrack, len(response.Items))
	for _, item := range response.Items {
		durationSeconds, durationErr := youtubeDurationSeconds(item.ContentDetails.Duration)
		if durationErr != nil {
			continue
		}
		if item.Snippet.CategoryID == youtubeMusicCategoryID &&
			durationSeconds > 0 &&
			durationSeconds <= generatedTrackMaxDurationSeconds {
			thumbnailURL := item.Snippet.Thumbnails.High.URL
			if thumbnailURL == "" {
				thumbnailURL = item.Snippet.Thumbnails.Medium.URL
			}
			if thumbnailURL == "" {
				thumbnailURL = item.Snippet.Thumbnails.Default.URL
			}
			validTracks[item.ID] = vibe.GeneratedTrack{
				YouTubeVideoID: item.ID,
				Title:          html.UnescapeString(item.Snippet.Title),
				Artist:         html.UnescapeString(item.Snippet.ChannelTitle),
				ThumbnailURL:   thumbnailURL,
				Duration:       durationSeconds,
			}
		}
	}

	for index := range playlist.Tracks {
		verified, ok := validTracks[ids[index]]
		if ok {
			playlist.Tracks[index] = verified
		}
	}

	return nil
}

func youtubeDurationSeconds(value string) (int, error) {
	if !strings.HasPrefix(value, "PT") {
		return 0, fmt.Errorf("error unsupported youtube duration %q", value)
	}

	duration := strings.TrimPrefix(value, "PT")
	number := ""
	totalSeconds := 0
	for _, character := range duration {
		if character >= '0' && character <= '9' {
			number += string(character)
			continue
		}
		if number == "" {
			return 0, fmt.Errorf("error invalid youtube duration %q", value)
		}

		amount, err := strconv.Atoi(number)
		if err != nil {
			return 0, fmt.Errorf("error parsing youtube duration %q: %w", value, err)
		}
		switch character {
		case 'H':
			totalSeconds += amount * 60 * 60
		case 'M':
			totalSeconds += amount * 60
		case 'S':
			totalSeconds += amount
		default:
			return 0, fmt.Errorf("error unsupported youtube duration unit %q", character)
		}
		number = ""
	}
	if number != "" {
		return 0, fmt.Errorf("error incomplete youtube duration %q", value)
	}

	return totalSeconds, nil
}

const youtubeMusicCategoryID = "10"

const generatedTrackMaxDurationSeconds = 20 * 60
