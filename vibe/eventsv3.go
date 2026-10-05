package vibe

import (
	"context"
	"encoding/json/v2"
	"fmt"
)

type RoomEventV3StateFetcherUpdater interface {
	RoomEventParticipantFetcherUpdater
	RoomV2Fetcher
	PlaylistItemsFetcher
	PlaybackV2Fetcher
}

type RoomEventV3Payload struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

type PlaylistItemPositionUpdate struct {
	PlaylistItem PlaylistItem `json:"playlistItem"`
	Position     int          `json:"position"`
}

type PlaylistItemIDUpdate struct {
	ID string `json:"id"`
}

type SkipVoteUpdateV2 struct {
	UserID         string `json:"userId"`
	PlaylistItemID string `json:"playlistItemId"`
	CurrentVotes   int    `json:"currentVotes"`
	RequiredVotes  int    `json:"requiredVotes"`
}

const PlaylistItemAdded = "playlist_item_added"
const PlaylistItemRemoved = "playlist_item_removed"
const PlaylistItemUpdated = "playlist_item_updated"
const PlaylistItemsSnapshot = "playlist_items_snapshot"

func (e RoomEventV3Payload) ToRoomEventV2Payload() (*RoomEventV2Payload, error) {
	result := RoomEventV2Payload{Type: e.Type, Payload: e.Payload}
	switch e.Type {
	case PlaylistItemsUpdate, PlaylistItemsSnapshot:
		var items []PlaylistItem
		err := json.Unmarshal(e.Payload, &items)
		if err != nil {
			return nil, fmt.Errorf("error decoding playlist snapshot: %w", err)
		}

		var legacy []Song
		if items != nil {
			legacy = make([]Song, len(items))
			for index, item := range items {
				legacy[index] = *item.ToSong()
			}
		}
		result.Payload, err = json.Marshal(legacy)
		if err != nil {
			return nil, fmt.Errorf("error encoding legacy playlist snapshot: %w", err)
		}
		result.Type = QueueSnapshot
		if e.Type == PlaylistItemsUpdate {
			result.Type = QueueReordered
		}

	case PlaylistItemAdded:
		var item PlaylistItem
		err := json.Unmarshal(e.Payload, &item)
		if err != nil {
			return nil, fmt.Errorf("error decoding added playlist item: %w", err)
		}
		result.Payload, err = json.Marshal(item.ToSong())
		if err != nil {
			return nil, fmt.Errorf("error encoding legacy added item: %w", err)
		}
		result.Type = SongAdded

	case PlaylistItemUpdated:
		var item PlaylistItemPositionUpdate
		err := json.Unmarshal(e.Payload, &item)
		if err != nil {
			return nil, fmt.Errorf("error decoding positioned playlist item: %w", err)
		}
		result.Payload, err = json.Marshal(SongPositionUpdate{Song: *item.PlaylistItem.ToSong(), Position: item.Position})
		if err != nil {
			return nil, fmt.Errorf("error encoding legacy positioned item: %w", err)
		}
		result.Type = SongUpdated

	case PlaylistItemRemoved:
		result.Type = SongRemoved

	case PlaybackUpdate:
		var playback *PlaybackStateV2
		err := json.Unmarshal(e.Payload, &playback)
		if err != nil {
			return nil, fmt.Errorf("error decoding playback event: %w", err)
		}
		var legacy *PlaybackState
		if playback != nil {
			legacy = playback.ToPlaybackState()
		}
		result.Payload, err = json.Marshal(legacy)
		if err != nil {
			return nil, fmt.Errorf("error encoding legacy playback event: %w", err)
		}

	case SettingsUpdate:
		var room RoomV2
		err := json.Unmarshal(e.Payload, &room)
		if err != nil {
			return nil, fmt.Errorf("error decoding settings event: %w", err)
		}
		result.Payload, err = json.Marshal(room.ToRoom())
		if err != nil {
			return nil, fmt.Errorf("error encoding legacy settings event: %w", err)
		}

	case SkipVoteEvent:
		var vote SkipVoteUpdateV2
		err := json.Unmarshal(e.Payload, &vote)
		if err != nil {
			return nil, fmt.Errorf("error decoding skip vote event: %w", err)
		}
		result.Payload, err = json.Marshal(SkipVoteUpdate{UserID: vote.UserID, SongID: vote.PlaylistItemID, CurrentVotes: vote.CurrentVotes, RequiredVotes: vote.RequiredVotes})
		if err != nil {
			return nil, fmt.Errorf("error encoding legacy skip vote event: %w", err)
		}
	}

	return &result, nil
}

// ToRoomEventV3Payload translates retained legacy stream records without
// publishing a second event or changing their cursor, origin, or identity.
func (e RoomEvent) ToRoomEventV3Payload() (*RoomEventV3Payload, error) {
	result := RoomEventV3Payload{Type: e.Type, Payload: e.Payload}
	if e.V2 != nil {
		result.Type = e.V2.Type
		result.Payload = e.V2.Payload
	}

	switch result.Type {
	case QueueReordered, QueueSnapshot:
		var legacy []Song
		err := json.Unmarshal(result.Payload, &legacy)
		if err != nil {
			return nil, fmt.Errorf("error decoding legacy queue event: %w", err)
		}

		var items []PlaylistItem
		if legacy != nil {
			items = make([]PlaylistItem, len(legacy))
			for index, item := range legacy {
				items[index] = *item.ToPlaylistItem()
			}
		}

		result.Payload, err = json.Marshal(items)
		if err != nil {
			return nil, fmt.Errorf("error encoding playlist item snapshot: %w", err)
		}
		result.Type = PlaylistItemsSnapshot

	case SongAdded:
		var legacy Song
		err := json.Unmarshal(result.Payload, &legacy)
		if err != nil {
			return nil, fmt.Errorf("error decoding legacy added item: %w", err)
		}

		item := legacy.ToPlaylistItem()
		result.Payload, err = json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("error encoding added playlist item: %w", err)
		}
		result.Type = PlaylistItemAdded

	case SongUpdated:
		var legacy SongPositionUpdate
		err := json.Unmarshal(result.Payload, &legacy)
		if err != nil {
			return nil, fmt.Errorf("error decoding legacy positioned item: %w", err)
		}

		item := legacy.Song.ToPlaylistItem()
		positioned := PlaylistItemPositionUpdate{PlaylistItem: *item, Position: legacy.Position}
		result.Payload, err = json.Marshal(positioned)
		if err != nil {
			return nil, fmt.Errorf("error encoding positioned playlist item: %w", err)
		}
		result.Type = PlaylistItemUpdated

	case SongRemoved:
		result.Type = PlaylistItemRemoved

	case PlaybackUpdate:
		var legacy *PlaybackState
		err := json.Unmarshal(result.Payload, &legacy)
		if err != nil {
			return nil, fmt.Errorf("error decoding legacy playback event: %w", err)
		}

		var playback *PlaybackStateV2
		if legacy != nil {
			playback = legacy.ToPlaybackStateV2()
		}
		result.Payload, err = json.Marshal(playback)
		if err != nil {
			return nil, fmt.Errorf("error encoding playback event: %w", err)
		}

	case SettingsUpdate:
		var legacy Room
		err := json.Unmarshal(result.Payload, &legacy)
		if err != nil {
			return nil, fmt.Errorf("error decoding legacy settings event: %w", err)
		}

		room := legacy.ToRoomV2()
		result.Payload, err = json.Marshal(room)
		if err != nil {
			return nil, fmt.Errorf("error encoding settings event: %w", err)
		}

	case SkipVoteEvent:
		var legacy SkipVoteUpdate
		err := json.Unmarshal(result.Payload, &legacy)
		if err != nil {
			return nil, fmt.Errorf("error decoding legacy skip vote event: %w", err)
		}

		vote := SkipVoteUpdateV2{
			UserID:         legacy.UserID,
			PlaylistItemID: legacy.SongID,
			CurrentVotes:   legacy.CurrentVotes,
			RequiredVotes:  legacy.RequiredVotes,
		}
		result.Payload, err = json.Marshal(vote)
		if err != nil {
			return nil, fmt.Errorf("error encoding skip vote event: %w", err)
		}
	}

	return &result, nil
}

type RoomEventV3 struct {
	Version int                 `json:"version"`
	ID      string              `json:"id,omitempty"`
	Type    string              `json:"type"`
	Payload []byte              `json:"payload"`
	UserID  string              `json:"userId,omitempty"`
	Origin  string              `json:"origin,omitempty"`
	Compact *RoomEventV3Payload `json:"compact,omitempty"`
}

const PlaylistItemsUpdate = "playlist_items_update"

func (e RoomEvent) ToRoomEventV3() (*RoomEventV3, error) {
	full := e
	full.V2 = nil
	payload, err := full.ToRoomEventV3Payload()
	if err != nil {
		return nil, fmt.Errorf("error converting full room event: %w", err)
	}
	if e.Type == QueueReordered {
		payload.Type = PlaylistItemsUpdate
	}

	result := RoomEventV3{Version: 3, ID: e.ID, Type: payload.Type, Payload: payload.Payload, UserID: e.UserID, Origin: e.Origin}
	if e.V2 != nil {
		result.Compact, err = e.ToRoomEventV3Payload()
		if err != nil {
			return nil, fmt.Errorf("error converting compact room event: %w", err)
		}
	}

	return &result, nil
}

func (e RoomEventV3) ToRoomEventV3Payload() (*RoomEventV3Payload, error) {
	result := RoomEventV3Payload{Type: e.Type, Payload: e.Payload}
	if e.Compact != nil {
		result = *e.Compact
	}
	if result.Type == PlaylistItemsUpdate {
		result.Type = PlaylistItemsSnapshot
	}

	return &result, nil
}

func (e RoomEventV3) ToRoomEvent() (*RoomEvent, error) {
	full := RoomEventV3Payload{Type: e.Type, Payload: e.Payload}
	payload, err := full.ToRoomEventV2Payload()
	if err != nil {
		return nil, fmt.Errorf("error converting legacy full room event: %w", err)
	}

	result := RoomEvent{ID: e.ID, Type: payload.Type, Payload: payload.Payload, UserID: e.UserID, Origin: e.Origin}
	if e.Compact != nil {
		result.V2, err = e.Compact.ToRoomEventV2Payload()
		if err != nil {
			return nil, fmt.Errorf("error converting legacy compact room event: %w", err)
		}
	}

	return &result, nil
}

func ParseRoomEvent(data []byte) (*RoomEventV3, error) {
	var event RoomEventV3
	err := json.Unmarshal(data, &event)
	if err != nil {
		return nil, fmt.Errorf("error decoding room event: %w", err)
	}
	if event.Version == 3 {
		return &event, nil
	}
	if event.Version != 0 {
		return nil, fmt.Errorf("error unsupported room event version: %d", event.Version)
	}

	var legacy RoomEvent
	err = json.Unmarshal(data, &legacy)
	if err != nil {
		return nil, fmt.Errorf("error decoding retained legacy room event: %w", err)
	}

	canonical, err := legacy.ToRoomEventV3()
	if err != nil {
		return nil, fmt.Errorf("error converting retained legacy room event: %w", err)
	}

	return canonical, nil
}

func ParseLegacyRoomEvent(data []byte) (*RoomEvent, error) {
	event, err := ParseRoomEvent(data)
	if err != nil {
		return nil, fmt.Errorf("error decoding legacy room delivery: %w", err)
	}

	legacy, err := event.ToRoomEvent()
	if err != nil {
		return nil, fmt.Errorf("error converting legacy room delivery: %w", err)
	}

	return legacy, nil
}

type RoomEventV3Notifier interface {
	NotifyRoomUpdateV3(ctx context.Context, roomID string, event RoomEventV3) error
}

type RoomBatchEventV3Notifier interface {
	NotifyRoomUpdatesV3(ctx context.Context, roomID string, events []RoomEventV3) error
}

type RoomEventV3ReplayNotifier interface {
	ReplaySubscriber
	RoomEventV3Notifier
}

type RoomEventBatchV3Notifier interface {
	RoomEventV3Notifier
	RoomBatchEventV3Notifier
}

type RoomRemoteEventV3Notifier interface {
	RoomEventV3Notifier
	RemoteEventNotifierV2
}
