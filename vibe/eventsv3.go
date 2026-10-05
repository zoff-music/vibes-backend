package vibe

import (
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
