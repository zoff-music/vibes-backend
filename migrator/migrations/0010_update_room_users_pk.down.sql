-- Migration to revert room_users to use single primary key (id)
-- Note: This might cause issues if multiple rows have same ID but different room_id

-- 1. Create new table with old schema
CREATE TABLE IF NOT EXISTS room_users_old (
	id TEXT PRIMARY KEY DEFAULT (LOWER(HEX(RANDOMBLOB(4))) || '-' || LOWER(HEX(RANDOMBLOB(2))) || '-4' || SUBSTR(LOWER(HEX(RANDOMBLOB(2))),2) || '-' || SUBSTR('89ab',ABS(RANDOM()) % 4 + 1, 1) || SUBSTR(LOWER(HEX(RANDOMBLOB(2))),2) || '-' || LOWER(HEX(RANDOMBLOB(6)))),
	room_id TEXT NOT NULL,
	nickname TEXT,
	is_admin INTEGER DEFAULT 0,
	joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

-- 2. Copy data (might lose some if duplicate IDs exist across rooms)
INSERT INTO room_users_old (id, room_id, nickname, is_admin, joined_at, last_seen_at)
SELECT id, room_id, nickname, is_admin, joined_at, last_seen_at FROM room_users
GROUP BY id;

-- 3. Drop current table and rename old one
DROP TABLE room_users;
ALTER TABLE room_users_old RENAME TO room_users;

-- 4. Re-create indexes
CREATE INDEX IF NOT EXISTS idx_room_users_room_id ON room_users(room_id);
CREATE INDEX IF NOT EXISTS idx_room_users_last_seen ON room_users(last_seen_at);
