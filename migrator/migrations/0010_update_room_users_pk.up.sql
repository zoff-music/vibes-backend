-- Migration to update room_users to use composite primary key (id, room_id)
-- This allows same session ID across different rooms.

-- 1. Create new table with new schema
CREATE TABLE IF NOT EXISTS room_users_new (
	id TEXT NOT NULL,
	room_id TEXT NOT NULL,
	nickname TEXT,
	is_admin INTEGER DEFAULT 0,
	joined_at DATETIME DEFAULT now,
	last_seen_at DATETIME DEFAULT now,
	PRIMARY KEY (id, room_id),
	FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

-- 2. Copy data
INSERT INTO room_users_new (id, room_id, nickname, is_admin, joined_at, last_seen_at)
SELECT id, room_id, nickname, is_admin, joined_at, last_seen_at FROM room_users;

-- 3. Drop old table and rename new one
DROP TABLE room_users;
ALTER TABLE room_users_new RENAME TO room_users;

-- 4. Re-create indexes
CREATE INDEX IF NOT EXISTS idx_room_users_room_id ON room_users(room_id);
CREATE INDEX IF NOT EXISTS idx_room_users_last_seen ON room_users(last_seen_at);
