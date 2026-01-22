-- Migration to consolidate room_participants and room_users
-- SQLite < 3.35.0 doesn't support ON CONFLICT on INSERT INTO ... SELECT
-- We use a more compatible GROUP BY approach over a UNION ALL.

-- 1. Ensure a clean state for the consolidation table
DROP TABLE IF EXISTS room_users_consolidated;

-- 2. Create the consolidated room_users table
CREATE TABLE room_users_consolidated (
    id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    is_admin INTEGER DEFAULT 0,
    is_active_listener INTEGER DEFAULT 0,
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, room_id),
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

-- 3. Consolidate data from both tables using a UNION ALL approach
-- This is compatible with older SQLite versions as it doesn't use UPSERT.
INSERT INTO room_users_consolidated (id, room_id, is_admin, joined_at, last_seen_at, is_active_listener)
SELECT 
    id, 
    room_id, 
    MAX(is_admin), 
    MIN(joined_at), 
    MAX(last_seen_at), 
    MAX(is_active_listener)
FROM (
    -- Migrate existing users
    SELECT id, room_id, is_admin, joined_at, last_seen_at, 0 as is_active_listener FROM room_users
    UNION ALL
    -- Migrate existing participants
    SELECT user_id as id, room_id, 0 as is_admin, last_seen_at as joined_at, last_seen_at, is_active_listener FROM room_participants
)
GROUP BY id, room_id;

-- 4. Finalize: drop old tables and swap the consolidated one in
DROP TABLE IF EXISTS room_participants;
DROP TABLE IF EXISTS room_users;
ALTER TABLE room_users_consolidated RENAME TO room_users;

-- 5. Restore indexes
CREATE INDEX IF NOT EXISTS idx_room_users_room_id ON room_users(room_id);
CREATE INDEX IF NOT EXISTS idx_room_users_last_seen ON room_users(last_seen_at);
