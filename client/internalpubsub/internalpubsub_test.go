package internalpubsub

import "testing"

type topicPublishTest struct {
	expectedBufferedMessages int
	expectedClosed           bool
	expectedSubscriptions    int
	name                     string
	publishCount             int
}

func TestTopicPublishDisconnectsSlowSubscribers(t *testing.T) {
	tests := []topicPublishTest{
		{
			name:                     "keeps a subscriber connected while messages fit",
			publishCount:             1,
			expectedBufferedMessages: 1,
			expectedSubscriptions:    1,
		},
		{
			name:                     "disconnects a subscriber instead of dropping an overflow message",
			publishCount:             subscriptionBufferSize + 1,
			expectedBufferedMessages: subscriptionBufferSize,
			expectedClosed:           true,
			expectedSubscriptions:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic := &Topic{
				name:          "room:electro",
				subscriptions: make(map[string]*Subscription),
			}
			subscription, err := topic.createSubscription()
			if err != nil {
				t.Fatalf("expected subscription creation to succeed: %v", err)
			}

			for index := 0; index < tt.publishCount; index++ {
				topic.publish([]byte("event"))
			}

			subscription.mu.Lock()
			closed := subscription.closed
			subscription.mu.Unlock()
			if closed != tt.expectedClosed {
				t.Fatalf("expected closed to be %t, got %t", tt.expectedClosed, closed)
			}

			topic.mu.Lock()
			subscriptionCount := len(topic.subscriptions)
			topic.mu.Unlock()
			if subscriptionCount != tt.expectedSubscriptions {
				t.Fatalf(
					"expected %d subscriptions, got %d",
					tt.expectedSubscriptions,
					subscriptionCount,
				)
			}

			bufferedMessages := 0
			for bufferedMessages < tt.expectedBufferedMessages {
				_, ok := <-subscription.Listen()
				if !ok {
					t.Fatalf(
						"expected %d buffered messages, channel closed after %d",
						tt.expectedBufferedMessages,
						bufferedMessages,
					)
				}
				bufferedMessages++
			}

			if tt.expectedClosed {
				_, ok := <-subscription.Listen()
				if ok {
					t.Fatal("expected subscription channel to be closed")
				}
				return
			}

			subscription.Destroy()
		})
	}
}
