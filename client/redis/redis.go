package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
)

type Client struct {
	client        *goredis.Client
	consumeScript *goredis.Script
}

func (c *Client) Init(ctx context.Context, cfg *config.Config) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "Init")
	defer span.End()

	options, err := goredis.ParseURL(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("error parsing redis URL: %w", err)
	}

	c.client = goredis.NewClient(options)
	c.consumeScript = goredis.NewScript(consumeRateLimitScript)

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = c.client.Ping(cctx).Err()
	if err != nil {
		closeErr := c.client.Close()
		if closeErr != nil {
			return fmt.Errorf("error pinging redis: %w; error closing redis client: %v", err, closeErr)
		}
		return fmt.Errorf("error pinging redis: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}

	err := c.client.Close()
	if err != nil {
		return fmt.Errorf("error closing redis client: %w", err)
	}

	return nil
}
