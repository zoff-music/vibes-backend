package handler

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/vibe"
)

type GenerateRoomPlaylist struct {
	AI       vibe.PlaylistGenerator
	Cache    vibe.GeneratedPlaylistCacheNotifier
	DB       vibe.RoomGenerationWorker
	Searcher vibe.GeneratedPlaylistSearcher
}

func (h *GenerateRoomPlaylist) Handle(ctx context.Context, _ []byte) error {
	generation, err := h.DB.ProcessNextRoomGeneration(ctx)
	if err != nil {
		return fmt.Errorf("error processing next room generation in Handle: %w", err)
	}

	if generation.Exhausted {
		update := vibe.RoomGenerationUpdate{
			Status: vibe.RoomGenerationFailed,
			Error:  vibe.RoomGenerationFailure,
		}

		payload, err := json.Marshal(update)
		if err != nil {
			return fmt.Errorf("error marshaling failed room generation update in Handle: %w", err)
		}

		err = h.Cache.NotifyRoomUpdateV3(ctx, generation.RoomID, vibe.RoomEventV3{
			Type:    vibe.GenerationUpdate,
			Payload: payload,
		})
		if err != nil {
			return fmt.Errorf("error notifying exhausted room generation in Handle: %w", err)
		}

		return nil
	}

	room := generation.Room
	playbackState := generation.PlaybackState
	currentPlaylistItem := playbackState.CurrentPlaylistItem

	prompt, err := vibe.GeneratePlaylistPrompt(
		generation.Prompt,
		currentPlaylistItem,
		generation.PlaylistItems,
	)
	if err != nil {
		return fmt.Errorf("error generating playlist prompt in Handle: %w", err)
	}

	playlist, err := h.AI.GeneratePlaylist(ctx, prompt, room.RoomType)
	if err != nil {
		return fmt.Errorf("error generating playlist in Handle: %w", err)
	}

	queries := make([]string, 0, len(*playlist))
	for _, playlistItem := range *playlist {
		if playlistItem.Publisher == "" || playlistItem.Title == "" {
			continue
		}

		queries = append(queries, playlistItem.Publisher+" "+playlistItem.Title)
	}

	cachedSearches, err := h.Cache.GetCachedProviderSearches(ctx, vibe.SourceTypeYouTube, queries, room.RoomType)
	if err != nil {
		log.Printf("error getting cached youtube searches for room generation: %v", err)
		cachedSearches = []vibe.CachedProviderSearch{}
	}

	searchQuotaReset, err := h.Cache.GetProviderQuotaReset(
		ctx,
		vibe.SourceTypeYouTube,
		vibe.ProviderQuotaOperationSearch,
	)
	if err != nil {
		log.Printf("error getting cached youtube quota reset for room generation: %v", err)
		searchQuotaReset = time.Time{}
	}

	searchResult, err := h.Searcher.SearchGeneratedPlaylist(
		ctx,
		*playlist,
		cachedSearches,
		searchQuotaReset,
		room.RoomType,
	)
	if err != nil {
		var quotaError internalerror.ErrProviderQuotaExceeded
		if errors.As(err, &quotaError) {
			err = h.Cache.CacheProviderQuotaReset(
				ctx,
				vibe.SourceTypeYouTube,
				vibe.ProviderQuotaOperationSearch,
				quotaError.ResetAt,
			)
			if err != nil {
				log.Printf(
					"error caching youtube quota reset for room generation: %v",
					err,
				)
			}
			err = h.DB.FailRoomGeneration(
				ctx,
				generation.RoomID,
				vibe.RoomGenerationYouTubeQuotaFailure,
			)
			if err != nil {
				return fmt.Errorf(
					"error failing room generation after youtube quota exhaustion in Handle: %w",
					err,
				)
			}

			update := vibe.RoomGenerationUpdate{
				Status: vibe.RoomGenerationFailed,
				Error:  vibe.RoomGenerationYouTubeQuotaFailure,
			}
			payload, err := json.Marshal(update)
			if err != nil {
				return fmt.Errorf(
					"error marshaling youtube quota room generation update in Handle: %w",
					err,
				)
			}
			err = h.Cache.NotifyRoomUpdateV3(ctx, generation.RoomID, vibe.RoomEventV3{
				Type:    vibe.GenerationUpdate,
				Payload: payload,
			})
			if err != nil {
				return fmt.Errorf(
					"error notifying youtube quota room generation failure in Handle: %w",
					err,
				)
			}

			return internalerror.ErrExpected{
				Err: internalerror.ErrNonRecoverable{
					Err: fmt.Errorf("error checking youtube search quota in Handle: %w", quotaError),
				},
			}
		}

		return fmt.Errorf("error searching generated playlist in Handle: %w", err)
	}
	searchUsages := make([]vibe.SearchUsage, 0, len(searchResult.SearchUsages))
	for _, usage := range searchResult.SearchUsages {
		usage.RoomID = generation.RoomID
		searchUsages = append(searchUsages, usage)
	}

	err = h.DB.CreateSearchUsages(ctx, searchUsages)
	if err != nil {
		log.Printf("error creating generated playlist search usage: %v", err)
	}
	err = h.Cache.CacheProviderSearches(ctx, vibe.SourceTypeYouTube, searchResult.CachedSearches, room.RoomType)
	if err != nil {
		log.Printf("error caching youtube searches for room generation: %v", err)
	}
	playlist = &searchResult.Playlist

	shouldStartPlayback := playbackState.CurrentPlaylistItem == nil
	for _, generatedItem := range *playlist {
		playlistItem := &vibe.PlaylistItem{
			ID:                  uuid.NewString(),
			RoomID:              room.ID,
			SourceType:          vibe.SourceTypeYouTube,
			SourceID:            generatedItem.YouTubeID,
			ProviderURL:         fmt.Sprintf("https://www.youtube.com/watch?v=%s", generatedItem.YouTubeID),
			PlaybackRestriction: generatedItem.PlaybackRestriction,
			Title:               generatedItem.Title,
			Publisher:           generatedItem.Publisher,
			ThumbnailURL:        generatedItem.ThumbnailURL,
			Duration:            generatedItem.Duration,
			AddedBySessionID:    room.HostID,
			AddedAt:             time.Now(),
		}

		addedPlaylistItem, err := h.DB.AddGeneratedPlaylistItem(ctx, playlistItem)
		if err != nil {
			return fmt.Errorf("error adding generated playlist item in Handle: %w", err)
		}
		if addedPlaylistItem.IsEmpty() {
			continue
		}

		playlistItemPayload, err := json.Marshal(addedPlaylistItem)
		if err != nil {
			return fmt.Errorf("error marshaling generated playlist item in Handle: %w", err)
		}

		err = h.Cache.NotifyRoomUpdateV3(ctx, room.ID, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemAdded,
			Payload: playlistItemPayload,
		})
		if err != nil {
			return fmt.Errorf("error notifying generated playlist item in Handle: %w", err)
		}

		if shouldStartPlayback {
			playbackState := &vibe.PlaybackStateV2{
				RoomID:              room.ID,
				CurrentPlaylistItem: addedPlaylistItem,
				IsPlaying:           true,
				PositionMs:          0,
				UpdatedAt:           time.Now(),
				ServerTimeMs:        int(time.Now().UnixMilli()),
			}
			err = h.DB.UpsertPlaybackStateV2(ctx, playbackState)
			if err != nil {
				return fmt.Errorf("error starting generated room playback in Handle: %w", err)
			}

			playbackPayload, err := json.Marshal(playbackState)
			if err != nil {
				return fmt.Errorf("error marshaling generated room playback in Handle: %w", err)
			}
			err = h.Cache.NotifyRoomUpdateV3(ctx, room.ID, vibe.RoomEventV3{
				Type:    vibe.PlaybackUpdate,
				Payload: playbackPayload,
			})
			if err != nil {
				return fmt.Errorf("error notifying generated room playback in Handle: %w", err)
			}
			shouldStartPlayback = false
		}
	}

	err = h.DB.CompleteRoomGeneration(ctx, generation.RoomID)
	if err != nil {
		return fmt.Errorf("error completing room generation in Handle: %w", err)
	}

	update := vibe.RoomGenerationUpdate{Status: vibe.RoomGenerationCompleted}
	payload, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("error marshaling completed room generation update in Handle: %w", err)
	}
	err = h.Cache.NotifyRoomUpdateV3(ctx, generation.RoomID, vibe.RoomEventV3{
		Type:    vibe.GenerationUpdate,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("error notifying completed room generation in Handle: %w", err)
	}

	return nil
}
