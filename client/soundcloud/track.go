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

// GetProviderItem fetches details for a specific track ID
func (c *Client) GetProviderItem(ctx context.Context, id string) (*vibe.ProviderItem, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetProviderItem")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf(
			"error validating soundcloud client in GetProviderItem: client is not enabled",
		)
	}

	// Ensure valid access token
	err := c.EnsureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error ensuring token in GetProviderItem: %w", err)
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
				return nil, internalerror.ErrProviderItemNotFound{
					Err: fmt.Errorf(
						"error getting soundcloud track in GetProviderItem: track %s not found",
						id,
					),
				}
			}
			if statusCodeError.StatusCode == http.StatusTooManyRequests {
				return nil, internalerror.ErrProviderQuotaExceeded{
					Err: fmt.Errorf(
						"error requesting soundcloud track in GetProviderItem: %w",
						err,
					),
					Provider: string(vibe.SourceTypeSoundCloud),
				}
			}
		}

		return nil, fmt.Errorf(
			"error requesting soundcloud track in GetProviderItem: %w",
			err,
		)
	}

	var res trackResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding soundcloud response in GetProviderItem: %w",
			err,
		)
	}
	if res.ID == 0 || res.Title == "" || res.PermalinkURL == "" {
		return nil, internalerror.ErrProviderItemNotFound{
			Err: fmt.Errorf(
				"error getting soundcloud track in GetProviderItem: response did not contain a track",
			),
		}
	}

	track, err := res.toProviderItem()
	if err != nil {
		return nil, fmt.Errorf("error converting soundcloud track in GetProviderItem: %w", err)
	}

	return track, nil
}

func (c *Client) ResolveProviderItem(
	ctx context.Context,
	providerURL string,
) (*vibe.ProviderItem, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ResolveProviderItem")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf(
			"error validating soundcloud client in ResolveProviderItem: client is not enabled",
		)
	}

	err := c.EnsureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error ensuring token in ResolveProviderItem: %w", err)
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
				return nil, internalerror.ErrProviderItemNotFound{
					Err: fmt.Errorf(
						"error resolving soundcloud track in ResolveProviderItem: track not found",
					),
				}
			}
			if statusCodeError.StatusCode == http.StatusTooManyRequests {
				return nil, internalerror.ErrProviderQuotaExceeded{
					Err: fmt.Errorf(
						"error requesting soundcloud track in ResolveProviderItem: %w",
						err,
					),
					Provider: string(vibe.SourceTypeSoundCloud),
				}
			}
		}

		return nil, fmt.Errorf(
			"error requesting soundcloud track in ResolveProviderItem: %w",
			err,
		)
	}

	var res trackResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding soundcloud response in ResolveProviderItem: %w",
			err,
		)
	}
	if res.ID == 0 || res.Title == "" || res.PermalinkURL == "" {
		return nil, internalerror.ErrProviderItemNotFound{
			Err: fmt.Errorf(
				"error resolving soundcloud track in ResolveProviderItem: URL did not resolve to a track",
			),
		}
	}

	track, err := res.toProviderItem()
	if err != nil {
		return nil, fmt.Errorf("error converting soundcloud track in ResolveProviderItem: %w", err)
	}

	return track, nil
}
