package database

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/zoff-music/vibes-backend/vibe"
)

func TestSearchPublicRooms(t *testing.T) {
	databaseURL := os.Getenv("VIBES_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set VIBES_TEST_DATABASE_URL to a migrated local PostgreSQL database")
	}
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal("invalid local test database URL")
	}
	if parsedURL.Hostname() != "127.0.0.1" && parsedURL.Hostname() != "localhost" && parsedURL.Hostname() != "::1" {
		t.Fatal("room integration tests require a loopback database")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open local database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	statements := []string{
		`SET TIME ZONE 'UTC'`,
		`CREATE TEMP TABLE rooms (LIKE public.rooms INCLUDING DEFAULTS)`,
		`CREATE TEMP TABLE room_settings (LIKE public.room_settings INCLUDING DEFAULTS)`,
		`CREATE TEMP TABLE room_users (LIKE public.room_users INCLUDING DEFAULTS)`,
		`CREATE TEMP TABLE songs (LIKE public.songs INCLUDING DEFAULTS)`,
		`INSERT INTO rooms (id, name, admin_password_hash) VALUES
			('a', 'Session ambient', 'protected'), ('b', 'Session beats', 'protected'),
			('c', 'Session club', 'protected'), ('d', 'Session dusk', 'protected'),
			('e', 'Session empty', 'protected'), ('f', 'Session cast', 'protected'),
			('g', 'Session idle', 'protected'), ('literal', '100%_mix\room', 'protected'),
			('hidden', 'Session hidden', 'protected'), ('unprotected', 'Session open', NULL),
			('blank', 'Session blank', '')`,
		`INSERT INTO room_settings (room_id, is_public)
			SELECT id, id != 'hidden' FROM rooms`,
		`INSERT INTO room_users (id, room_id, is_active_listener, last_seen_at)
			SELECT i::text, a.id, TRUE, NOW() FROM rooms a
			CROSS JOIN generate_series(1, 3) i
			WHERE a.id IN ('a', 'b', 'c', 'd') AND (i <= 2 OR a.id = 'b')`,
		`INSERT INTO room_users (id, room_id, is_cast_receiver, last_seen_at)
			VALUES ('cast1', 'f', TRUE, NOW()), ('cast2', 'f', TRUE, NOW()),
			('cast1', 'b', TRUE, NOW())`,
		`INSERT INTO room_users (id, room_id, is_active_listener, last_seen_at)
			VALUES ('inactive', 'g', FALSE, NOW()), ('stale', 'g', TRUE, NOW() - INTERVAL '1 minute')`,
		`INSERT INTO songs (room_id, source_type, source_id, title, thumbnail_url, duration, added_by)
			SELECT a.id, 'youtube', i::text, 'Song', '', 210, 'guest'
			FROM rooms a CROSS JOIN generate_series(1, 3) i
			WHERE (a.id IN ('a', 'b', 'g') AND i = 1)
			OR a.id IN ('c', 'd') OR (a.id = 'f' AND i <= 2)`,
		`INSERT INTO songs (room_id, source_type, source_id, title, thumbnail_url, duration, added_by)
			VALUES ('a', 'soundcloud', 'disabled', 'Song', '', 210, 'guest')`,
	}
	for _, query := range statements {
		stmt, prepareErr := db.PrepareContext(ctx, query)
		if prepareErr != nil {
			t.Fatalf("prepare fixture: %v", prepareErr)
		}
		_, err = stmt.ExecContext(ctx)
		_ = stmt.Close()
		if err != nil {
			t.Fatalf("create fixture: %v", err)
		}
	}
	client := &Client{DB: db, enabledProviders: []string{"youtube"}}
	err = client.prepareSearchPublicRoomsStmt()
	if err != nil {
		t.Fatalf("prepare room search: %v", err)
	}
	t.Cleanup(func() { _ = client.SearchPublicRoomsStatement.Close() })

	tests := []struct {
		name   string
		search vibe.PublicRoomSearch
		ids    []string
		total  int
	}{
		{name: "all public rooms sorted by listeners songs and id", search: vibe.PublicRoomSearch{Query: "session", To: 9}, ids: []string{"b", "d", "c", "a", "f", "g", "e"}, total: 7},
		{name: "live excludes idle and stale listeners", search: vibe.PublicRoomSearch{Query: "session", Live: true, To: 9}, ids: []string{"b", "d", "c", "a", "f"}, total: 5},
		{name: "case insensitive substring", search: vibe.PublicRoomSearch{Query: "SSION", To: 9}, ids: []string{"b", "d", "c", "a", "f", "g", "e"}, total: 7},
		{name: "inclusive range", search: vibe.PublicRoomSearch{Query: "session", From: 2, To: 4}, ids: []string{"c", "a", "f"}, total: 7},
		{name: "past last page retains total", search: vibe.PublicRoomSearch{Query: "session", From: 20, To: 29}, ids: []string{}, total: 7},
		{name: "literal wildcards", search: vibe.PublicRoomSearch{Query: "%_", To: 9}, ids: []string{"literal"}, total: 1},
		{name: "literal backslash", search: vibe.PublicRoomSearch{Query: `\`, To: 9}, ids: []string{"literal"}, total: 1},
		{name: "empty result", search: vibe.PublicRoomSearch{Query: "missing", To: 9}, ids: []string{}, total: 0},
		{name: "private rooms never match", search: vibe.PublicRoomSearch{Query: "hidden", To: 9}, ids: []string{}, total: 0},
		{name: "public without admin password is excluded", search: vibe.PublicRoomSearch{Query: "Session open", To: 9}, ids: []string{}, total: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, searchErr := client.SearchPublicRooms(t.Context(), tt.search)
			if searchErr != nil {
				t.Fatalf("search rooms: %v", searchErr)
			}
			ids := []string{}
			for _, room := range result.Rooms {
				ids = append(ids, room.ID)
				if room.ID == "a" && room.SongCount != 1 {
					t.Fatalf("disabled source counted: %#v", room)
				}
				if room.ID == "f" && room.ListenerCount != 1 {
					t.Fatalf("cast listeners counted incorrectly: %#v", room)
				}
			}
			if !slices.Equal(ids, tt.ids) || result.Total != tt.total || result.Count != len(tt.ids) {
				t.Fatalf("result = %#v, want IDs %v and total %d", result, tt.ids, tt.total)
			}
			if result.From != tt.search.From || result.To != tt.search.From+max(0, len(tt.ids)-1) {
				t.Fatalf("unexpected range: %#v", result)
			}
		})
	}
}
