package soundcloud

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes-backend/client"
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
	if resolved.URN == "" || (resolved.Kind != "playlist" && resolved.Kind != "system-playlist") {
		return nil, fmt.Errorf(
			"error resolving soundcloud playlist in ResolveProviderPlaylist: URL did not resolve to a playlist",
		)
	}

	playlistItems := make([]vibe.ProviderItem, 0)
	truncated := false
	nextURL := fmt.Sprintf("%s/playlists/%s/tracks", c.Endpoint, url.PathEscape(resolved.URN))
	firstPage := true
	for nextURL != "" {
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
		for _, item := range page.Collection {
			if item.ID == 0 || item.Title == "" || item.PermalinkURL == "" {
				continue
			}
			if len(playlistItems) == playlistItemLimit {
				truncated = true
				break
			}
			providerItem, err := item.toProviderItem()
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

const playlistItemLimit = 500
