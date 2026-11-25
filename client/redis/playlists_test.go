package redis

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/zoff-music/vibes-backend/vibe"
)

func TestMetadataCache(t *testing.T) {
	url := os.Getenv("VIBES_TEST_REDIS_URL")
	if url == "" {
		t.Skip("VIBES_TEST_REDIS_URL is required for Redis integration tests")
	}
	tests := []struct {
		name        string
		force       bool
		provider    string
		unavailable bool
	}{
		{name: "duplicates reuse verified metadata", provider: "youtube"},
		{name: "error report bypasses cached metadata", provider: "youtube", force: true},
		{name: "providers have separate keys", provider: "soundcloud"},
		{name: "unavailable result is cached", provider: "youtube", unavailable: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &redis.Pool{MaxIdle: 4, MaxActive: 20, Dial: func() (redis.Conn, error) {
				conn, err := redis.DialURL(url)
				if err != nil {
					return nil, fmt.Errorf("error dialing test redis: %w", err)
				}
				return conn, nil
			}}
			t.Cleanup(func() { pool.Close() })
			c := Client{Redis: pool, Prefix: "metadata-test:" + uuid.NewString()}
			ctx := context.Background()
			claim, err := c.GetOrClaimMetadata(ctx, "youtube", "video", false)
			if err != nil {
				t.Fatal(err)
			}
			if claim.Token == "" {
				t.Fatal("missing initial lease")
			}
			busy, err := c.GetOrClaimMetadata(ctx, "youtube", "video", false)
			if err != nil {
				t.Fatal(err)
			}
			if busy.Token != "" || !busy.Metadata.CheckedAt.IsZero() {
				t.Fatal("duplicate lookup acquired lease")
			}
			checkedAt := time.Now().UTC()
			metadata := vibe.ProviderMetadata{Item: vibe.ProviderItem{Title: "Verified"}, Unavailable: tt.unavailable, CheckedAt: checkedAt}
			completed, err := c.CompleteMetadata(ctx, "youtube", "video", claim.Token, metadata, time.Minute)
			if err != nil || !completed {
				t.Fatalf("complete=%v error=%v", completed, err)
			}
			next, err := c.GetOrClaimMetadata(ctx, tt.provider, "video", tt.force)
			if err != nil {
				t.Fatal(err)
			}
			newClaim := tt.force || tt.provider != "youtube"
			if newClaim != (next.Token != "") {
				t.Fatalf("unexpected claim token=%q", next.Token)
			}
			if !newClaim && (!next.Metadata.CheckedAt.Equal(checkedAt) || next.Metadata.Unavailable != tt.unavailable || next.Metadata.Item.Title != "Verified") {
				t.Fatalf("wrong cached result: %+v", next.Metadata)
			}
			stale, err := c.CompleteMetadata(ctx, "youtube", "video", claim.Token, metadata, time.Minute)
			if err != nil || stale {
				t.Fatalf("stale owner overwrote cache: %v %v", stale, err)
			}
			err = c.ReleaseMetadata(ctx, "youtube", "video", claim.Token)
			if err != nil {
				t.Fatal(err)
			}
			if tt.force {
				stillBusy, err := c.GetOrClaimMetadata(ctx, "youtube", "video", false)
				if err != nil || stillBusy.Token != "" {
					t.Fatalf("stale release removed new owner's lease: %v", err)
				}
			}
		})
	}
}

func TestMetadataConcurrentClaim(t *testing.T) {
	url := os.Getenv("VIBES_TEST_REDIS_URL")
	if url == "" {
		t.Skip("VIBES_TEST_REDIS_URL is required for Redis integration tests")
	}
	tests := []struct {
		name    string
		workers int
	}{{name: "one owner across workers", workers: 20}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &redis.Pool{MaxIdle: 20, MaxActive: 20, Dial: func() (redis.Conn, error) {
				conn, err := redis.DialURL(url)
				if err != nil {
					return nil, fmt.Errorf("error dialing test redis: %w", err)
				}
				return conn, nil
			}}
			t.Cleanup(func() { pool.Close() })
			c := Client{Redis: pool, Prefix: "metadata-test:" + uuid.NewString()}
			var wg sync.WaitGroup
			results := make(chan *vibe.ProviderMetadataLookup, tt.workers)
			for i := 0; i < tt.workers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					result, err := c.GetOrClaimMetadata(context.Background(), "youtube", "video", false)
					if err != nil {
						t.Error(err)
						return
					}
					results <- result
				}()
			}
			wg.Wait()
			close(results)
			owners := 0
			for result := range results {
				if result.Token != "" {
					owners++
				}
			}
			if owners != 1 {
				t.Fatalf("owners=%d", owners)
			}
		})
	}
}
