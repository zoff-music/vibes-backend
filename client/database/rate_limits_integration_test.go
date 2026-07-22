package database

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/vibe"
)

func TestConsumeRateLimitIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	cfg := config.Config{
		DatabaseURL:          databaseURL,
		DatabaseMaxConns:     2,
		DatabaseMaxIdleConns: 1,
	}
	client := &Client{}
	err := client.Init(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("error initializing database client: %v", err)
	}
	t.Cleanup(func() {
		closeErr := client.Close()
		if closeErr != nil {
			t.Errorf("error closing database client: %v", closeErr)
		}
	})

	routeName := "rate-limit-integration-" + time.Now().Format("20060102150405.000000000")
	request := vibe.RateLimitRequest{
		RouteName:          routeName,
		DeviceIdentityHash: strings.Repeat("a", 64),
		IPIdentityHash:     strings.Repeat("b", 64),
		Rate:               time.Minute,
		DeviceLimit:        2,
		IPLimit:            20,
	}

	first, err := client.ConsumeRateLimit(context.Background(), request)
	if err != nil {
		t.Fatalf("error consuming first rate limit: %v", err)
	}
	if !first.Allowed {
		t.Fatal("expected first request to be allowed")
	}

	second, err := client.ConsumeRateLimit(context.Background(), request)
	if err != nil {
		t.Fatalf("error consuming second rate limit: %v", err)
	}
	if !second.Allowed {
		t.Fatal("expected second request to be allowed")
	}

	third, err := client.ConsumeRateLimit(context.Background(), request)
	if err != nil {
		t.Fatalf("error consuming third rate limit: %v", err)
	}
	if third.Allowed {
		t.Fatal("expected third request to be rate limited")
	}
	if third.RetryAfter <= 0 {
		t.Fatalf("expected positive retry duration, got %s", third.RetryAfter)
	}

	var ipRequestCount int
	err = client.DB.QueryRowContext(
		context.Background(),
		"SELECT request_count FROM rate_limits WHERE route_name = $1 AND scope = 'ip'",
		routeName,
	).Scan(&ipRequestCount)
	if err != nil {
		t.Fatalf("error getting IP request count: %v", err)
	}
	if ipRequestCount != 2 {
		t.Fatalf("expected blocked device request not to consume the IP bucket, got %d", ipRequestCount)
	}

	request.DeviceIdentityHash = strings.Repeat("c", 64)
	differentDevice, err := client.ConsumeRateLimit(context.Background(), request)
	if err != nil {
		t.Fatalf("error consuming rate limit for another device: %v", err)
	}
	if !differentDevice.Allowed {
		t.Fatal("expected another device on the same IP to have its own allowance")
	}

	_, err = client.DB.ExecContext(
		context.Background(),
		`UPDATE rate_limits
		 SET window_started_at = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') - INTERVAL '2 seconds',
		     expires_at = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') - INTERVAL '1 second'
		 WHERE route_name = $1`,
		routeName,
	)
	if err != nil {
		t.Fatalf("error expiring integration rate limits: %v", err)
	}

	err = client.DeleteExpiredRateLimits(context.Background())
	if err != nil {
		t.Fatalf("error deleting expired rate limits: %v", err)
	}

	var remaining int
	err = client.DB.QueryRowContext(
		context.Background(),
		"SELECT COUNT(*) FROM rate_limits WHERE route_name = $1",
		routeName,
	).Scan(&remaining)
	if err != nil {
		t.Fatalf("error counting remaining rate limits: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected no remaining rate limits, got %d", remaining)
	}
}
