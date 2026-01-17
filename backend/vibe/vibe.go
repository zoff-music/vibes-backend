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

// Subscriber subscribes to listen for InsiderThreatDetectionOrgStats messages
type Subscriber interface {
	Subscribe(topic string) (*SubscriptionContainer, error)
}
