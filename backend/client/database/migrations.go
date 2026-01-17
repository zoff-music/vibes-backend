package database

import (
	"context"

	"github.com/zoff-music/vibes/monitoring/opentracing"
)

func (c *Client) migrateSchema(ctx context.Context) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "migrateSchema")
	defer span.Finish()

	return nil
}
