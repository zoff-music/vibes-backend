-- Remove the unique constraint on songs table to allow duplicates when room setting allows it
-- The application will handle duplicate prevention using INSERT OR IGNORE when needed

-- Create new songs table without the unique constraint
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
	added_at DATETIME DEFAULT now,
	position INTEGER,
	FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
	-- Removed UNIQUE(room_id, source_type, source_id) constraint
);

-- Copy data from old table to new table
INSERT INTO songs_new (id, room_id, source_type, source_id, title, artist, thumbnail_url, duration, added_by, added_by_nickname, added_at, position)
SELECT id, room_id, source_type, source_id, title, artist, thumbnail_url, duration, added_by, added_by_nickname, added_at, position
FROM songs;

-- Drop old table
DROP TABLE songs;

-- Rename new table
ALTER TABLE songs_new RENAME TO songs;

-- Recreate indexes (without the unique constraint index)
CREATE INDEX IF NOT EXISTS idx_songs_room_id ON songs(room_id);
CREATE INDEX IF NOT EXISTS idx_songs_room_position ON songs(room_id, position);
-- Removed: CREATE UNIQUE INDEX IF NOT EXISTS idx_songs_unique_source ON songs(room_id, source_type, source_id);