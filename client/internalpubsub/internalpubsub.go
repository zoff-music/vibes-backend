package internalpubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"

	"github.com/google/uuid"
)

type Client struct {
	mu     sync.Mutex
	topics map[string]*Topic
}

func (c *Client) Init() error {
	c.topics = make(map[string]*Topic)

	return nil
}

func (c *Client) NotifyTopic(ctx context.Context, topicName string, data []byte) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "NotifyTopic")
	defer span.Finish()

	topic := c.getTopic(topicName)

	topic.publish(data)
	return nil
}

func (c *Client) NotifyRoomUpdate(ctx context.Context, roomID string, event vibe.RoomEvent) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "NotifyRoomUpdate")
	defer span.Finish()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling room event: %w", err)
	}

	topicName := fmt.Sprintf("room:%s", roomID)
	return c.NotifyTopic(ctx, topicName, data)
}

func (c *Client) NotifyAdminUpdate(ctx context.Context, event vibe.AdminEvent) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "NotifyAdminUpdate")
	defer span.Finish()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling admin event: %w", err)
	}

	return c.NotifyTopic(ctx, adminTopicName, data)
}

func (c *Client) NotifyRoomUpdates(ctx context.Context, roomID string, events []vibe.RoomEvent) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "NotifyRoomUpdates")
	defer span.Finish()

	var err error
	topicName := fmt.Sprintf("room:%s", roomID)

	for _, event := range events {
		data, marshalErr := json.Marshal(event)
		if marshalErr != nil {
			// Log error and continue? Or return first error?
			// User request implies batch update, but underlying system is per-message usually.
			// Let's return first error but try others? No, fail fast usually better or log.
			// Standard practice: stop on error.
			if err == nil {
				err = fmt.Errorf("error marshaling room event: %w", marshalErr)
			}
			continue
		}

		if notifyErr := c.NotifyTopic(ctx, topicName, data); notifyErr != nil {
			if err == nil {
				err = fmt.Errorf("error notifying topic: %w", notifyErr)
			}
		}
	}

	return err
}

func (c *Client) Subscribe(topicName string) (*vibe.SubscriptionContainer, error) {
	topic := c.getTopic(topicName)

	subscription, err := topic.createSubscription()
	if err != nil {
		return nil, fmt.Errorf("error adding subscriber to topic %s: %w", topicName, err)
	}

	return &vibe.SubscriptionContainer{
		Subscription: subscription,
	}, nil
}

func (c *Client) getTopic(name string) *Topic {
	c.mu.Lock()
	defer c.mu.Unlock()

	s, exists := c.topics[name]
	if exists {
		return s
	}

	c.topics[name] = &Topic{
		name:          name,
		subscriptions: make(map[string]*Subscription),
	}

	return c.topics[name]
}

type Subscription struct {
	id       string
	mu       sync.Mutex
	closed   bool
	messages chan []byte
	parent   *Topic
}

func (s *Subscription) Destroy() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}

	s.closed = true
	close(s.messages)
	s.mu.Unlock()

	s.parent.removeSubscription(s.id)
}

func (s *Subscription) Listen() chan []byte {
	return s.messages
}

type Topic struct {
	name          string
	mu            sync.Mutex
	subscriptions map[string]*Subscription
}

func (t *Topic) publish(msg []byte) {
	t.mu.Lock()

	subscriptions := make([]*Subscription, 0, len(t.subscriptions))
	for _, subscription := range t.subscriptions {
		subscriptions = append(subscriptions, subscription)
	}

	t.mu.Unlock()

	for _, subscription := range subscriptions {
		subscription.mu.Lock()

		if !subscription.closed {
			select {
			case subscription.messages <- msg:
			default:
				log.Printf("dropped message due to full buffer on topic %s subscription subscription-id %s", t.name, subscription.id)
			}
		}

		subscription.mu.Unlock()
	}
}

func (t *Topic) createSubscription() (*Subscription, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	id, err := generateSubscriptionID()
	if err != nil {
		return nil, fmt.Errorf("error creating subscription id: %w", err)
	}

	subscription := &Subscription{
		id:       id,
		parent:   t,
		messages: make(chan []byte, 32),
	}

	t.subscriptions[id] = subscription

	return subscription, nil
}

func (t *Topic) removeSubscription(subscriptionID string) {
	t.mu.Lock()
	delete(t.subscriptions, subscriptionID)
	t.mu.Unlock()
}

func generateSubscriptionID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("error generating subscription-id: %w", err)
	}

	return id.String(), nil
}

const adminTopicName string = "admin"
