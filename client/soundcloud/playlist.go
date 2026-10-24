package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) ResolveProviderPlaylist(
	ctx context.Context,
	providerURL string,
) (*vibe.ProviderPlaylist, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ResolveProviderPlaylist")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf(
			"error validating soundcloud client in ResolveProviderPlaylist: client is not enabled",
		)
	}

	err := c.EnsureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error ensuring token in ResolveProviderPlaylist: %w", err)
	}

	params := url.Values{}
	params.Set("url", providerURL)
	responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/resolve", c.Endpoint),
		Payload: &params,
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("OAuth %s", c.accessToken),
			"Accept":        "application/json; charset=utf-8",
		},
	})
	if err != nil {
		return nil, fmt.Errorf(
			"error resolving soundcloud playlist in ResolveProviderPlaylist: %w",
			err,
		)
	}

	var resolved soundCloudPlaylistResponse
	err = json.Unmarshal(responseBody, &resolved)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding soundcloud playlist in ResolveProviderPlaylist: %w",
			err,
		)
	}
	if resolved.Kind == "" {
		return nil, fmt.Errorf("error resolving soundcloud playlist in ResolveProviderPlaylist: resource kind is missing")
	}

	if resolved.Kind != "playlist" && resolved.Kind != "system-playlist" {
		return nil, internalerror.ErrProviderPlaylistInvalid{
			Err: fmt.Errorf("error resolving soundcloud playlist in ResolveProviderPlaylist: URL did not resolve to a playlist"),
		}
	}

	if resolved.URN == "" {
		return nil, fmt.Errorf(
			"error resolving soundcloud playlist in ResolveProviderPlaylist: playlist identifier is missing",
		)
	}

	playlistItems := make([]vibe.ProviderItem, 0)
	seenTrackIDs := make(map[int]bool)
	seenPages := make(map[string]bool)
	truncated := false
	nextURL := fmt.Sprintf("%s/playlists/%s/tracks", c.Endpoint, url.PathEscape(resolved.URN))
	firstPage := true
	for nextURL != "" {
		if seenPages[nextURL] || len(seenPages) >= playlistItemLimit/soundCloudPlaylistPageSize {
			truncated = true
			break
		}
		seenPages[nextURL] = true
		requestData := client.HTTPRequestData{
			Method: http.MethodGet,
			URL:    nextURL,
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("OAuth %s", c.accessToken),
				"Accept":        "application/json; charset=utf-8",
			},
		}
		if firstPage {
			pageParams := url.Values{}
			pageParams.Set("access", "playable,preview")
			pageParams.Set("limit", fmt.Sprintf("%d", soundCloudPlaylistPageSize))
			pageParams.Set("linked_partitioning", "true")
			requestData.Payload = &pageParams
			firstPage = false
		}

		responseBody, err = c.HTTPClient.RequestBytes(ctx, requestData)
		if err != nil {
			return nil, fmt.Errorf(
				"error requesting soundcloud playlist items in ResolveProviderPlaylist: %w",
				err,
			)
		}

		var page soundCloudPlaylistTracksResponse
		err = json.Unmarshal(responseBody, &page)
		if err != nil {
			return nil, fmt.Errorf(
				"error decoding soundcloud playlist items in ResolveProviderPlaylist: %w",
				err,
			)
		}
		trackIDs := make([]int, 0, len(page.Collection))
		for _, item := range page.Collection {
			if item.ID == 0 || item.Title == "" || item.PermalinkURL == "" || seenTrackIDs[item.ID] {
				continue
			}
			if len(playlistItems)+len(trackIDs) == playlistItemLimit {
				truncated = true
				break
			}
			trackIDs = append(trackIDs, item.ID)
			seenTrackIDs[item.ID] = true
		}

		tracks, err := c.getTracks(ctx, trackIDs)
		if err != nil {
			return nil, fmt.Errorf("error fetching soundcloud playlist metadata in ResolveProviderPlaylist: %w", err)
		}

		for _, track := range tracks {
			providerItem, err := track.toProviderItem()
			if err != nil {
				return nil, fmt.Errorf("error converting soundcloud playlist item: %w", err)
			}

			playlistItems = append(playlistItems, *providerItem)
		}
		if truncated {
			break
		}
		nextURL = page.NextHref
	}

	return &vibe.ProviderPlaylist{
		ID:        resolved.URN,
		Source:    vibe.SourceTypeSoundCloud,
		Title:     resolved.Title,
		Items:     playlistItems,
		Truncated: truncated,
	}, nil
}

type soundCloudPlaylistResponse struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	URN   string `json:"urn"`
}

type soundCloudPlaylistTracksResponse struct {
	Collection []trackResponse `json:"collection"`
	NextHref   string          `json:"next_href"`
}

const soundCloudPlaylistPageSize = 200

const playlistItemLimit = 5000
