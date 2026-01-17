package pubsub

import (
	"context"
	"fmt"
	"github.com/zoff-music/cibes/monitoring/trace"

	ps "cloud.google.com/go/pubsub/v2"

	"github.com/zoff-music/cibes/config"
)

// Client holds the PubSub client.
type Client struct {
	PubSubClient *ps.Client
}

// Init sets up a new pubsub client.
func (c *Client) Init(config *config.Config) error {
	psClient, err := ps.NewClient(context.Background(), config.PubSubProjectName)
	if err != nil {
		return fmt.Errorf("error creating pubsub client: %w", err)
	}
	c.PubSubClient = psClient
	return nil
}

func (c *Client) send(ctx context.Context, topicName string, data []byte) error {
	spanCarrier := trace.InjectIntoCarrier(ctx)

	topic := c.PubSubClient.Publisher(topicName)
	defer topic.Stop()
	var results []*ps.PublishResult
	r := topic.Publish(ctx, &ps.Message{Data: data, Attributes: spanCarrier})
	results = append(results, r)
	for _, r := range results {
		_, err := r.Get(ctx)
		if err != nil {
			return fmt.Errorf("error publishing to pubsub: %v", err)
		}
	}
	return nil
}
