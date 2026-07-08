package vibe

import (
	"context"
	"time"
)

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

type SubscriberPublisher interface {
	Subscriber
	RoomEventNotifier
}

// RoomEvent represents an SSE event for a room
type RoomEvent struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
	UserID  string `json:"userId,omitempty"` // ID of user who triggered this event
}

// RoomEventNotifier broadcasts events to room subscribers
type RoomEventNotifier interface {
	NotifyRoomUpdate(ctx context.Context, roomID string, event RoomEvent) error
}

type RoomEventAdminNotifier interface {
	RoomEventNotifier
	AdminEventNotifier
}

// RoomBatchEventNotifier broadcasts events to room subscribers in batches
type RoomBatchEventNotifier interface {
	NotifyRoomUpdates(ctx context.Context, roomID string, events []RoomEvent) error
}

type RoomBatchEventAdminNotifier interface {
	RoomBatchEventNotifier
	AdminEventNotifier
}

// ParticipantGetterUpdaterPlaybackGetter defines methods for managing room participants
type ParticipantGetterUpdaterPlaybackGetter interface {
	UpdateParticipant(ctx context.Context, roomID, userID string, isActiveListener bool, isCastReceiver bool, castOwnerID string) error
	GetActiveParticipants(ctx context.Context, roomID string, activeWithin time.Duration) ([]Participant, error)
	GetActiveListenerCounts(ctx context.Context, roomID string, activeWithin time.Duration) (ListenerCounts, error)
	GetPlaybackState(ctx context.Context, roomID string) (*PlaybackState, error)
	RemoveParticipant(ctx context.Context, roomID, userID string) error
}

const PlaybackUpdate = "playback_update"
const SongAdded = "song_added"
const SongRemoved = "song_removed"
const QueueReordered = "songs_update"
const SkipVoteEvent = "skip_vote"
const NewHost = "new_host"
const UserJoined = "user_joined"
const UserLeft = "user_left"
const UsersUpdate = "users_update"
const SettingsUpdate = "settings_update"
