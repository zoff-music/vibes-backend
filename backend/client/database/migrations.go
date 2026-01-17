package database

import (
	"context"
	"fmt"
	"time"

	"github.com/zoff-music/vibes/monitoring/opentracing"
)

const schema = `
-- Rooms table
CREATE TABLE IF NOT EXISTS rooms (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	admin_password_hash TEXT,
	settings_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Songs table (queue items)
CREATE TABLE IF NOT EXISTS songs (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	source_type TEXT NOT NULL,
	source_id TEXT NOT NULL,
	title TEXT NOT NULL,
	artist TEXT,
	thumbnail_url TEXT NOT NULL,
	duration INTEGER NOT NULL,
	added_by TEXT NOT NULL,
	added_by_nickname TEXT,
	added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	position INTEGER NOT NULL,
	FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_songs_room_id ON songs(room_id);
CREATE INDEX IF NOT EXISTS idx_songs_room_position ON songs(room_id, position);

-- Playback state table
CREATE TABLE IF NOT EXISTS playback_state (
	room_id TEXT PRIMARY KEY,
	current_song_id TEXT,
	is_playing INTEGER DEFAULT 0,
	position_ms INTEGER DEFAULT 0,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

-- Room users table (session tracking)
CREATE TABLE IF NOT EXISTS room_users (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	nickname TEXT,
	is_admin INTEGER DEFAULT 0,
	joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_room_users_room_id ON room_users(room_id);
CREATE INDEX IF NOT EXISTS idx_room_users_last_seen ON room_users(last_seen_at);

-- Skip votes table
CREATE TABLE IF NOT EXISTS skip_votes (
	room_id TEXT NOT NULL,
	song_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (room_id, song_id, user_id),
	FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
	FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_skip_votes_song ON skip_votes(room_id, song_id);
`

func (c *Client) migrateSchema(ctx context.Context) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "migrateSchema")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := c.DB.ExecContext(cctx, schema)
	if err != nil {
		return fmt.Errorf("error executing schema migration: %w", err)
	}

	return nil
}
