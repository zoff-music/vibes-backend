package vibe

import "context"

// Subscription defines interface to use the internalpubsub client
type Subscription interface {
	Listen() chan []byte
	Destroy()
}

type SubscriptionContainer struct {
	Subscription Subscription
}

// Publisher relays messages to subscribed clients
type Publisher interface {
	PublishToInternalSubscription(ctx context.Context, topic string, data []byte) error
}

// Subscriber subscribes to listen for room events
type Subscriber interface {
	Subscribe(topic string) (*SubscriptionContainer, error)
}

// EventType represents the type of SSE event
type EventType string

const (
	EventTypePlaybackUpdate EventType = "playback_update"
	EventTypeSongAdded      EventType = "song_added"
	EventTypeSongRemoved    EventType = "song_removed"
	EventTypeQueueReordered EventType = "queue_reordered"
	EventTypeUserJoined     EventType = "user_joined"
	EventTypeUserLeft       EventType = "user_left"
	EventTypeSettingsUpdate EventType = "settings_update"
)

// RoomEvent represents an SSE event for a room
type RoomEvent struct {
	Type    EventType   `json:"type"`
	Payload interface{} `json:"payload"`
}

// RoomEventNotifier broadcasts events to room subscribers
type RoomEventNotifier interface {
	NotifyRoom(ctx context.Context, roomID string, event *RoomEvent) error
}
