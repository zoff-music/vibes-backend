package vibe

import (
	"context"
	"encoding/json/v2"
	"fmt"
)

type AdminEventV2 struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

type AdminEventRecord struct {
	Version int          `json:"version"`
	Event   AdminEventV2 `json:"event"`
}

func (e AdminEvent) ToAdminEventV2() (*AdminEventV2, error) {
	event := AdminEventV2{Type: e.Type, Payload: e.Payload}
	if e.Type != AdminRoomsUpdate {
		return &event, nil
	}

	var legacy []AdminRoomSummary
	err := json.Unmarshal(e.Payload, &legacy)
	if err != nil {
		return nil, fmt.Errorf("error decoding legacy admin room update: %w", err)
	}

	var rooms []AdminRoomSummaryV2
	if legacy != nil {
		rooms = make([]AdminRoomSummaryV2, len(legacy))
		for index, room := range legacy {
			rooms[index] = *room.ToAdminRoomSummaryV2()
		}
	}

	event.Payload, err = json.Marshal(rooms)
	if err != nil {
		return nil, fmt.Errorf("error encoding admin room update: %w", err)
	}

	return &event, nil
}

func (e AdminEventV2) ToAdminEvent() (*AdminEvent, error) {
	event := AdminEvent{Type: e.Type, Payload: e.Payload}
	if e.Type != AdminRoomsUpdate {
		return &event, nil
	}

	var rooms []AdminRoomSummaryV2
	err := json.Unmarshal(e.Payload, &rooms)
	if err != nil {
		return nil, fmt.Errorf("error decoding admin room update: %w", err)
	}

	var legacy []AdminRoomSummary
	if rooms != nil {
		legacy = make([]AdminRoomSummary, len(rooms))
		for index, room := range rooms {
			legacy[index] = *room.ToAdminRoomSummary()
		}
	}

	event.Payload, err = json.Marshal(legacy)
	if err != nil {
		return nil, fmt.Errorf("error encoding legacy admin room update: %w", err)
	}

	return &event, nil
}

func ParseAdminEvent(data []byte) (*AdminEventV2, error) {
	var record AdminEventRecord
	err := json.Unmarshal(data, &record)
	if err != nil {
		return nil, fmt.Errorf("error decoding admin event record: %w", err)
	}
	if record.Version == 2 {
		return &record.Event, nil
	}
	if record.Version != 0 {
		return nil, fmt.Errorf("error unsupported admin event version: %d", record.Version)
	}

	var legacy AdminEvent
	err = json.Unmarshal(data, &legacy)
	if err != nil {
		return nil, fmt.Errorf("error decoding retained legacy admin event: %w", err)
	}

	event, err := legacy.ToAdminEventV2()
	if err != nil {
		return nil, fmt.Errorf("error converting retained legacy admin event: %w", err)
	}

	return event, nil
}

type AdminEventV2Notifier interface {
	NotifyAdminUpdateV2(ctx context.Context, event AdminEventV2) error
}
