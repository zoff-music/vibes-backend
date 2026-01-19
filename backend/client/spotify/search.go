package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/zoff-music/vibes/client"
	"github.com/zoff-music/vibes/vibe"
)

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

// Search searches for tracks on Spotify
func (c *Client) Search(ctx context.Context, query string) ([]vibe.MusicTrack, error) {
	if !c.Enabled {
		return nil, fmt.Errorf("Spotify client is not enabled")
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting access token: %w", err)
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("type", "track")
	params.Set("limit", "10")

	reqData := client.HTTPRequestData{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("%s/search", c.Endpoint),
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Payload: &params,
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error request failed: %w", err)
	}

	var res searchResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
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
