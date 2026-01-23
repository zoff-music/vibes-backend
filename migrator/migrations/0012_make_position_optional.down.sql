-- Rollback: Make position field required again and remove unique constraint

-- Create new songs table with NOT NULL constraint on position and without unique constraint
CREATE TABLE songs_new (
	id TEXT PRIMARY KEY DEFAULT (LOWER(HEX(RANDOMBLOB(4))) || '-' || LOWER(HEX(RANDOMBLOB(2))) || '-4' || SUBSTR(LOWER(HEX(RANDOMBLOB(2))),2) || '-' || SUBSTR('89ab',ABS(RANDOM()) % 4 + 1, 1) || SUBSTR(LOWER(HEX(RANDOMBLOB(2))),2) || '-' || LOWER(HEX(RANDOMBLOB(6)))),
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
	position INTEGER NOT NULL, -- Restored NOT NULL constraint
	FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
	-- Removed unique constraint
);

-- Copy data from old table to new table (only rows with non-null position)
INSERT INTO songs_new (id, room_id, source_type, source_id, title, artist, thumbnail_url, duration, added_by, added_by_nickname, added_at, position)
SELECT id, room_id, source_type, source_id, title, artist, thumbnail_url, duration, added_by, added_by_nickname, added_at, position
FROM songs
WHERE position IS NOT NULL;

-- Drop old table
DROP TABLE songs;

-- Rename new table
ALTER TABLE songs_new RENAME TO songs;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_songs_room_id ON songs(room_id);
CREATE INDEX IF NOT EXISTS idx_songs_room_position ON songs(room_id, position);
CREATE UNIQUE INDEX IF NOT EXISTS idx_songs_unique_source ON songs(room_id, source_type, source_id);