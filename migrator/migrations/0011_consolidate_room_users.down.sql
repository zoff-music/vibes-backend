-- Down migration to revert consolidation

-- 1. Drop new table
DROP TABLE IF EXISTS room_users;

-- 2. Restore old room_users table template (ID only as PK)
CREATE TABLE room_users (
	id TEXT PRIMARY KEY DEFAULT (LOWER(HEX(RANDOMBLOB(4))) || '-' || LOWER(HEX(RANDOMBLOB(2))) || '-4' || SUBSTR(LOWER(HEX(RANDOMBLOB(2))),2) || '-' || SUBSTR('89ab',ABS(RANDOM()) % 4 + 1, 1) || SUBSTR(LOWER(HEX(RANDOMBLOB(2))),2) || '-' || LOWER(HEX(RANDOMBLOB(6)))),
	room_id TEXT NOT NULL,
	nickname TEXT,
	is_admin INTEGER DEFAULT 0,
	joined_at DATETIME DEFAULT now,
	last_seen_at DATETIME DEFAULT now,
	FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

-- 3. Restore old room_participants table
CREATE TABLE room_participants (
    room_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    last_seen_at DATETIME NOT NULL,
    is_active_listener INTEGER DEFAULT 0,
    PRIMARY KEY (room_id, user_id),
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_room_users_room_id ON room_users(room_id);
CREATE INDEX IF NOT EXISTS idx_room_users_last_seen ON room_users(last_seen_at);
