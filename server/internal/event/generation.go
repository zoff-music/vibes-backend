package event

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
	Cache    vibe.GeneratedPlaylistCache
	DB       vibe.RoomGenerationWorker
	Events   vibe.RoomEventNotifier
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
		err = h.Events.NotifyRoomUpdate(ctx, generation.RoomID, vibe.RoomEvent{
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
	prompt, err := vibe.GeneratePlaylistPrompt(
		generation.Prompt,
		playbackState.CurrentSong,
		generation.Songs,
	)
	if err != nil {
		return fmt.Errorf("error generating playlist prompt in Handle: %w", err)
	}

	playlist, err := h.AI.GeneratePlaylist(ctx, prompt)
	if err != nil {
		return fmt.Errorf("error generating playlist in Handle: %w", err)
	}

	queries := make([]string, 0, len(*playlist))
	for _, track := range *playlist {
		if track.Artist == "" || track.Title == "" {
			continue
		}
		queries = append(queries, track.Artist+" "+track.Title)
	}
	cachedSearches, err := h.Cache.GetCachedSearches(ctx, vibe.SourceTypeYouTube, queries)
	if err != nil {
		log.Printf("error getting cached youtube searches for room generation: %v", err)
		cachedSearches = []vibe.CachedSearch{}
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
			err = h.Events.NotifyRoomUpdate(ctx, generation.RoomID, vibe.RoomEvent{
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
	err = h.DB.CreateSearchUsages(ctx, searchResult.SearchUsages)
	if err != nil {
		log.Printf("error creating generated playlist search usage: %v", err)
	}
	err = h.Cache.CacheSearches(ctx, vibe.SourceTypeYouTube, searchResult.CachedSearches)
	if err != nil {
		log.Printf("error caching youtube searches for room generation: %v", err)
	}
	playlist = &searchResult.Playlist

	shouldStartPlayback := playbackState.CurrentSong == nil
	for _, track := range *playlist {
		song := &vibe.Song{
			ID:                  uuid.NewString(),
			RoomID:              room.ID,
			SourceType:          vibe.SourceTypeYouTube,
			SourceID:            track.YouTubeID,
			ProviderURL:         fmt.Sprintf("https://www.youtube.com/watch?v=%s", track.YouTubeID),
			PlaybackRestriction: track.PlaybackRestriction,
			Title:               track.Title,
			Artist:              track.Artist,
			ThumbnailURL:        track.ThumbnailURL,
			Duration:            track.Duration,
			AddedBy:             room.HostID,
			AddedAt:             time.Now(),
		}

		addedSong, err := h.DB.AddGeneratedSong(ctx, song)
		if err != nil {
			return fmt.Errorf("error adding generated song in Handle: %w", err)
		}
		if addedSong.IsEmpty() {
			continue
		}

		songPayload, err := json.Marshal(addedSong)
		if err != nil {
			return fmt.Errorf("error marshaling generated song in Handle: %w", err)
		}

		err = h.Events.NotifyRoomUpdate(ctx, room.ID, vibe.RoomEvent{
			Type:    vibe.SongAdded,
			Payload: songPayload,
		})
		if err != nil {
			return fmt.Errorf("error notifying generated song in Handle: %w", err)
		}

		if shouldStartPlayback {
			playbackState := &vibe.PlaybackState{
				RoomID:       room.ID,
				CurrentSong:  addedSong,
				IsPlaying:    true,
				PositionMs:   0,
				UpdatedAt:    time.Now(),
				ServerTimeMs: int(time.Now().UnixMilli()),
			}
			err = h.DB.UpsertPlaybackState(ctx, playbackState)
			if err != nil {
				return fmt.Errorf("error starting generated room playback in Handle: %w", err)
			}

			playbackPayload, err := json.Marshal(playbackState)
			if err != nil {
				return fmt.Errorf("error marshaling generated room playback in Handle: %w", err)
			}
			err = h.Events.NotifyRoomUpdate(ctx, room.ID, vibe.RoomEvent{
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
	err = h.Events.NotifyRoomUpdate(ctx, generation.RoomID, vibe.RoomEvent{
		Type:    vibe.GenerationUpdate,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("error notifying completed room generation in Handle: %w", err)
	}

	return nil
}
