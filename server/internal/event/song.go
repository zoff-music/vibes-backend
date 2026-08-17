package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

type MetaRefresh struct {
	DB           vibe.SongMetadataRefreshStorage
	Events       vibe.RoomEventNotifier
	Provider     vibe.MusicTrackFetcher
	ProviderName string
}

func (h *MetaRefresh) Handle(ctx context.Context, _ []byte) error {
	refresh, err := h.DB.ClaimSongMetadataRefresh(ctx, h.ProviderName, metadataRefreshRetry)
	if err != nil {
		return fmt.Errorf(
			"error claiming %s metadata in MetaRefresh.Handle: %w",
			h.ProviderName,
			err,
		)
	}

	track, err := h.Provider.GetTrack(ctx, refresh.SourceID)
	if err != nil {
		var quotaError internalerror.ErrProviderQuotaExceeded
		if errors.As(err, &quotaError) {
			err = h.DB.DeferSongMetadataRefresh(ctx, refresh.SongID, metadataQuotaRetry)
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

		var notFoundError internalerror.ErrMusicTrackNotFound
		if !errors.As(err, &notFoundError) {
			return fmt.Errorf(
				"error fetching %s metadata in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}

		err = h.DB.RemoveSong(ctx, refresh.RoomID, refresh.SongID)
		if err != nil {
			return fmt.Errorf(
				"error removing unavailable %s song in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}

		songs, err := h.DB.GetSongs(ctx, refresh.RoomID)
		if err != nil {
			return fmt.Errorf(
				"error fetching songs after removing unavailable %s song in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}

		payload, err := json.Marshal(songs)
		if err != nil {
			return fmt.Errorf(
				"error marshaling songs after removing unavailable %s song in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}

		err = h.Events.NotifyRoomUpdate(ctx, refresh.RoomID, vibe.RoomEvent{
			Type:    vibe.QueueReordered,
			Payload: payload,
		})
		if err != nil {
			log.Printf(
				"failed to notify room after removing unavailable %s song: %v",
				h.ProviderName,
				err,
			)
		}

		return nil
	}

	err = h.DB.RefreshSongMetadata(ctx, *refresh, *track, metadataRefreshInterval)
	if err != nil {
		return fmt.Errorf(
			"error refreshing %s metadata in MetaRefresh.Handle: %w",
			h.ProviderName,
			err,
		)
	}

	return nil
}

const metadataRefreshInterval = 21 * 24 * time.Hour
const metadataRefreshRetry = 5 * time.Minute
const metadataQuotaRetry = 24 * time.Hour
