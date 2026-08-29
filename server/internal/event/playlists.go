package event

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/vibe"
)

type ImportPlaylistSong struct {
	Events  vibe.RoomBatchEventNotifier
	Imports vibe.PlaylistImportProcessor
	Queue   vibe.PlaylistImportSongQueue
}

func (h *ImportPlaylistSong) Handle(ctx context.Context, _ []byte) error {
	playlistImport, err := h.Imports.ProcessNextPlaylistImport(
		ctx,
		playlistImportRetryInterval,
	)
	if err != nil {
		return fmt.Errorf("error processing next playlist import in Handle: %w", err)
	}
	if playlistImport.Exhausted {
		err = h.Imports.DeletePlaylistImport(ctx, playlistImport.ID)
		if err != nil {
			return fmt.Errorf("error deleting exhausted playlist import in Handle: %w", err)
		}

		return nil
	}

	queuedSongs, err := h.Queue.GetSongs(ctx, playlistImport.RoomID)
	if err != nil {
		return fmt.Errorf("error getting songs before playlist import item in Handle: %w", err)
	}

	result, err := h.Queue.AddSong(ctx, &playlistImport.Song)
	if err != nil {
		addSongErr := err
		existingSong, err := h.Queue.GetSong(
			ctx,
			playlistImport.RoomID,
			playlistImport.Song.ID,
		)
		if err != nil {
			return fmt.Errorf("error checking playlist import item after add failure in Handle: %w", err)
		}
		if existingSong.IsEmpty() {
			return fmt.Errorf("error adding playlist import item in Handle: %w", addSongErr)
		}

		result = &vibe.AddSongResult{
			Song:    *existingSong,
			Outcome: vibe.AddSongOutcomeAdded,
		}
	}

	var events []vibe.RoomEvent
	if result.Outcome == vibe.AddSongOutcomeAdded {
		songPayload, err := json.Marshal(result.Song)
		if err != nil {
			return fmt.Errorf("error marshaling playlist import song in Handle: %w", err)
		}
		events = append(events, vibe.RoomEvent{
			Type:    vibe.SongAdded,
			Payload: songPayload,
		})
	} else {
		updatedSongs, err := h.Queue.GetSongs(ctx, playlistImport.RoomID)
		if err != nil {
			return fmt.Errorf("error getting songs after duplicate playlist import item in Handle: %w", err)
		}
		queuePayload, err := json.Marshal(updatedSongs)
		if err != nil {
			return fmt.Errorf("error marshaling playlist import queue in Handle: %w", err)
		}
		events = append(events, vibe.RoomEvent{
			Type:    vibe.QueueReordered,
			Payload: queuePayload,
		})
	}
	if len(queuedSongs) == 0 && result.Outcome == vibe.AddSongOutcomeAdded {
		playbackState, err := h.Queue.StartPlaybackIfIdle(ctx, playlistImport.RoomID)
		if err != nil {
			return fmt.Errorf("error starting playlist import playback in Handle: %w", err)
		}
		playbackPayload, err := json.Marshal(playbackState)
		if err != nil {
			return fmt.Errorf("error marshaling playlist import playback in Handle: %w", err)
		}
		events = append(events, vibe.RoomEvent{
			Type:    vibe.PlaybackUpdate,
			Payload: playbackPayload,
		})
	}

	err = h.Imports.CompletePlaylistImportItem(
		ctx,
		playlistImport.ID,
		playlistImport.NextPosition,
	)
	if err != nil {
		return fmt.Errorf("error completing playlist import item in Handle: %w", err)
	}

	err = h.Events.NotifyRoomUpdates(ctx, playlistImport.RoomID, events)
	if err != nil {
		return fmt.Errorf("error notifying playlist import item in Handle: %w", err)
	}

	return nil
}

const playlistImportRetryInterval = 5 * time.Minute
