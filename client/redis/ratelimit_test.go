package redis

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/vibe"
)

type rateLimitStep struct {
	deviceHash string
	wantAllow  bool
	wait       time.Duration
}

type rateLimitClientTestCase struct {
	name        string
	rate        time.Duration
	deviceLimit int
	ipLimit     int
	steps       []rateLimitStep
}

func TestConsumeRateLimit(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("TEST_REDIS_URL is not configured")
	}

	testCases := []rateLimitClientTestCase{
		{
			name:        "device window rolls forward",
			rate:        200 * time.Millisecond,
			deviceLimit: 2,
			ipLimit:     20,
			steps: []rateLimitStep{
				{deviceHash: strings.Repeat("a", 64), wantAllow: true},
				{deviceHash: strings.Repeat("a", 64), wantAllow: true},
				{deviceHash: strings.Repeat("a", 64), wantAllow: false},
				{deviceHash: strings.Repeat("a", 64), wantAllow: true, wait: 250 * time.Millisecond},
			},
		},
		{
			name:        "denied device does not consume IP window",
			rate:        time.Second,
			deviceLimit: 1,
			ipLimit:     2,
			steps: []rateLimitStep{
				{deviceHash: strings.Repeat("a", 64), wantAllow: true},
				{deviceHash: strings.Repeat("a", 64), wantAllow: false},
				{deviceHash: strings.Repeat("b", 64), wantAllow: true},
				{deviceHash: strings.Repeat("c", 64), wantAllow: false},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := config.Config{RedisURL: redisURL}
			client := &Client{}
			err := client.Init(context.Background(), &cfg)
			if err != nil {
				t.Fatalf("error initializing redis client: %v", err)
			}
			t.Cleanup(func() {
				closeErr := client.Close()
				if closeErr != nil {
					t.Errorf("error closing redis client: %v", closeErr)
				}
			})

			routeName := "test-" + uuid.NewString()
			for index, step := range testCase.steps {
				if step.wait > 0 {
					time.Sleep(step.wait)
				}

				result, consumeErr := client.ConsumeRateLimit(context.Background(), vibe.RateLimitRequest{
					RouteName:          routeName,
					DeviceIdentityHash: step.deviceHash,
					IPIdentityHash:     strings.Repeat("f", 64),
					Rate:               testCase.rate,
					DeviceLimit:        testCase.deviceLimit,
					IPLimit:            testCase.ipLimit,
				})
				if consumeErr != nil {
					t.Fatalf("step %d returned an error: %v", index, consumeErr)
				}
				if result.Allowed != step.wantAllow {
					t.Fatalf("step %d expected allowed %t, got %t", index, step.wantAllow, result.Allowed)
				}
				if !step.wantAllow && result.RetryAfter <= 0 {
					t.Fatalf("step %d expected a positive retry duration, got %s", index, result.RetryAfter)
				}
			}
		})
	}
}
