package database

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/zoff-music/vibes-backend/vibe"
)

func TestOwnedRemoteReports(t *testing.T) {
	dsn := os.Getenv("REMOTE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("REMOTE_TEST_DATABASE_URL must point to a disposable local PostgreSQL database")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`CREATE TEMP TABLE remote_controls (
		id TEXT PRIMARY KEY, owner_user_id TEXT NOT NULL,
		current_room_id TEXT NOT NULL, current_playlist_item_id TEXT NOT NULL,
		playback_position_ms BIGINT NOT NULL, playback_is_playing BOOLEAN NOT NULL,
		party_mode BOOLEAN NOT NULL, playback_observed_at TIMESTAMPTZ NOT NULL,
		last_seen_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL,
		controller_token_hash TEXT NOT NULL, pairing_expires_at TIMESTAMPTZ NOT NULL
	)`)
	if err != nil {
		t.Fatal(err)
	}

	client := Client{DB: db}
	err = client.prepareUpdateOwnedRemoteControlV2Stmt()
	if err != nil {
		t.Fatal(err)
	}
	defer client.UpdateOwnedRemoteControlV2Statement.Close()

	tests := []struct {
		name      string
		owner     string
		playing   bool
		position  int
		unchanged bool
		empty     bool
	}{
		{name: "paused duplicate", owner: "owner", position: 1000, unchanged: true},
		{name: "paused seek", owner: "owner", position: 2000},
		{name: "playing repeated position is a new time anchor", owner: "owner", playing: true, position: 1000},
		{name: "wrong owner", owner: "other", position: 1000, empty: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err = db.Exec(`DELETE FROM remote_controls`)
			if err != nil {
				t.Fatal(err)
			}
			_, err = db.Exec(`INSERT INTO remote_controls VALUES (
				'remote', 'owner', 'room', 'item', 1000, $1, FALSE,
				'2000-01-01', '2000-01-01', '2000-01-01', '', '2000-01-01'
			)`, tt.playing)
			if err != nil {
				t.Fatal(err)
			}

			remote, updateErr := client.UpdateOwnedRemoteControlV2(context.Background(), "remote", tt.owner, vibe.RemoteUpdateRequestV2{
				RoomID: "room", CurrentPlaylistItemID: "item",
				PlaybackPositionMs: tt.position, PlaybackIsPlaying: tt.playing,
			})
			var unchanged vibe.ErrRemoteUnchanged
			if errors.As(updateErr, &unchanged) != tt.unchanged {
				t.Fatalf("unexpected no-op result: %v", updateErr)
			}
			if updateErr != nil && !tt.unchanged {
				t.Fatal(updateErr)
			}
			if !tt.unchanged && remote.IsEmpty() != tt.empty {
				t.Fatalf("unexpected remote: %+v", remote)
			}

			var untouched bool
			err = db.QueryRow(`SELECT updated_at = '2000-01-01' AND playback_observed_at = '2000-01-01' AND last_seen_at = '2000-01-01' FROM remote_controls`).Scan(&untouched)
			if err != nil {
				t.Fatal(err)
			}
			if untouched != (tt.unchanged || tt.empty) {
				t.Fatalf("unexpected timestamp write: untouched=%v", untouched)
			}
		})
	}
}
