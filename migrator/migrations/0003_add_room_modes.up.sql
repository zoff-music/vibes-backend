ALTER TABLE rooms ADD COLUMN mode TEXT NOT NULL DEFAULT 'server';
ALTER TABLE rooms ADD COLUMN host_id TEXT DEFAULT NULL;

CREATE TABLE room_participants (
    room_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    last_seen_at DATETIME NOT NULL,
    PRIMARY KEY (room_id, user_id),
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);
