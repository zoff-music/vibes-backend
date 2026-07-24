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

// Search searches for tracks on Spotify
func (c *Client) Search(ctx context.Context, query string) ([]vibe.MusicTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "Search")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf(
			"error validating spotify client in Search: client is not enabled",
		)
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting access token in Search: %w", err)
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("type", "track")
	params.Set("limit", strconv.Itoa(spotifySearchResultCount))

	reqData := client.HTTPRequestData{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("%s/search", c.Endpoint),
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
			"Accept":        "application/json",
		},
		Payload: &params,
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error requesting spotify search in Search: %w", err)
	}

	var res searchResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf("error decoding spotify response in Search: %w", err)
	}

	tracks := make([]vibe.MusicTrack, 0, len(res.Tracks.Items))
	for _, item := range res.Tracks.Items {
		artists := make([]string, 0, len(item.Artists))
		for _, a := range item.Artists {
			artists = append(artists, a.Name)
		}

		thumbnail := ""
		if len(item.Album.Images) > 0 {
			thumbnail = item.Album.Images[0].URL
		}

		tracks = append(tracks, vibe.MusicTrack{
			ID:           item.ID,
			Source:       vibe.SourceTypeSpotify,
			Title:        item.Name,
			ChannelTitle: strings.Join(artists, ", "),
			ThumbnailURL: thumbnail,
			Duration:     fmt.Sprintf("PT%dM%dS", (item.DurationMS/1000)/60, (item.DurationMS/1000)%60),
		})
	}

	return tracks, nil
}

type searchResponse struct {
	Tracks searchTracks `json:"tracks"`
}

type searchTracks struct {
	Items []spotifyTrack `json:"items"`
}

type spotifyTrack struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Artists    []spotifyArtist `json:"artists"`
	Album      spotifyAlbum    `json:"album"`
	DurationMS int             `json:"duration_ms"`
}

type spotifyArtist struct {
	Name string `json:"name"`
}

type spotifyAlbum struct {
	Images []spotifyImage `json:"images"`
}

type spotifyImage struct {
	URL string `json:"url"`
}

const spotifySearchResultCount = 10
