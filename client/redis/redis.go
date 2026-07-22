package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
)

type Client struct {
	Prefix string
	Redis  *redis.Pool
}

func (c *Client) Init(ctx context.Context, cfg *config.Config) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "Init")
	defer span.End()

	c.Prefix = redisPrefix
	c.Redis = &redis.Pool{
		MaxIdle:   80,
		MaxActive: 1200,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(
				cfg.RedisURL,
				redis.DialConnectTimeout(5*time.Second),
				redis.DialReadTimeout(5*time.Second),
				redis.DialWriteTimeout(5*time.Second),
			)
		},
		TestOnBorrow: func(connection redis.Conn, lastUsed time.Time) error {
			if time.Since(lastUsed) < time.Minute {
				return nil
			}

			_, err := connection.Do("PING")
			if err != nil {
				return fmt.Errorf("error pinging redis connection: %w", err)
			}
			return nil
		},
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connection, err := c.Redis.GetContext(cctx)
	if err != nil {
		closeErr := c.Redis.Close()
		if closeErr != nil {
			return fmt.Errorf("error getting redis connection: %w; error closing redis pool: %v", err, closeErr)
		}
		return fmt.Errorf("error getting redis connection: %w", err)
	}
	defer connection.Close()

	_, err = redis.DoContext(connection, cctx, "PING")
	if err != nil {
		closeErr := c.Redis.Close()
		if closeErr != nil {
			return fmt.Errorf("error pinging redis: %w; error closing redis pool: %v", err, closeErr)
		}
		return fmt.Errorf("error pinging redis: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	if c.Redis == nil {
		return nil
	}

	err := c.Redis.Close()
	if err != nil {
		return fmt.Errorf("error closing redis pool: %w", err)
	}

	return nil
}

func (c *Client) getKeyWithPrefix(key string) string {
	return fmt.Sprintf("%s:%s", c.Prefix, key)
}

const redisPrefix = "vibes"
