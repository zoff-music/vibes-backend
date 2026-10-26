package youtube

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) SearchGeneratedPlaylist(
	ctx context.Context,
	playlist vibe.GeneratedPlaylist,
	cachedSearches []vibe.CachedProviderSearch,
	roomType vibe.RoomType,
) (*vibe.GeneratedPlaylistSearchResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SearchGeneratedPlaylist")
	defer span.End()

	if !roomType.AllowsSource(vibe.SourceTypeYouTube) {
		return nil, fmt.Errorf("error searching generated playlist: invalid room type %q", roomType)
	}

	if c.apiKey == "" {
		return nil, fmt.Errorf(
			"error validating youtube API key in SearchGeneratedPlaylist: not configured",
		)
	}

	cachedSearchesByQuery := make(
		map[string]vibe.CachedProviderSearch,
		len(cachedSearches),
	)
	for _, search := range cachedSearches {
		cachedSearchesByQuery[search.Query] = search
	}

	youtubeIDs := make([]string, 0, len(playlist))
	seenYouTubeIDs := make(map[string]bool, len(playlist))
	for _, playlistItem := range playlist {
		query := playlistItem.Publisher + " " + playlistItem.Title
		_, cached := cachedSearchesByQuery[query]
		if cached {
			continue
		}
		if playlistItem.YouTubeID == "" || seenYouTubeIDs[playlistItem.YouTubeID] {
			continue
		}
		seenYouTubeIDs[playlistItem.YouTubeID] = true
		youtubeIDs = append(youtubeIDs, playlistItem.YouTubeID)
	}

	itemsByID, err := c.getGeneratedPlaylistItems(ctx, youtubeIDs, roomType)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting generated playlist items by ID in SearchGeneratedPlaylist: %w",
			err,
		)
	}

	found := make(
		vibe.GeneratedPlaylist,
		0,
		len(playlist),
	)
	seen := make(map[string]bool, len(playlist))
	searchesToCache := make(
		[]vibe.CachedProviderSearch,
		0,
		len(playlist),
	)
	searchUsages := make(
		[]vibe.SearchUsage,
		0,
		len(playlist),
	)
	unresolvedCandidates := make(
		vibe.GeneratedPlaylist,
		0,
		generatedPlaylistFallbackSearchLimit,
	)
	for _, candidate := range playlist {
		query := candidate.Publisher + " " + candidate.Title
		cachedSearch, cached := cachedSearchesByQuery[query]
		if cached {
			searchUsages = append(
				searchUsages,
				vibe.GenerateSearchUsage(
					vibe.SourceTypeYouTube,
					query,
					true,
				),
			)
			for _, cachedItem := range cachedSearch.Items {
				if !cachedItem.AllowedInRoom(roomType) {
					continue
				}

				playlistItem, err := cachedItem.ToGeneratedPlaylistItem(query)
				if err != nil {
					return nil, fmt.Errorf("error converting cached generated playlist item: %w", err)
				}

				if playlistItem.Duration <= 0 ||
					(roomType == vibe.RoomTypeMusic && playlistItem.Duration > generatedItemMaxDurationSeconds) ||
					playlistItem.PlaybackRestriction == vibe.PlaybackRestrictionAge ||
					playlistItem.PlaybackRestriction == vibe.PlaybackRestrictionEmbedding ||
					seen[playlistItem.YouTubeID] {
					continue
				}

				seen[playlistItem.YouTubeID] = true
				found = append(found, *playlistItem)
				break
			}
			continue
		}

		playlistItem, ok := itemsByID[candidate.YouTubeID]
		if !ok || seen[playlistItem.YouTubeID] {
			if candidate.Title != "" &&
				candidate.Publisher != "" &&
				len(unresolvedCandidates) < generatedPlaylistFallbackSearchLimit {
				unresolvedCandidates = append(unresolvedCandidates, candidate)
			}
			continue
		}

		playlistItem.SearchQuery = query
		seen[playlistItem.YouTubeID] = true
		found = append(found, playlistItem)
		providerItem, err := playlistItem.ToProviderItem()
		if err != nil {
			return nil, fmt.Errorf("error converting generated playlist item for cache: %w", err)
		}

		searchesToCache = append(searchesToCache, vibe.CachedProviderSearch{
			Resolution: true,
			Query:      query,
			Items: []vibe.ProviderItem{
				*providerItem,
			},
		})
	}
	if len(found) >= c.generatedPlaylistSelectedCount {
		unresolvedCandidates = unresolvedCandidates[:0]
	}
	remainingItemCount := c.generatedPlaylistSelectedCount - len(found)
	if remainingItemCount > 0 && len(unresolvedCandidates) > remainingItemCount {
		unresolvedCandidates = unresolvedCandidates[:remainingItemCount]
	}

	slices.SortStableFunc(found, func(a, b vibe.GeneratedPlaylistItem) int {
		viewComparison := cmp.Compare(b.ViewCount, a.ViewCount)
		if viewComparison != 0 {
			return viewComparison
		}
		likeComparison := cmp.Compare(b.LikeCount, a.LikeCount)

		return likeComparison
	})
	if len(found) > c.generatedPlaylistSelectedCount {
		found = found[:c.generatedPlaylistSelectedCount]
	}

	result := vibe.GeneratedPlaylistSearchResult{
		Playlist:       found,
		Unresolved:     unresolvedCandidates,
		CachedSearches: searchesToCache,
		SearchUsages:   searchUsages,
	}

	return &result, nil
}

func (c *Client) getGeneratedPlaylistItems(
	ctx context.Context,
	youtubeIDs []string,
	roomType vibe.RoomType,
) (map[string]vibe.GeneratedPlaylistItem, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getGeneratedPlaylistItems")
	defer span.End()

	playlistItems := make(map[string]vibe.GeneratedPlaylistItem, len(youtubeIDs))
	if len(youtubeIDs) == 0 {
		return playlistItems, nil
	}

	for start := 0; start < len(youtubeIDs); start += youtubeVideoBatchSize {
		end := min(start+youtubeVideoBatchSize, len(youtubeIDs))
		params := url.Values{}
		params.Set("part", "snippet,contentDetails,statistics,status")
		params.Set("id", strings.Join(youtubeIDs[start:end], ","))
		params.Set(
			"fields",
			"items(id,snippet(title,channelTitle,categoryId,liveBroadcastContent,thumbnails),contentDetails(duration,contentRating/ytRating,regionRestriction),statistics(viewCount,likeCount),status(embeddable,madeForKids))",
		)
		params.Set("key", c.apiKey)

		responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
			Method:  http.MethodGet,
			URL:     fmt.Sprintf("%s/videos", c.Endpoint),
			Payload: &params,
		})
		if err != nil {
			return nil, fmt.Errorf(
				"error requesting youtube generated playlist item details in getGeneratedPlaylistItems: %w",
				err,
			)
		}

		var response videoResponse
		err = json.Unmarshal(responseBody, &response)
		if err != nil {
			return nil, fmt.Errorf(
				"error unmarshaling youtube generated playlist item details in getGeneratedPlaylistItems: %w",
				err,
			)
		}

		for _, item := range response.Items {
			if item.isLiveVideo() || item.Status.MadeForKids {
				continue
			}
			playbackRestriction := item.playbackRestriction()
			durationSeconds, err := youtubeDurationSeconds(item.ContentDetails.Duration)
			if err != nil ||
				(roomType == vibe.RoomTypeMusic && item.Snippet.CategoryID != youtubeMusicCategoryID) ||
				playbackRestriction == vibe.PlaybackRestrictionAge ||
				playbackRestriction == vibe.PlaybackRestrictionEmbedding ||
				durationSeconds <= 0 ||
				(roomType == vibe.RoomTypeMusic && durationSeconds > generatedItemMaxDurationSeconds) {
				continue
			}

			thumbnailURL := item.Snippet.Thumbnails.High.URL
			if thumbnailURL == "" {
				thumbnailURL = item.Snippet.Thumbnails.Medium.URL
			}
			if thumbnailURL == "" {
				thumbnailURL = item.Snippet.Thumbnails.Default.URL
			}

			viewCount, err := strconv.ParseUint(item.Statistics.ViewCount, 10, 64)
			if err != nil {
				viewCount = 0
			}
			likeCount, err := strconv.ParseUint(item.Statistics.LikeCount, 10, 64)
			if err != nil {
				likeCount = 0
			}
			playlistItems[item.ID] = vibe.GeneratedPlaylistItem{
				CategoryID:          item.Snippet.CategoryID,
				YouTubeID:           item.ID,
				Title:               html.UnescapeString(item.Snippet.Title),
				Publisher:           html.UnescapeString(item.Snippet.ChannelTitle),
				ThumbnailURL:        thumbnailURL,
				Duration:            durationSeconds,
				ViewCount:           viewCount,
				LikeCount:           likeCount,
				PlaybackRestriction: playbackRestriction,
			}
		}
	}

	return playlistItems, nil
}

func youtubeDurationSeconds(value string) (int, error) {
	if value == youtubeZeroDuration {
		return 0, nil
	}
	if !strings.HasPrefix(value, "P") {
		return 0, fmt.Errorf(
			"error validating youtube duration in youtubeDurationSeconds: unsupported value %q",
			value,
		)
	}

	duration := strings.TrimPrefix(value, "P")
	number := ""
	totalSeconds := 0
	inTime := false
	lastUnit := 0
	for _, character := range duration {
		if character == 'T' && !inTime && number == "" {
			inTime = true
			continue
		}

		if character >= '0' && character <= '9' {
			number += string(character)
			continue
		}
		if number == "" {
			return 0, fmt.Errorf(
				"error validating youtube duration in youtubeDurationSeconds: invalid value %q",
				value,
			)
		}

		amount, err := strconv.Atoi(number)
		if err != nil {
			return 0, fmt.Errorf(
				"error parsing youtube duration in youtubeDurationSeconds for %q: %w",
				value,
				err,
			)
		}
		unit := 0
		multiplier := 0
		switch character {
		case 'D':
			if !inTime {
				unit = 1
				multiplier = 24 * 60 * 60
			}
		case 'H':
			if inTime {
				unit = 2
				multiplier = 60 * 60
			}
		case 'M':
			if inTime {
				unit = 3
				multiplier = 60
			}
		case 'S':
			if inTime {
				unit = 4
				multiplier = 1
			}
		}

		if unit <= lastUnit {
			return 0, fmt.Errorf(
				"error validating youtube duration in youtubeDurationSeconds: invalid unit %q",
				character,
			)
		}

		maximumInt := int(^uint(0) >> 1)
		if amount > (maximumInt-totalSeconds)/multiplier {
			return 0, fmt.Errorf("error validating youtube duration in youtubeDurationSeconds: duration exceeds integer range")
		}

		totalSeconds += amount * multiplier
		lastUnit = unit
		number = ""
	}
	if number != "" || lastUnit == 0 || strings.HasSuffix(value, "T") {
		return 0, fmt.Errorf(
			"error validating youtube duration in youtubeDurationSeconds: incomplete value %q",
			value,
		)
	}

	return totalSeconds, nil
}

const youtubeMusicCategoryID = "10"
const youtubeZeroDuration = "P0D"
const generatedItemMaxDurationSeconds = 20 * 60
const generatedPlaylistFallbackSearchLimit = 5
const youtubeVideoBatchSize = 50
