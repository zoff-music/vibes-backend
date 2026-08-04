package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) ResolvePlaylist(
	ctx context.Context,
	providerURL string,
) (*vibe.MusicPlaylist, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ResolvePlaylist")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf(
			"error validating soundcloud client in ResolvePlaylist: client is not enabled",
		)
	}

	err := c.EnsureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error ensuring token in ResolvePlaylist: %w", err)
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
			"error resolving soundcloud playlist in ResolvePlaylist: %w",
			err,
		)
	}

	var resolved soundCloudPlaylistResponse
	err = json.Unmarshal(responseBody, &resolved)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding soundcloud playlist in ResolvePlaylist: %w",
			err,
		)
	}
	if resolved.URN == "" || (resolved.Kind != "playlist" && resolved.Kind != "system-playlist") {
		return nil, fmt.Errorf(
			"error resolving soundcloud playlist in ResolvePlaylist: URL did not resolve to a playlist",
		)
	}

	tracks := make([]*vibe.MusicTrack, 0)
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
				"error requesting soundcloud playlist tracks in ResolvePlaylist: %w",
				err,
			)
		}

		var page soundCloudPlaylistTracksResponse
		err = json.Unmarshal(responseBody, &page)
		if err != nil {
			return nil, fmt.Errorf(
				"error decoding soundcloud playlist tracks in ResolvePlaylist: %w",
				err,
			)
		}
		for _, item := range page.Collection {
			if item == nil || item.ID == 0 || item.Title == "" || item.PermalinkURL == "" {
				continue
			}
			if len(tracks) == playlistTrackLimit {
				truncated = true
				break
			}
			track := item.MusicTrack()
			tracks = append(tracks, &track)
		}
		if truncated {
			break
		}
		nextURL = page.NextHref
	}

	return &vibe.MusicPlaylist{
		ID:        resolved.URN,
		Source:    vibe.SourceTypeSoundCloud,
		Title:     resolved.Title,
		Tracks:    tracks,
		Truncated: truncated,
	}, nil
}

type soundCloudPlaylistResponse struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	URN   string `json:"urn"`
}

type soundCloudPlaylistTracksResponse struct {
	Collection []*trackResponse `json:"collection"`
	NextHref   string           `json:"next_href"`
}

const soundCloudPlaylistPageSize = 200

const playlistTrackLimit = 500
