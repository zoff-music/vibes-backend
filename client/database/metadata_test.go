package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/zoff-music/vibes-backend/internalerror"
	neturl "net/url"
	"os"
	"testing"
	"time"

	"github.com/zoff-music/vibes-backend/vibe"
)

func TestMetadataScheduling(t *testing.T) {
	url := os.Getenv("VIBES_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("VIBES_TEST_DATABASE_URL is required for Postgres integration tests")
	}
	tests := []struct {
		name string
		room string
		due  bool
	}{
		{name: "matching room schedules immediately", room: "room", due: true},
		{name: "other room cannot schedule item", room: "other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := sql.Open("pgx", url)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { db.Close() })
			db.SetMaxOpenConns(1)
			ctx := context.Background()
			_, err = db.ExecContext(ctx, `CREATE TEMP TABLE playlist_items (
 id TEXT PRIMARY KEY, room_id TEXT, source_type TEXT, source_id TEXT,
 title TEXT, publisher TEXT, thumbnail_url TEXT, duration INT, provider_url TEXT, playback_restriction TEXT,
 metadata_updated_at TIMESTAMPTZ DEFAULT NOW(), metadata_refresh_after TIMESTAMPTZ DEFAULT NOW() + INTERVAL '21 days'
 )`)
			if err != nil {
				t.Fatal(err)
			}
			_, err = db.ExecContext(ctx, `INSERT INTO playlist_items (id,room_id,source_type,source_id) VALUES ('item','room','youtube','video')`)
			if err != nil {
				t.Fatal(err)
			}
			_, err = db.ExecContext(ctx, `CREATE TEMP TABLE skip_votes(room_id TEXT,playlist_item_id TEXT); CREATE TEMP TABLE playlist_item_votes(room_id TEXT,playlist_item_id TEXT); CREATE TEMP TABLE playback_state(room_id TEXT,current_playlist_item_id TEXT,is_playing BOOLEAN,position_ms BIGINT,updated_at TIMESTAMPTZ)`)
			if err != nil {
				t.Fatal(err)
			}
			c := Client{DB: db}
			err = c.prepareSchedulePlaylistItemMetadataRefreshStmt()
			if err != nil {
				t.Fatal(err)
			}
			err = c.prepareClaimPlaylistItemMetadataRefreshStmt()
			if err != nil {
				t.Fatal(err)
			}
			err = c.prepareApplyProviderMetadataStmt()
			if err != nil {
				t.Fatal(err)
			}
			err = c.SchedulePlaylistItemMetadataRefresh(ctx, tt.room, "item")
			if err != nil {
				t.Fatal(err)
			}
			var due bool
			err = db.QueryRowContext(ctx, `SELECT metadata_refresh_after <= NOW() FROM playlist_items WHERE id='item'`).Scan(&due)
			if err != nil {
				t.Fatal(err)
			}
			if due != tt.due {
				t.Fatalf("due=%v", due)
			}
			refresh, err := c.ClaimPlaylistItemMetadataRefresh(ctx, "youtube", 5*time.Minute)
			if !tt.due {
				if err == nil {
					t.Fatal("claimed an item before its original deadline")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			second, err := c.ClaimPlaylistItemMetadataRefresh(ctx, "youtube", 5*time.Minute)
			if err == nil || second != nil {
				t.Fatal("claimed the same item while leased")
			}
			if !refresh.Force {
				t.Fatal("reported error did not request fresh provider metadata")
			}
			verifiedAt := time.Now().Add(-24 * time.Hour)
			_, err = c.ApplyProviderMetadata(ctx, *refresh, vibe.ProviderMetadata{Item: vibe.ProviderItem{Title: "Updated"}, CheckedAt: verifiedAt}, 5*24*time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			var interval float64
			err = db.QueryRowContext(ctx, `SELECT EXTRACT(EPOCH FROM metadata_refresh_after-metadata_updated_at) FROM playlist_items WHERE id='item'`).Scan(&interval)
			if err != nil {
				t.Fatal(err)
			}
			if interval != (5 * 24 * time.Hour).Seconds() {
				t.Fatalf("refresh interval=%v", interval)
			}
		})
	}
}

// Each claim is one source job, irrespective of the number of room entries.
func TestMetadataSourceJobs(t *testing.T) {
	url := os.Getenv("VIBES_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("VIBES_TEST_DATABASE_URL is required")
	}
	tests := []struct {
		name                                  string
		entries, unique                       int
		unavailable, quota, reportDuringClaim bool
	}{
		{name: "200000 entries produce 5000 jobs", entries: 200000, unique: 5000},
		{name: "unavailable removes all duplicates", entries: 40, unique: 1, unavailable: true},
		{name: "quota delays all duplicates", entries: 40, unique: 1, quota: true},
		{name: "new error report survives completion", entries: 40, unique: 1, reportDuringClaim: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := sql.Open("pgx", url)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { db.Close() })
			db.SetMaxOpenConns(1)
			ctx := context.Background()
			_, err = db.ExecContext(ctx, `CREATE TEMP TABLE playlist_items (
   id TEXT PRIMARY KEY,room_id TEXT,source_type TEXT,source_id TEXT,
   title TEXT,publisher TEXT,thumbnail_url TEXT,duration INT,provider_url TEXT,playback_restriction TEXT,
   metadata_updated_at TIMESTAMPTZ DEFAULT NOW()-INTERVAL '6 days',metadata_refresh_after TIMESTAMPTZ DEFAULT NOW()-INTERVAL '1 day');
   CREATE INDEX ON playlist_items(source_type,metadata_refresh_after);
   CREATE INDEX ON playlist_items(room_id,source_type,source_id);
   CREATE TEMP TABLE skip_votes(room_id TEXT,playlist_item_id TEXT);
   CREATE TEMP TABLE playlist_item_votes(room_id TEXT,playlist_item_id TEXT);
   CREATE TEMP TABLE playback_state(room_id TEXT,current_playlist_item_id TEXT,is_playing BOOLEAN,position_ms BIGINT,updated_at TIMESTAMPTZ)`)
			if err != nil {
				t.Fatal(err)
			}
			_, err = db.ExecContext(ctx, `INSERT INTO playlist_items(id,room_id,source_type,source_id)
   SELECT 'item-'||a,'room-'||(a%40),'youtube','video-'||(a%$2) FROM generate_series(1,$1) a`, tt.entries, tt.unique)
			if err != nil {
				t.Fatal(err)
			}
			_, err = db.ExecContext(ctx, `INSERT INTO skip_votes SELECT room_id,id FROM playlist_items;
   INSERT INTO playlist_item_votes SELECT room_id,id FROM playlist_items;
   INSERT INTO playback_state SELECT DISTINCT ON(room_id) room_id,id,TRUE,100,NOW() FROM playlist_items;
   INSERT INTO playlist_items(id,room_id,source_type,source_id,metadata_refresh_after) VALUES ('other-provider','room','soundcloud','video-0',NOW()+INTERVAL '21 days');
   ANALYZE playlist_items`)
			if err != nil {
				t.Fatal(err)
			}
			c := Client{DB: db}
			for _, prepare := range []func() error{c.prepareClaimPlaylistItemMetadataRefreshStmt, c.prepareApplyProviderMetadataStmt, c.prepareDeferPlaylistItemMetadataRefreshStmt, c.prepareSchedulePlaylistItemMetadataRefreshStmt} {
				err = prepare()
				if err != nil {
					t.Fatal(err)
				}
			}
			checkedAt := time.Now()
			for job := 0; job < tt.unique; job++ {
				refresh, err := c.ClaimPlaylistItemMetadataRefresh(ctx, "youtube", 5*time.Minute)
				if err != nil {
					t.Fatalf("job %d: %v", job, err)
				}
				if refresh.Force {
					t.Fatal("scheduled source unexpectedly forced")
				}
				if tt.reportDuringClaim {
					err = c.SchedulePlaylistItemMetadataRefresh(ctx, "room-1", "item-1")
					if err != nil {
						t.Fatal(err)
					}
				}
				if tt.quota {
					err = c.DeferPlaylistItemMetadataRefresh(ctx, *refresh, 24*time.Hour)
					if err != nil {
						t.Fatal(err)
					}
					continue
				}
				affected, err := c.ApplyProviderMetadata(ctx, *refresh, vibe.ProviderMetadata{Item: vibe.ProviderItem{Title: "Updated"}, Unavailable: tt.unavailable, CheckedAt: checkedAt}, 5*24*time.Hour)
				if err != nil {
					t.Fatalf("job %d: %v", job, err)
				}
				if len(affected.RoomIDs) == 0 {
					t.Fatal("no affected rooms")
				}
			}
			refresh, err := c.ClaimPlaylistItemMetadataRefresh(ctx, "youtube", 5*time.Minute)
			if tt.reportDuringClaim {
				if err != nil || !refresh.Force {
					t.Fatalf("new report lost: %v %v", refresh, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("duplicate scheduled job: %v", refresh)
			}
			var count int
			err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM playlist_items WHERE source_type='soundcloud' AND metadata_refresh_after>NOW()+INTERVAL '20 days'`).Scan(&count)
			if err != nil || count != 1 {
				t.Fatalf("other provider modified: %d %v", count, err)
			}
			if tt.unavailable {
				for _, query := range []string{`SELECT COUNT(*) FROM playlist_items WHERE source_type='youtube'`, `SELECT COUNT(*) FROM skip_votes`, `SELECT COUNT(*) FROM playlist_item_votes`, `SELECT COUNT(*) FROM playback_state WHERE is_playing OR current_playlist_item_id IS NOT NULL`} {
					err = db.QueryRowContext(ctx, query).Scan(&count)
					if err != nil || count != 0 {
						t.Fatalf("removal incomplete: %d %v", count, err)
					}
				}
				return
			}
			if tt.quota {
				err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM playlist_items WHERE source_type='youtube' AND metadata_refresh_after>NOW()+INTERVAL '23 hours'`).Scan(&count)
			} else {
				err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM playlist_items WHERE source_type='youtube' AND title='Updated' AND metadata_updated_at=$1 AND metadata_refresh_after=$1::timestamptz+INTERVAL '5 days'`, checkedAt).Scan(&count)
			}
			if err != nil || count != tt.entries {
				t.Fatalf("updated %d/%d: %v", count, tt.entries, err)
			}
		})
	}
}

func TestConcurrentMetadataSourceClaims(t *testing.T) {
	url := os.Getenv("VIBES_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("VIBES_TEST_DATABASE_URL is required")
	}
	tests := []struct {
		name    string
		workers int
	}{{name: "one source lease across twenty workers", workers: 20}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := neturl.Parse(url)
			if err != nil {
				t.Fatal(err)
			}
			schema := fmt.Sprintf("metadata_claim_%d", time.Now().UnixNano())
			query := parsed.Query()
			query.Set("search_path", schema)
			parsed.RawQuery = query.Encode()
			db, err := sql.Open("pgx", parsed.String())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { db.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); db.Close() })
			ctx := context.Background()
			_, err = db.ExecContext(ctx, "CREATE SCHEMA "+schema)
			if err != nil {
				t.Fatal(err)
			}
			_, err = db.ExecContext(ctx, `CREATE TABLE playlist_items(id TEXT PRIMARY KEY,room_id TEXT,source_type TEXT,source_id TEXT,metadata_updated_at TIMESTAMPTZ,metadata_refresh_after TIMESTAMPTZ);
   INSERT INTO playlist_items SELECT 'item-'||a,'room','youtube','video',NOW()-INTERVAL '6 days',NOW()-INTERVAL '1 day' FROM generate_series(1,40) a`)
			if err != nil {
				t.Fatal(err)
			}
			c := Client{DB: db}
			err = c.prepareClaimPlaylistItemMetadataRefreshStmt()
			if err != nil {
				t.Fatal(err)
			}
			start := make(chan bool)
			results := make(chan error, tt.workers)
			for worker := 0; worker < tt.workers; worker++ {
				go func() {
					<-start
					_, claimErr := c.ClaimPlaylistItemMetadataRefresh(ctx, "youtube", 5*time.Minute)
					results <- claimErr
				}()
			}
			close(start)
			claims := 0
			for worker := 0; worker < tt.workers; worker++ {
				err = <-results
				if err == nil {
					claims++
					continue
				}
				var expected internalerror.ErrExpected
				if !errors.As(err, &expected) {
					t.Errorf("unexpected claim failure: %v", err)
				}
			}
			if claims != 1 {
				t.Fatalf("claimed %d jobs for one source", claims)
			}
		})
	}
}
