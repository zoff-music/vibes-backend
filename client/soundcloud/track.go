package soundcloud

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetTrack fetches details for a specific track ID
func (c *Client) GetTrack(ctx context.Context, id string) (*vibe.MusicTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetTrack")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf(
			"error validating soundcloud client in GetTrack: client is not enabled",
		)
	}

	// Ensure valid access token
	err := c.EnsureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error ensuring token in GetTrack: %w", err)
	}

	reqData := client.HTTPRequestData{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("%s/tracks/%s", c.Endpoint, id),
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("OAuth %s", c.accessToken),
			"Accept":        "application/json; charset=utf-8",
		},
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		var statusCodeError client.HTTPStatusCodeError
		if errors.As(err, &statusCodeError) {
			if statusCodeError.StatusCode == http.StatusNotFound {
				return nil, internalerror.ErrMusicTrackNotFound{
					Err: fmt.Errorf(
						"error getting soundcloud track in GetTrack: track %s not found",
						id,
					),
				}
			}
			if statusCodeError.StatusCode == http.StatusTooManyRequests {
				return nil, internalerror.ErrProviderQuotaExceeded{
					Err: fmt.Errorf(
						"error requesting soundcloud track in GetTrack: %w",
						err,
					),
					Provider: string(vibe.SourceTypeSoundCloud),
				}
			}
		}

		return nil, fmt.Errorf(
			"error requesting soundcloud track in GetTrack: %w",
			err,
		)
	}

	var res trackResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding soundcloud response in GetTrack: %w",
			err,
		)
	}
	if res.ID == 0 || res.Title == "" || res.PermalinkURL == "" {
		return nil, internalerror.ErrMusicTrackNotFound{
			Err: fmt.Errorf(
				"error getting soundcloud track in GetTrack: response did not contain a track",
			),
		}
	}

	track := res.MusicTrack()
	return &track, nil
}

func (c *Client) ResolveTrack(
	ctx context.Context,
	providerURL string,
) (*vibe.MusicTrack, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ResolveTrack")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf(
			"error validating soundcloud client in ResolveTrack: client is not enabled",
		)
	}

	err := c.EnsureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error ensuring token in ResolveTrack: %w", err)
	}

	params := url.Values{}
	params.Set("url", providerURL)
	reqData := client.HTTPRequestData{
		Method:  http.MethodGet,
		URL:     fmt.Sprintf("%s/resolve", c.Endpoint),
		Payload: &params,
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("OAuth %s", c.accessToken),
			"Accept":        "application/json; charset=utf-8",
		},
	}

	resp, err := c.HTTPClient.RequestBytes(ctx, reqData)
	if err != nil {
		var statusCodeError client.HTTPStatusCodeError
		if errors.As(err, &statusCodeError) {
			if statusCodeError.StatusCode == http.StatusNotFound {
				return nil, internalerror.ErrMusicTrackNotFound{
					Err: fmt.Errorf(
						"error resolving soundcloud track in ResolveTrack: track not found",
					),
				}
			}
			if statusCodeError.StatusCode == http.StatusTooManyRequests {
				return nil, internalerror.ErrProviderQuotaExceeded{
					Err: fmt.Errorf(
						"error requesting soundcloud track in ResolveTrack: %w",
						err,
					),
					Provider: string(vibe.SourceTypeSoundCloud),
				}
			}
		}

		return nil, fmt.Errorf(
			"error requesting soundcloud track in ResolveTrack: %w",
			err,
		)
	}

	var res trackResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding soundcloud response in ResolveTrack: %w",
			err,
		)
	}
	if res.ID == 0 || res.Title == "" || res.PermalinkURL == "" {
		return nil, internalerror.ErrMusicTrackNotFound{
			Err: fmt.Errorf(
				"error resolving soundcloud track in ResolveTrack: URL did not resolve to a track",
			),
		}
	}

	track := res.MusicTrack()
	return &track, nil
}
