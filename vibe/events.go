package vibe

import (
	"context"
)

// Subscription exposes a stream of application events.
type Subscription interface {
	Listen() chan []byte
	Destroy()
}

type SubscriptionContainer struct {
	Subscription Subscription
}

// Subscriber subscribes to listen for room events
type Subscriber interface {
	Subscribe(ctx context.Context, topic string) (*SubscriptionContainer, error)
}

type ReplaySubscription struct {
	AfterID          string
	RequiresSnapshot bool
}

type ReplaySubscriber interface {
	PrepareReplay(ctx context.Context, topic string, lastEventID string) (*ReplaySubscription, error)
	SubscribeFrom(ctx context.Context, topic string, afterID string) (*SubscriptionContainer, error)
}

type RoomEventReplayNotifier interface {
	ReplaySubscriber
	RoomEventNotifier
}

// RoomEventSnapshotFetcher fetches the authoritative state sent on connection.
type RoomEventSnapshotFetcher interface {
	RoomFetcher
	SongsFetcher
	PlaybackFetcher
}

// RoomEventConnection describes when an SSE connection was established.
type RoomEventConnection struct {
	Time int64 `json:"time"`
}

// RoomEvent represents an SSE event for a room
type RoomEvent struct {
	ID      string              `json:"id,omitempty"`
	Type    string              `json:"type"`
	Payload []byte              `json:"payload"`
	UserID  string              `json:"userId,omitempty"` // ID of user who triggered this event
	Origin  string              `json:"origin,omitempty"`
	V2      *RoomEventV2Payload `json:"v2,omitempty"`
}

// RoomEventV2Payload is the compact event representation exposed by the v2 SSE endpoint.
type RoomEventV2Payload struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

// SongIDUpdate identifies one song removed from a queue.
type SongIDUpdate struct {
	ID string `json:"id"`
}

// SongPositionUpdate replaces and repositions one song in a queue.
type SongPositionUpdate struct {
	Song     Song `json:"song"`
	Position int  `json:"position"`
}

type RoomEventCursor struct {
	ID string `json:"id"`
}

// RoomEventNotifier broadcasts events to room subscribers
type RoomEventNotifier interface {
	NotifyRoomUpdate(ctx context.Context, roomID string, event RoomEvent) error
}

// RoomBatchEventNotifier broadcasts events to room subscribers in batches
type RoomBatchEventNotifier interface {
	NotifyRoomUpdates(ctx context.Context, roomID string, events []RoomEvent) error
}

const PlaybackUpdate = "playback_update"
const SongAdded = "song_added"
const SongRemoved = "song_removed"
const SongUpdated = "song_updated"
const QueueReordered = "songs_update"
const QueueSnapshot = "songs_snapshot"
const SkipVoteEvent = "skip_vote"
const NewHost = "new_host"
const UserJoined = "user_joined"
const UserLeft = "user_left"
const UsersUpdate = "users_update"
const SettingsUpdate = "settings_update"
const GenerationUpdate = "generation_update"
const Connected = "connected"
const EventCursor = "event_cursor"

const RoomEventOriginRemote = "remote"
