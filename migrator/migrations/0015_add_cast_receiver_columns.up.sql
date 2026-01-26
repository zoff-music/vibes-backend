ALTER TABLE room_users ADD COLUMN is_cast_receiver INTEGER DEFAULT 0;
ALTER TABLE room_users ADD COLUMN cast_owner_id TEXT;
