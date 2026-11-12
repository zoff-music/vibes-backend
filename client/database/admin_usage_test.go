package database

import (
	"context"
	"database/sql"
	"os"
	"testing"
)

type adminUsageNameCase struct {
	name     string
	roomID   string
	roomName string
}

// Uses temporary tables only; point VIBES_TEST_DATABASE_URL at a test Postgres.
func TestAdminUsageRoomNames(t *testing.T) {
	connection := os.Getenv("VIBES_TEST_DATABASE_URL")
	if connection == "" {
		t.Skip("VIBES_TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("pgx", connection)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`
 CREATE TEMP TABLE rooms (id TEXT PRIMARY KEY, name TEXT, room_type TEXT);
 CREATE TEMP TABLE chat_usage (room_id TEXT, created_at TIMESTAMPTZ, message_count BIGINT);
 CREATE TEMP TABLE room_search_usage (room_id TEXT, provider TEXT, created_at TIMESTAMPTZ, search_count BIGINT, cached_count BIGINT);
 CREATE TEMP TABLE listener_usage (created_at TIMESTAMPTZ, listener_count BIGINT);
 CREATE TEMP TABLE room_listener_usage (room_id TEXT, created_at TIMESTAMPTZ, listener_count BIGINT);
 INSERT INTO rooms VALUES ('named', 'Evening listening', 'MUSIC');
 INSERT INTO chat_usage VALUES ('named', NOW(), 3), ('deleted', NOW(), 4);
 INSERT INTO room_search_usage VALUES ('named', 'youtube', NOW(), 3, 1), ('deleted', 'youtube', NOW(), 4, 2);
 INSERT INTO listener_usage VALUES (DATE_TRUNC('MINUTE', NOW()), 7);
 INSERT INTO room_listener_usage VALUES ('named', DATE_TRUNC('MINUTE', NOW()), 3), ('deleted', DATE_TRUNC('MINUTE', NOW()), 4);
 `)
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{DB: db}
	err = client.prepareListAdminMessageUsageStmt()
	if err != nil {
		t.Fatal(err)
	}
	err = client.prepareListAdminRoomSearchUsageStmt()
	if err != nil {
		t.Fatal(err)
	}
	err = client.prepareListAdminRoomListenerUsageStmt()
	if err != nil {
		t.Fatal(err)
	}
	tests := []adminUsageNameCase{
		{name: "current room has a display name", roomID: "named", roomName: "Evening listening"},
		{name: "deleted room retains its history", roomID: "deleted", roomName: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := client.ListAdminMessageUsage(context.Background(), tt.roomID)
			if err != nil {
				t.Fatal(err)
			}
			if len(messages.RoomPoints) != 3 {
				t.Fatalf("expected three message windows, got %d", len(messages.RoomPoints))
			}
			for _, point := range messages.RoomPoints {
				if point.RoomName != tt.roomName {
					t.Errorf("message name = %q, want %q", point.RoomName, tt.roomName)
				}
			}
			searches, err := client.ListAdminRoomSearchUsage(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			searchMatches := 0
			for _, point := range searches {
				if point.RoomID != tt.roomID {
					continue
				}
				searchMatches++
				if point.RoomName != tt.roomName {
					t.Errorf("search name = %q, want %q", point.RoomName, tt.roomName)
				}
				if point.Total != point.Live+point.Cached {
					t.Error("search cache split does not reconcile")
				}
			}
			if searchMatches != 2 {
				t.Errorf("expected two search windows, got %d", searchMatches)
			}
			listeners, err := client.ListAdminRoomListenerUsage(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			listenerMatches := 0
			for _, point := range listeners {
				if point.RoomID != tt.roomID {
					continue
				}
				listenerMatches++
				if point.RoomName != tt.roomName {
					t.Errorf("listener name = %q, want %q", point.RoomName, tt.roomName)
				}
			}
			if listenerMatches != 4 {
				t.Errorf("expected four audience windows, got %d", listenerMatches)
			}
		})
	}
}
