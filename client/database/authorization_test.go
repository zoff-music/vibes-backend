package database

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

func TestPlaylistAuthorizationConsumption(t *testing.T) {
	url := os.Getenv("VIBES_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("VIBES_TEST_DATABASE_URL is required")
	}
	tests := []struct {
		name      string
		source    string
		lifetime  time.Duration
		wantToken string
	}{
		{"correct playlist", "source", time.Minute, "token"},
		{"other playlist", "other", time.Minute, ""},
		{"expired code", "source", -time.Minute, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := sql.Open("pgx", url)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { db.Close() })
			db.SetMaxOpenConns(1)
			ctx := t.Context()
			_, err = db.ExecContext(ctx, "CREATE TEMP TABLE playlist_authorizations (LIKE public.playlist_authorizations INCLUDING ALL)")
			if err != nil {
				t.Fatal(err)
			}
			c := Client{DB: db}
			err = c.prepareSavePlaylistAuthorizationStmt()
			if err != nil {
				t.Fatal(err)
			}
			err = c.prepareConsumePlaylistAuthorizationStmt()
			if err != nil {
				t.Fatal(err)
			}
			err = c.SavePlaylistAuthorization(ctx, "receipt", "source", "token", time.Now().Add(tt.lifetime))
			if err != nil {
				t.Fatal(err)
			}
			token, err := c.ConsumePlaylistAuthorization(ctx, "receipt", tt.source)
			if err != nil || token != tt.wantToken {
				t.Fatalf("unexpected token or error: %v", err)
			}
			token, err = c.ConsumePlaylistAuthorization(ctx, "receipt", "source")
			if err != nil || (token == "token") != (tt.source == "other") {
				t.Fatalf("unexpected replay or wrong-source consumption: %v", err)
			}
		})
	}
}
