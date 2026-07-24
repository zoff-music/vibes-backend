package youtube

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
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
	seenYouTubeIDs := make(map[string]bool, len(playlist))
	for _, track := range playlist {
		if track.YouTubeID == "" || seenYouTubeIDs[track.YouTubeID] {
			continue
		}
		seenYouTubeIDs[track.YouTubeID] = true
		youtubeIDs = append(youtubeIDs, track.YouTubeID)
	}

	tracksByID, err := c.getGeneratedTracks(ctx, youtubeIDs)
	if err != nil {
		return nil, fmt.Errorf("error getting generated tracks by ID: %w", err)
	}

	found := make(
		vibe.GeneratedPlaylist,
		0,
		len(playlist),
	)
	seen := make(map[string]bool, len(playlist))
	unresolvedCandidates := make(
		vibe.GeneratedPlaylist,
		0,
		generatedPlaylistFallbackSearchLimit,
	)
	for _, candidate := range playlist {
		track, ok := tracksByID[candidate.YouTubeID]
		if !ok || seen[track.YouTubeID] {
			if candidate.Title != "" &&
				candidate.Artist != "" &&
				len(unresolvedCandidates) < generatedPlaylistFallbackSearchLimit {
				unresolvedCandidates = append(unresolvedCandidates, candidate)
			}
			continue
		}

		seen[track.YouTubeID] = true
		found = append(found, track)
	}
	if len(found) >= vibe.GeneratedPlaylistSelectedTrackCount {
		unresolvedCandidates = unresolvedCandidates[:0]
	}
	remainingTrackCount := vibe.GeneratedPlaylistSelectedTrackCount - len(found)
	if remainingTrackCount > 0 && len(unresolvedCandidates) > remainingTrackCount {
		unresolvedCandidates = unresolvedCandidates[:remainingTrackCount]
	}

	fallbackIDs := make([]string, 0, len(unresolvedCandidates))
	fallbackIDGroups := make([][]string, 0, len(unresolvedCandidates))
	var fallbackSearchErr error
	for _, candidate := range unresolvedCandidates {
		var result *searchResponse
		result, err = c.searchVideos(
			ctx,
			candidate.Artist+" "+candidate.Title,
			generatedTrackSearchResults,
		)
		if err != nil {
			var quotaError internalerror.ErrProviderQuotaExceeded
			if errors.As(err, &quotaError) {
				fallbackSearchErr = err
				break
			}
			return nil, fmt.Errorf("error searching generated track: %w", err)
		}
		candidateIDs := make([]string, 0, len(result.Items))
		for _, item := range result.Items {
			if item.ID.VideoID != "" {
				fallbackIDs = append(fallbackIDs, item.ID.VideoID)
				candidateIDs = append(candidateIDs, item.ID.VideoID)
			}
		}
		fallbackIDGroups = append(fallbackIDGroups, candidateIDs)
	}

	fallbackTracksByID, err := c.getGeneratedTracks(ctx, fallbackIDs)
	if err != nil {
		return nil, fmt.Errorf("error getting fallback generated tracks: %w", err)
	}
	for _, candidateIDs := range fallbackIDGroups {
		for _, youtubeID := range candidateIDs {
			track, ok := fallbackTracksByID[youtubeID]
			if !ok || seen[track.YouTubeID] {
				continue
			}

			seen[track.YouTubeID] = true
			found = append(found, track)
			break
		}
	}

	if len(found) == 0 {
		if fallbackSearchErr != nil {
			return nil, fmt.Errorf("error searching generated playlist: %w", fallbackSearchErr)
		}
		return nil, fmt.Errorf("error no generated songs found on youtube")
	}

	slices.SortStableFunc(found, func(a, b vibe.GeneratedTrack) int {
		viewComparison := cmp.Compare(b.ViewCount, a.ViewCount)
		if viewComparison != 0 {
			return viewComparison
		}
		return cmp.Compare(b.LikeCount, a.LikeCount)
	})
	if len(found) > vibe.GeneratedPlaylistSelectedTrackCount {
		found = found[:vibe.GeneratedPlaylistSelectedTrackCount]
	}

	return &found, nil
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

	for start := 0; start < len(youtubeIDs); start += youtubeVideoBatchSize {
		end := min(start+youtubeVideoBatchSize, len(youtubeIDs))
		params := url.Values{}
		params.Set("part", "snippet,contentDetails,statistics")
		params.Set("id", strings.Join(youtubeIDs[start:end], ","))
		params.Set(
			"fields",
			"items(id,snippet(title,channelTitle,categoryId,thumbnails),contentDetails/duration,statistics(viewCount,likeCount))",
		)
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
			durationSeconds, err := youtubeDurationSeconds(item.ContentDetails.Duration)
			if err != nil ||
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

			viewCount, _ := strconv.ParseUint(item.Statistics.ViewCount, 10, 64)
			likeCount, _ := strconv.ParseUint(item.Statistics.LikeCount, 10, 64)
			tracks[item.ID] = vibe.GeneratedTrack{
				YouTubeID:    item.ID,
				Title:        html.UnescapeString(item.Snippet.Title),
				Artist:       html.UnescapeString(item.Snippet.ChannelTitle),
				ThumbnailURL: thumbnailURL,
				Duration:     durationSeconds,
				ViewCount:    viewCount,
				LikeCount:    likeCount,
			}
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

const generatedTrackSearchResults = 1

const generatedPlaylistFallbackSearchLimit = 5

const youtubeVideoBatchSize = 50
