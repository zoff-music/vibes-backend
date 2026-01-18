-- Clean up duplicates before adding unique index
DELETE FROM songs
WHERE id NOT IN (
    SELECT id
    FROM (
        SELECT id, MIN(added_at)
        FROM songs
        GROUP BY room_id, source_type, source_id
    )
);

-- Add unique index
CREATE UNIQUE INDEX idx_songs_unique_source ON songs(room_id, source_type, source_id);

-- Create votes table
CREATE TABLE song_votes (
    room_id TEXT NOT NULL,
    song_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (room_id, song_id, user_id),
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
);

CREATE INDEX idx_song_votes_song ON song_votes(room_id, song_id);
