package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) GetPlaylist(
	ctx context.Context,
	id string,
) (*vibe.MusicPlaylist, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetPlaylist")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf(
			"error validating spotify client in GetPlaylist: client is not enabled",
		)
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting access token in GetPlaylist: %w", err)
	}

	tracks := make([]*vibe.MusicTrack, 0)
	offset := 0
	truncated := false
	for {
		params := url.Values{}
		params.Set("limit", strconv.Itoa(spotifyPlaylistPageSize))
		params.Set("offset", strconv.Itoa(offset))
		responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
			Method: http.MethodGet,
			URL:    fmt.Sprintf("%s/playlists/%s/items", c.Endpoint, id),
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Accept":        "application/json",
			},
			Payload: &params,
		})
		if err != nil {
			return nil, fmt.Errorf(
				"error requesting spotify playlist items in GetPlaylist: %w",
				err,
			)
		}

		var page spotifyPlaylistItemsResponse
		err = json.Unmarshal(responseBody, &page)
		if err != nil {
			return nil, fmt.Errorf(
				"error decoding spotify playlist items in GetPlaylist: %w",
				err,
			)
		}

		for _, playlistItem := range page.Items {
			item := playlistItem.MusicTrack()
			if item == nil || item.ID == "" {
				continue
			}
			if len(tracks) == playlistTrackLimit {
				truncated = true
				break
			}

			artists := make([]string, 0, len(item.Artists))
			for _, artist := range item.Artists {
				artists = append(artists, artist.Name)
			}
			thumbnailURL := ""
			if len(item.Album.Images) > 0 {
				thumbnailURL = item.Album.Images[0].URL
			}

			tracks = append(tracks, &vibe.MusicTrack{
				ID:              item.ID,
				Source:          vibe.SourceTypeSpotify,
				ProviderURL:     fmt.Sprintf("https://open.spotify.com/track/%s", item.ID),
				Title:           item.Name,
				ChannelTitle:    strings.Join(artists, ", "),
				ThumbnailURL:    thumbnailURL,
				Duration:        fmt.Sprintf("PT%dM%dS", (item.DurationMS/1000)/60, (item.DurationMS/1000)%60),
				DurationSeconds: item.DurationMS / 1000,
			})
		}
		if truncated || page.Next == "" || len(page.Items) == 0 {
			break
		}
		offset += len(page.Items)
	}

	return &vibe.MusicPlaylist{
		ID:        id,
		Source:    vibe.SourceTypeSpotify,
		Tracks:    tracks,
		Truncated: truncated,
	}, nil
}

type spotifyPlaylistItemsResponse struct {
	Items []*spotifyPlaylistItem `json:"items"`
	Next  string                 `json:"next"`
}

type spotifyPlaylistItem struct {
	Item  *spotifyTrack `json:"item"`
	Track *spotifyTrack `json:"track"`
}

func (i spotifyPlaylistItem) MusicTrack() *spotifyTrack {
	if i.Item != nil {
		return i.Item
	}

	return i.Track
}

const spotifyPlaylistPageSize = 50

const playlistTrackLimit = 500
