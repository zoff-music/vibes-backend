package database

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/zoff-music/vibes-backend/vibe"
)

func TestGetRoomsV3(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL integration tests")
	}
	tests := []struct {
		name  string
		ids   []string
		from  int
		to    int
		total int
		want  []vibe.PublicRoomV3
	}{
		{name: "known private and public rooms retain order", to: 2, total: 2, ids: []string{"private", "public"}, want: []vibe.PublicRoomV3{{ID: "private", Name: "Private", RoomType: vibe.RoomTypeWatch, ListenerCount: 1, PlaylistItemCount: 1}, {ID: "public", Name: "Public", RoomType: vibe.RoomTypeMusic, ListenerCount: 2, PlaylistItemCount: 2}}},
		{name: "missing and flagged rooms omitted", ids: []string{"missing", "flagged"}, want: []vibe.PublicRoomV3{}},
		{name: "empty rooms remain available", to: 2, total: 1, ids: []string{"empty"}, want: []vibe.PublicRoomV3{{ID: "empty", Name: "Empty", RoomType: vibe.RoomTypeMusic}}},
		{name: "public browsing excludes unrequested private rooms", to: 2, total: 1, want: []vibe.PublicRoomV3{{ID: "public", Name: "Public", RoomType: vibe.RoomTypeMusic, ListenerCount: 2, PlaylistItemCount: 2}}},
		{name: "pagination applies after explicit IDs", ids: []string{"private", "public"}, from: 1, to: 1, total: 2, want: []vibe.PublicRoomV3{{ID: "public", Name: "Public", RoomType: vibe.RoomTypeMusic, ListenerCount: 2, PlaylistItemCount: 2}}},
		{name: "past last page keeps total", ids: []string{"private", "public"}, from: 4, to: 5, total: 2, want: []vibe.PublicRoomV3{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := sql.Open("pgx", url)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			db.SetMaxOpenConns(1)
			_, err = db.ExecContext(context.Background(), `
    CREATE TEMP TABLE rooms (id TEXT PRIMARY KEY, name TEXT, room_type TEXT, flagged BOOLEAN, admin_password_hash TEXT);
 CREATE TEMP TABLE room_settings (room_id TEXT, is_public BOOLEAN);
    CREATE TEMP TABLE playlist_items (room_id TEXT, source_type TEXT);
    CREATE TEMP TABLE room_users (room_id TEXT, is_active_listener BOOLEAN, is_cast_receiver BOOLEAN, last_seen_at TIMESTAMPTZ);
    INSERT INTO rooms VALUES ('private','Private','WATCH',false,'hash'),('public','Public','MUSIC',false,'hash'),('flagged','Flagged','MUSIC',true,'hash'),('empty','Empty','MUSIC',false,'');
 INSERT INTO room_settings VALUES ('private',false),('public',true),('flagged',true),('empty',false);
    INSERT INTO playlist_items VALUES ('private','youtube'),('public','youtube'),('public','youtube'),('public','soundcloud');
    INSERT INTO room_users VALUES ('private',false,true,NOW()),('public',true,false,NOW()),('public',true,false,NOW()),('public',false,true,NOW()),('public',false,false,NOW()),('public',true,false,NOW()-INTERVAL '1 minute');
   `)
			if err != nil {
				t.Fatal(err)
			}
			client := Client{DB: db, enabledProviders: []string{"youtube"}}
			err = client.prepareGetRoomsV3Stmt()
			if err != nil {
				t.Fatal(err)
			}
			defer client.GetRoomsV3Statement.Close()
			result, err := client.GetRoomsV3(context.Background(), vibe.RoomsSearchV3{IDs: tt.ids, From: tt.from, To: tt.to})
			if err != nil {
				t.Fatal(err)
			}
			if result.Total != tt.total || result.Count != len(tt.want) {
				t.Fatalf("total/count=%d/%d, want %d/%d", result.Total, result.Count, tt.total, len(tt.want))
			}
			if !reflect.DeepEqual(result.Rooms, tt.want) {
				t.Fatalf("rooms=%+v, want %+v", result.Rooms, tt.want)
			}
		})
	}
}
