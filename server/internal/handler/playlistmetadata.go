package handler

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

type MetaRefresh struct {
	DB           vibe.PlaylistItemMetadataRefreshStorage
	Events       vibe.RoomEventV3Notifier
	Provider     vibe.ProviderItemFetcher
	ProviderName string
}

func (h *MetaRefresh) Handle(ctx context.Context, _ []byte) error {
	refresh, err := h.DB.ClaimPlaylistItemMetadataRefresh(ctx, h.ProviderName, metadataRefreshRetry)
	if err != nil {
		return fmt.Errorf(
			"error claiming %s metadata in MetaRefresh.Handle: %w",
			h.ProviderName,
			err,
		)
	}

	providerItem, err := h.Provider.GetProviderItem(ctx, refresh.SourceID)
	if err != nil {
		var quotaError internalerror.ErrProviderQuotaExceeded
		if errors.As(err, &quotaError) {
			err = h.DB.DeferPlaylistItemMetadataRefresh(ctx, refresh.PlaylistItemID, metadataQuotaRetry)
			if err != nil {
				return fmt.Errorf(
					"error deferring %s metadata refresh in MetaRefresh.Handle: %w",
					h.ProviderName,
					err,
				)
			}

			return fmt.Errorf(
				"error fetching rate-limited %s metadata in MetaRefresh.Handle: %w",
				h.ProviderName,
				quotaError,
			)
		}

		var notFoundError internalerror.ErrProviderItemNotFound
		var liveVideoError internalerror.ErrLiveVideo
		var madeForKidsError internalerror.ErrMadeForKids
		if !errors.As(err, &notFoundError) &&
			!errors.As(err, &liveVideoError) &&
			!errors.As(err, &madeForKidsError) {
			return fmt.Errorf(
				"error fetching %s metadata in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}

		err = h.DB.RemovePlaylistItem(ctx, refresh.RoomID, refresh.PlaylistItemID)
		if err != nil {
			return fmt.Errorf(
				"error removing unavailable %s playlist item in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}

		playlistItems, err := h.DB.GetPlaylistItems(ctx, refresh.RoomID)
		if err != nil {
			return fmt.Errorf(
				"error fetching playlist items after removing unavailable %s playlist item in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}

		payload, err := json.Marshal(playlistItems)
		if err != nil {
			return fmt.Errorf(
				"error marshaling playlist items after removing unavailable %s playlist item in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}
		compactPayload, err := json.Marshal(vibe.PlaylistItemIDUpdate{ID: refresh.PlaylistItemID})
		if err != nil {
			return fmt.Errorf(
				"error marshaling compact unavailable %s playlist item event in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}
		compactEvent := &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemRemoved, Payload: compactPayload}

		err = h.Events.NotifyRoomUpdateV3(ctx, refresh.RoomID, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: payload,
			Compact: compactEvent,
		})
		if err != nil {
			log.Printf(
				"failed to notify room after removing unavailable %s playlist item: %v",
				h.ProviderName,
				err,
			)
		}

		playback, err := h.DB.GetPlaybackStateV2(ctx, refresh.RoomID)
		if err != nil {
			return fmt.Errorf("error fetching playback after metadata removal in MetaRefresh.Handle: %w", err)
		}

		playbackPayload, err := json.Marshal(playback)
		if err != nil {
			return fmt.Errorf("error marshaling playback after metadata removal in MetaRefresh.Handle: %w", err)
		}

		err = h.Events.NotifyRoomUpdateV3(ctx, refresh.RoomID, vibe.RoomEventV3{
			Type:    vibe.PlaybackUpdate,
			Payload: playbackPayload,
		})
		if err != nil {
			return fmt.Errorf("error notifying playback after metadata removal in MetaRefresh.Handle: %w", err)
		}

		return nil
	}

	err = h.DB.RefreshPlaylistItemMetadata(ctx, *refresh, *providerItem, metadataRefreshInterval)
	if err != nil {
		return fmt.Errorf(
			"error refreshing %s metadata in MetaRefresh.Handle: %w",
			h.ProviderName,
			err,
		)
	}

	playlistItems, err := h.DB.GetPlaylistItems(ctx, refresh.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching refreshed queue in MetaRefresh.Handle: %w", err)
	}

	for position, playlistItem := range playlistItems {
		if playlistItem.ID != refresh.PlaylistItemID {
			continue
		}

		payload, err := json.Marshal(playlistItems)
		if err != nil {
			return fmt.Errorf("error marshaling refreshed queue in MetaRefresh.Handle: %w", err)
		}

		compactPayload, err := json.Marshal(vibe.PlaylistItemPositionUpdate{PlaylistItem: playlistItem, Position: position})
		if err != nil {
			return fmt.Errorf("error marshaling refreshed playlist item in MetaRefresh.Handle: %w", err)
		}

		err = h.Events.NotifyRoomUpdateV3(ctx, refresh.RoomID, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: payload,
			Compact: &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemUpdated, Payload: compactPayload},
		})
		if err != nil {
			return fmt.Errorf("error notifying refreshed playlist item in MetaRefresh.Handle: %w", err)
		}

		break
	}

	playback, err := h.DB.GetPlaybackStateV2(ctx, refresh.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching refreshed playback in MetaRefresh.Handle: %w", err)
	}

	if playback.CurrentPlaylistItem != nil && playback.CurrentPlaylistItem.ID == refresh.PlaylistItemID {
		payload, err := json.Marshal(playback)
		if err != nil {
			return fmt.Errorf("error marshaling refreshed playback in MetaRefresh.Handle: %w", err)
		}

		err = h.Events.NotifyRoomUpdateV3(ctx, refresh.RoomID, vibe.RoomEventV3{
			Type:    vibe.PlaybackUpdate,
			Payload: payload,
		})
		if err != nil {
			return fmt.Errorf("error notifying refreshed playback in MetaRefresh.Handle: %w", err)
		}
	}

	return nil
}

type ExpirePlaylistItemMetadata struct {
	DB     vibe.PlaylistItemMetadataExpiryFetcher
	Events vibe.RoomBatchEventV3Notifier
}

func (h *ExpirePlaylistItemMetadata) Handle(ctx context.Context, _ []byte) error {
	expiry, err := h.DB.ExpirePlaylistItemMetadata(ctx)
	if err != nil {
		return fmt.Errorf("error expiring metadata in ExpirePlaylistItemMetadata.Handle: %w", err)
	}

	playlistItems, err := h.DB.GetPlaylistItems(ctx, expiry.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching queue in ExpirePlaylistItemMetadata.Handle: %w", err)
	}

	payload, err := json.Marshal(playlistItems)
	if err != nil {
		return fmt.Errorf("error marshaling queue in ExpirePlaylistItemMetadata.Handle: %w", err)
	}

	playback, err := h.DB.GetPlaybackStateV2(ctx, expiry.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching playback in ExpirePlaylistItemMetadata.Handle: %w", err)
	}

	playbackPayload, err := json.Marshal(playback)
	if err != nil {
		return fmt.Errorf("error marshaling playback in ExpirePlaylistItemMetadata.Handle: %w", err)
	}

	err = h.Events.NotifyRoomUpdatesV3(ctx, expiry.RoomID, []vibe.RoomEventV3{
		{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: payload,
			Compact: &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemsSnapshot, Payload: payload},
		},
		{
			Type:    vibe.PlaybackUpdate,
			Payload: playbackPayload,
		},
	})
	if err != nil {
		return fmt.Errorf("error notifying metadata expiry in ExpirePlaylistItemMetadata.Handle: %w", err)
	}

	return nil
}

const metadataRefreshInterval = 21 * 24 * time.Hour
const metadataRefreshRetry = 5 * time.Minute
const metadataQuotaRetry = 24 * time.Hour
