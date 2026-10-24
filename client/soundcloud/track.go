package soundcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetProviderItem fetches details for a specific track ID
func (c *Client) GetProviderItem(ctx context.Context, id string) (*vibe.ProviderItem, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetProviderItem")
	defer span.End()

	trackID, err := strconv.Atoi(strings.TrimPrefix(id, "soundcloud:tracks:"))
	if err != nil {
		return nil, internalerror.ErrProviderItemNotFound{
			Err: fmt.Errorf("error parsing soundcloud track ID in GetProviderItem: %w", err),
		}
	}
	if trackID <= 0 {
		return nil, internalerror.ErrProviderItemNotFound{
			Err: fmt.Errorf("error validating soundcloud track ID in GetProviderItem"),
		}
	}

	tracks, err := c.getTracks(ctx, []int{trackID})
	if err != nil {
		return nil, fmt.Errorf("error fetching soundcloud collection in GetProviderItem: %w", err)
	}
	if len(tracks) == 0 {
		return nil, internalerror.ErrProviderItemNotFound{
			Err: fmt.Errorf("error getting soundcloud track in GetProviderItem: track not found"),
		}
	}

	track, err := tracks[0].toProviderItem()
	if err != nil {
		return nil, fmt.Errorf("error converting soundcloud track in GetProviderItem: %w", err)
	}

	return track, nil
}

// getTracks uses collection metadata; single-track and resolve responses can report preview lengths.
func (c *Client) getTracks(ctx context.Context, ids []int) ([]trackResponse, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getTracks")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf("error validating soundcloud client in getTracks: client is not enabled")
	}

	err := c.EnsureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error ensuring token in getTracks: %w", err)
	}

	tracks := make([]trackResponse, 0, len(ids))
	for start := 0; start < len(ids); start += soundCloudTrackBatchSize {
		batch := ids[start:min(start+soundCloudTrackBatchSize, len(ids))]
		urns := make([]string, 0, len(batch))
		requested := make(map[int]bool, len(batch))
		for _, id := range batch {
			if id <= 0 || requested[id] {
				return nil, fmt.Errorf("error validating soundcloud track IDs in getTracks")
			}

			requested[id] = true
			urns = append(urns, fmt.Sprintf("soundcloud:tracks:%d", id))
		}

		params := url.Values{}
		params.Set("urns", strings.Join(urns, ","))
		params.Set("access", "playable,preview")
		params.Set("limit", strconv.Itoa(len(batch)))
		request := client.HTTPRequestData{
			Method:  http.MethodGet,
			URL:     fmt.Sprintf("%s/tracks", c.Endpoint),
			Payload: &params,
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("OAuth %s", c.accessToken),
				"Accept":        "application/json; charset=utf-8",
			},
		}

		body, err := c.HTTPClient.RequestBytes(ctx, request)
		if err != nil {
			var statusError internalerror.HTTPStatusCodeError
			if errors.As(err, &statusError) {
				if statusError.StatusCode == http.StatusNotFound {
					return nil, internalerror.ErrProviderItemNotFound{
						Err: fmt.Errorf("error fetching soundcloud collection in getTracks: %w", err),
					}
				}
				if statusError.StatusCode == http.StatusTooManyRequests {
					return nil, internalerror.ErrProviderQuotaExceeded{
						Err:      fmt.Errorf("error fetching soundcloud collection in getTracks: %w", err),
						Provider: vibe.SourceTypeSoundCloud,
					}
				}
			}

			return nil, fmt.Errorf("error requesting soundcloud collection in getTracks: %w", err)
		}

		var results []trackResponse
		err = json.Unmarshal(body, &results)
		if err != nil {
			return nil, fmt.Errorf("error decoding soundcloud collection in getTracks: %w", err)
		}

		byID := make(map[int]trackResponse, len(results))
		for _, result := range results {
			_, duplicate := byID[result.ID]
			if !requested[result.ID] || duplicate {
				return nil, fmt.Errorf("error validating soundcloud collection identity in getTracks")
			}
			if result.Title == "" || result.PermalinkURL == "" || result.Duration <= 0 {
				return nil, fmt.Errorf("error validating soundcloud collection metadata in getTracks")
			}

			byID[result.ID] = result
		}

		for _, id := range batch {
			track, exists := byID[id]
			if !exists {
				continue
			}

			tracks = append(tracks, track)
		}
	}

	return tracks, nil
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
		var statusCodeError internalerror.HTTPStatusCodeError
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

	track, err := c.GetProviderItem(ctx, strconv.Itoa(res.ID))
	if err != nil {
		return nil, fmt.Errorf("error converting soundcloud track in ResolveProviderItem: %w", err)
	}

	return track, nil
}

const soundCloudTrackBatchSize = 50
