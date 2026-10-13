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

// RoomEventConnection describes when an SSE connection was established.
type RoomEventConnection struct {
	Time int64 `json:"time"`
}

type RoomEventCursor struct {
	ID string `json:"id"`
}

const PlaybackUpdate = "playback_update"
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
