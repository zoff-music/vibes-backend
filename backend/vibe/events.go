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

const (
	PlaybackUpdate = "playback_update"
	SongAdded      = "song_added"
	SongRemoved    = "song_removed"
	QueueReordered = "songs_update"
	NewHost        = "new_host"
	UserJoined     = "user_joined"
	UserLeft       = "user_left"
	SettingsUpdate = "settings_update"
)

// RoomEvent represents an SSE event for a room
type RoomEvent struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

// RoomEventNotifier broadcasts events to room subscribers
type RoomEventNotifier interface {
	NotifyRoomUpdate(ctx context.Context, roomID string, event RoomEvent) error
}

type RoomBatchEventNotifier interface {
	NotifyRoomUpdates(ctx context.Context, roomID string, events []RoomEvent) error
}
