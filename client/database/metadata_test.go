package database

import (
	"context"
	"database/sql"
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
			c := Client{DB: db}
			err = c.prepareSchedulePlaylistItemMetadataRefreshStmt()
			if err != nil {
				t.Fatal(err)
			}
			err = c.prepareClaimPlaylistItemMetadataRefreshStmt()
			if err != nil {
				t.Fatal(err)
			}
			err = c.prepareRefreshPlaylistItemMetadataStmt()
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
			err = c.RefreshPlaylistItemMetadata(ctx, *refresh, vibe.ProviderItem{Title: "Updated"}, 5*24*time.Hour)
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
