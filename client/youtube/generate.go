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

func (c *Client) SearchGeneratedPlaylist(
	ctx context.Context,
	playlist vibe.GeneratedPlaylist,
) (*vibe.GeneratedPlaylist, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SearchGeneratedPlaylist")
	defer span.End()

	if c.apiKey == "" {
		return nil, fmt.Errorf("error youtube api key not configured")
	}

	youtubeIDs := make([]string, 0, len(playlist))
	for _, track := range playlist {
		if track.YouTubeID != "" {
			youtubeIDs = append(youtubeIDs, track.YouTubeID)
		}
	}

	tracksByID, err := c.getGeneratedTracks(ctx, youtubeIDs)
	if err != nil {
		return nil, fmt.Errorf("error getting generated tracks by ID: %w", err)
	}

	found := make(vibe.GeneratedPlaylist, 0, len(playlist))
	seen := make(map[string]bool, len(playlist))
	for _, candidate := range playlist {
		track, ok := tracksByID[candidate.YouTubeID]
		if !ok {
			track, err = c.searchGeneratedTrack(ctx, candidate)
			if err != nil {
				return nil, fmt.Errorf("error searching generated track: %w", err)
			}
		}
		if track.YouTubeID == "" || seen[track.YouTubeID] {
			continue
		}

		seen[track.YouTubeID] = true
		found = append(found, track)
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("error no generated songs found on youtube")
	}

	return &found, nil
}

func (c *Client) searchGeneratedTrack(
	ctx context.Context,
	candidate vibe.GeneratedTrack,
) (vibe.GeneratedTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "searchGeneratedTrack")
	defer span.End()

	if candidate.Title == "" || candidate.Artist == "" {
		return vibe.GeneratedTrack{}, nil
	}

	params := url.Values{}
	params.Set("part", "snippet")
	params.Set("q", candidate.Artist+" "+candidate.Title)
	params.Set("type", "video")
	params.Set("videoCategoryId", youtubeMusicCategoryID)
	params.Set("maxResults", strconv.Itoa(generatedTrackSearchResults))
	params.Set("key", c.apiKey)

	responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/search", c.Endpoint),
		Payload: &params,
	})
	if err != nil {
		return vibe.GeneratedTrack{}, fmt.Errorf("error requesting youtube generated track search: %w", err)
	}

	var response searchResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return vibe.GeneratedTrack{}, fmt.Errorf("error unmarshaling youtube generated track search: %w", err)
	}

	youtubeIDs := make([]string, 0, len(response.Items))
	for _, item := range response.Items {
		if item.ID.VideoID != "" {
			youtubeIDs = append(youtubeIDs, item.ID.VideoID)
		}
	}

	tracksByID, err := c.getGeneratedTracks(ctx, youtubeIDs)
	if err != nil {
		return vibe.GeneratedTrack{}, fmt.Errorf("error getting searched generated tracks: %w", err)
	}

	for _, youtubeID := range youtubeIDs {
		track, ok := tracksByID[youtubeID]
		if ok {
			return track, nil
		}
	}

	return vibe.GeneratedTrack{}, nil
}

func (c *Client) getGeneratedTracks(
	ctx context.Context,
	youtubeIDs []string,
) (map[string]vibe.GeneratedTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getGeneratedTracks")
	defer span.End()

	tracks := make(map[string]vibe.GeneratedTrack, len(youtubeIDs))
	if len(youtubeIDs) == 0 {
		return tracks, nil
	}

	params := url.Values{}
	params.Set("part", "snippet,contentDetails")
	params.Set("id", strings.Join(youtubeIDs, ","))
	params.Set("key", c.apiKey)

	responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/videos", c.Endpoint),
		Payload: &params,
	})
	if err != nil {
		return nil, fmt.Errorf("error requesting youtube generated track details: %w", err)
	}

	var response videoResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling youtube generated track details: %w", err)
	}

	for _, item := range response.Items {
		durationSeconds, durationErr := youtubeDurationSeconds(item.ContentDetails.Duration)
		if durationErr != nil ||
			item.Snippet.CategoryID != youtubeMusicCategoryID ||
			durationSeconds <= 0 ||
			durationSeconds > generatedTrackMaxDurationSeconds {
			continue
		}

		thumbnailURL := item.Snippet.Thumbnails.High.URL
		if thumbnailURL == "" {
			thumbnailURL = item.Snippet.Thumbnails.Medium.URL
		}
		if thumbnailURL == "" {
			thumbnailURL = item.Snippet.Thumbnails.Default.URL
		}

		tracks[item.ID] = vibe.GeneratedTrack{
			YouTubeID:    item.ID,
			Title:        html.UnescapeString(item.Snippet.Title),
			Artist:       html.UnescapeString(item.Snippet.ChannelTitle),
			ThumbnailURL: thumbnailURL,
			Duration:     durationSeconds,
		}
	}

	return tracks, nil
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

const generatedTrackSearchResults = 5
