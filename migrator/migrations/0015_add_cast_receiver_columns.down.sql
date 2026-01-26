CREATE TABLE room_users_revert (
    id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    is_admin INTEGER DEFAULT 0,
    is_active_listener INTEGER DEFAULT 0,
    joined_at DATETIME DEFAULT now,
    last_seen_at DATETIME DEFAULT now,
    PRIMARY KEY (id, room_id),
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

INSERT INTO room_users_revert (
    id,
    room_id,
    is_admin,
    is_active_listener,
    joined_at,
    last_seen_at
)
SELECT
    id,
    room_id,
    is_admin,
    is_active_listener,
    joined_at,
    last_seen_at
FROM room_users;

DROP TABLE room_users;
ALTER TABLE room_users_revert RENAME TO room_users;

CREATE INDEX IF NOT EXISTS idx_room_users_room_id ON room_users(room_id);
CREATE INDEX IF NOT EXISTS idx_room_users_last_seen ON room_users(last_seen_at);
