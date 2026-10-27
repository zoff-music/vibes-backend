package soundcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

// SearchProviderItems searches for tracks on SoundCloud
func (c *Client) SearchProviderItems(ctx context.Context, query string, roomType vibe.RoomType) ([]vibe.ProviderItem, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SearchProviderItems")
	defer span.End()

	if !roomType.AllowsSource(vibe.SourceTypeSoundCloud) {
		return nil, fmt.Errorf("error searching soundcloud: provider is unavailable for %s", roomType)
	}

	if !c.Enabled {
		return nil, fmt.Errorf(
			"error validating soundcloud client in SearchProviderItems: client is not enabled",
		)
	}

	// Ensure valid access token
	err := c.EnsureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error ensuring token in SearchProviderItems: %w", err)
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", "25")
	params.Set("access", "playable")

	reqData := client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/tracks", c.Endpoint),
		Payload: &params,
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("OAuth %s", c.accessToken),
			"Accept":        "application/json; charset=utf-8",
		},
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		var statusError internalerror.HTTPStatusCodeError
		if errors.As(err, &statusError) && statusError.StatusCode == http.StatusTooManyRequests {
			return nil, internalerror.ErrProviderQuotaExceeded{
				Err:      fmt.Errorf("error requesting soundcloud search: %w", err),
				Provider: vibe.SourceTypeSoundCloud,
			}
		}

		return nil, fmt.Errorf(
			"error requesting soundcloud tracks in SearchProviderItems: %w",
			err,
		)
	}

	var results []trackResponse
	err = json.Unmarshal(resp, &results)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding soundcloud response in SearchProviderItems: %w",
			err,
		)
	}

	tracks := make([]vibe.ProviderItem, 0, len(results))
	for _, res := range results {
		if res.Access != "playable" || res.EmbeddableBy != "all" {
			continue
		}

		track, err := res.toProviderItem()
		if err != nil {
			return nil, fmt.Errorf("error converting soundcloud track in SearchProviderItems: %w", err)
		}

		tracks = append(tracks, *track)
	}

	return tracks, nil
}

type trackResponse struct {
	Access       string  `json:"access"`
	EmbeddableBy string  `json:"embeddable_by"`
	ID           int     `json:"id"`
	Title        string  `json:"title"`
	User         *user   `json:"user"`
	ArtworkURL   *string `json:"artwork_url"`
	Duration     int     `json:"duration"`
	PermalinkURL string  `json:"permalink_url"`
	StreamURL    string  `json:"stream_url"`
	Description  string  `json:"description"`
}

type user struct {
	Username string `json:"username"`
}

func (r trackResponse) toProviderItem() (*vibe.ProviderItem, error) {
	playbackRestriction := ""
	if r.Access != "playable" || r.EmbeddableBy != "all" {
		playbackRestriction = vibe.PlaybackRestrictionEmbedding
	}

	username := "Unknown"
	if r.User != nil {
		username = r.User.Username
	}

	artworkURL := ""
	if r.ArtworkURL != nil {
		artworkURL = *r.ArtworkURL
	}

	return &vibe.ProviderItem{
		PlaybackRestriction: playbackRestriction,
		ID:                  fmt.Sprintf("%d", r.ID),
		Source:              vibe.SourceTypeSoundCloud,
		ProviderURL:         r.PermalinkURL,
		Title:               r.Title,
		Publisher:           username,
		ThumbnailURL:        artworkURL,
		Duration:            fmt.Sprintf("PT%dM%dS", (r.Duration/1000)/60, (r.Duration/1000)%60),
		DurationSeconds:     r.Duration / 1000,
	}, nil
}
