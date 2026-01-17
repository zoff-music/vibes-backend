package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zoff-music/vibes/example"
	"github.com/zoff-music/vibes/monitoring/opentracing"
)

// NotifyExampleData is used to publish example data to pubsub.
func (c *Client) NotifyExampleData(ctx context.Context, exampleData example.Data) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "NotifyExampleData")
	defer span.Finish()

	data, err := json.Marshal(exampleData)
	if err != nil {
		return fmt.Errorf("error marshalling example data to send to pubsub: %w", err)
	}
	err = c.send(ctx, "whatever-topic-name", data)
	if err != nil {
		return fmt.Errorf("error sending example data message to pubsub: %w", err)
	}
	return nil
}
