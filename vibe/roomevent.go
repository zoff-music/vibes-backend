package vibe

import (
	"context"
	"encoding/json/v2"
	"fmt"
)

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
