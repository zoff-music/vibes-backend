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
		v2Payload, err := json.Marshal(vibe.SongIDUpdate{ID: refresh.SongID})
		if err != nil {
			return fmt.Errorf(
				"error marshaling compact unavailable %s song event in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}
		v2Event := &vibe.RoomEventV2Payload{Type: vibe.SongRemoved, Payload: v2Payload}

		err = h.Events.NotifyRoomUpdate(ctx, refresh.RoomID, vibe.RoomEvent{
			Type:    vibe.QueueReordered,
			Payload: payload,
			V2:      v2Event,
		})
		if err != nil {
			log.Printf(
				"failed to notify room after removing unavailable %s song: %v",
				h.ProviderName,
				err,
			)
		}

		playback, err := h.DB.GetPlaybackState(ctx, refresh.RoomID)
		if err != nil {
			return fmt.Errorf("error fetching playback after metadata removal in MetaRefresh.Handle: %w", err)
		}

		playbackPayload, err := json.Marshal(playback)
		if err != nil {
			return fmt.Errorf("error marshaling playback after metadata removal in MetaRefresh.Handle: %w", err)
		}

		err = h.Events.NotifyRoomUpdate(ctx, refresh.RoomID, vibe.RoomEvent{
			Type:    vibe.PlaybackUpdate,
			Payload: playbackPayload,
		})
		if err != nil {
			return fmt.Errorf("error notifying playback after metadata removal in MetaRefresh.Handle: %w", err)
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

	songs, err := h.DB.GetSongs(ctx, refresh.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching refreshed queue in MetaRefresh.Handle: %w", err)
	}

	for position, song := range songs {
		if song.ID != refresh.SongID {
			continue
		}

		payload, err := json.Marshal(songs)
		if err != nil {
			return fmt.Errorf("error marshaling refreshed queue in MetaRefresh.Handle: %w", err)
		}

		v2Payload, err := json.Marshal(vibe.SongPositionUpdate{Song: song, Position: position})
		if err != nil {
			return fmt.Errorf("error marshaling refreshed song in MetaRefresh.Handle: %w", err)
		}

		err = h.Events.NotifyRoomUpdate(ctx, refresh.RoomID, vibe.RoomEvent{
			Type:    vibe.QueueReordered,
			Payload: payload,
			V2:      &vibe.RoomEventV2Payload{Type: vibe.SongUpdated, Payload: v2Payload},
		})
		if err != nil {
			return fmt.Errorf("error notifying refreshed song in MetaRefresh.Handle: %w", err)
		}

		break
	}

	playback, err := h.DB.GetPlaybackState(ctx, refresh.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching refreshed playback in MetaRefresh.Handle: %w", err)
	}

	if playback.CurrentSong != nil && playback.CurrentSong.ID == refresh.SongID {
		payload, err := json.Marshal(playback)
		if err != nil {
			return fmt.Errorf("error marshaling refreshed playback in MetaRefresh.Handle: %w", err)
		}

		err = h.Events.NotifyRoomUpdate(ctx, refresh.RoomID, vibe.RoomEvent{
			Type:    vibe.PlaybackUpdate,
			Payload: payload,
		})
		if err != nil {
			return fmt.Errorf("error notifying refreshed playback in MetaRefresh.Handle: %w", err)
		}
	}

	return nil
}

type ExpireSongMetadata struct {
	DB     vibe.SongMetadataExpiryFetcher
	Events vibe.RoomBatchEventNotifier
}

func (h *ExpireSongMetadata) Handle(ctx context.Context, _ []byte) error {
	expiry, err := h.DB.ExpireSongMetadata(ctx)
	if err != nil {
		return fmt.Errorf("error expiring metadata in ExpireSongMetadata.Handle: %w", err)
	}

	songs, err := h.DB.GetSongs(ctx, expiry.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching queue in ExpireSongMetadata.Handle: %w", err)
	}

	payload, err := json.Marshal(songs)
	if err != nil {
		return fmt.Errorf("error marshaling queue in ExpireSongMetadata.Handle: %w", err)
	}

	playback, err := h.DB.GetPlaybackState(ctx, expiry.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching playback in ExpireSongMetadata.Handle: %w", err)
	}

	playbackPayload, err := json.Marshal(playback)
	if err != nil {
		return fmt.Errorf("error marshaling playback in ExpireSongMetadata.Handle: %w", err)
	}

	err = h.Events.NotifyRoomUpdates(ctx, expiry.RoomID, []vibe.RoomEvent{
		{
			Type:    vibe.QueueReordered,
			Payload: payload,
			V2:      &vibe.RoomEventV2Payload{Type: vibe.QueueSnapshot, Payload: payload},
		},
		{
			Type:    vibe.PlaybackUpdate,
			Payload: playbackPayload,
		},
	})
	if err != nil {
		return fmt.Errorf("error notifying metadata expiry in ExpireSongMetadata.Handle: %w", err)
	}

	return nil
}

const metadataRefreshInterval = 21 * 24 * time.Hour
const metadataRefreshRetry = 5 * time.Minute
const metadataQuotaRetry = 24 * time.Hour
