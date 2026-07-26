package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetTrack fetches details for a specific track ID
func (c *Client) GetTrack(ctx context.Context, id string) (*vibe.MusicTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetTrack")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf(
			"error validating spotify client in GetTrack: client is not enabled",
		)
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting access token in GetTrack: %w", err)
	}

	reqData := client.HTTPRequestData{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("%s/tracks/%s", c.Endpoint, id),
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
			"Accept":        "application/json",
		},
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		var statusCodeError client.HTTPStatusCodeError
		if errors.As(err, &statusCodeError) {
			if statusCodeError.StatusCode == http.StatusNotFound {
				return nil, internalerror.ErrMusicTrackNotFound{
					Err: fmt.Errorf(
						"error getting spotify track in GetTrack: track %s not found",
						id,
					),
				}
			}
			if statusCodeError.StatusCode == http.StatusTooManyRequests {
				return nil, internalerror.ErrProviderQuotaExceeded{
					Err: fmt.Errorf(
						"error requesting spotify track in GetTrack: %w",
						err,
					),
					Provider: string(vibe.SourceTypeSpotify),
				}
			}
		}

		return nil, fmt.Errorf("error requesting spotify track in GetTrack: %w", err)
	}

	var item spotifyTrack
	err = json.Unmarshal(resp, &item)
	if err != nil {
		return nil, fmt.Errorf("error decoding spotify response in GetTrack: %w", err)
	}

	artists := make([]string, 0, len(item.Artists))
	for _, a := range item.Artists {
		artists = append(artists, a.Name)
	}

	thumbnail := ""
	if len(item.Album.Images) > 0 {
		thumbnail = item.Album.Images[0].URL
	}

	return &vibe.MusicTrack{
		ID:              item.ID,
		Source:          vibe.SourceTypeSpotify,
		ProviderURL:     fmt.Sprintf("https://open.spotify.com/track/%s", item.ID),
		Title:           item.Name,
		ChannelTitle:    strings.Join(artists, ", "),
		ThumbnailURL:    thumbnail,
		Duration:        fmt.Sprintf("PT%dM%dS", (item.DurationMS/1000)/60, (item.DurationMS/1000)%60),
		DurationSeconds: item.DurationMS / 1000,
	}, nil
}
