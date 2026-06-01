CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS rooms (
	id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	name TEXT NOT NULL,
	mode TEXT NOT NULL DEFAULT 'server',
	host_id TEXT,
	admin_password_hash TEXT,
	created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS room_settings (
	room_id TEXT PRIMARY KEY REFERENCES rooms(id) ON DELETE CASCADE,
	skip_allowed INTEGER NOT NULL DEFAULT 1,
	democratic_skip INTEGER NOT NULL DEFAULT 1,
	skip_vote_threshold REAL NOT NULL DEFAULT 0.5,
	max_continuous_adds INTEGER NOT NULL DEFAULT 3,
	remove_on_play INTEGER NOT NULL DEFAULT 1,
	loop_queue INTEGER NOT NULL DEFAULT 0,
	allow_duplicates INTEGER NOT NULL DEFAULT 0,
	enabled_sources TEXT NOT NULL DEFAULT 'youtube,spotify,soundcloud',
	only_admin_add_songs INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS songs (
	id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	room_id TEXT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
	source_type TEXT NOT NULL,
	source_id TEXT NOT NULL,
	title TEXT NOT NULL,
	artist TEXT,
	thumbnail_url TEXT NOT NULL,
	duration INTEGER NOT NULL,
	added_by TEXT NOT NULL,
	added_by_nickname TEXT,
	added_at TIMESTAMPTZ DEFAULT NOW(),
	position INTEGER
);

CREATE INDEX IF NOT EXISTS idx_songs_room_id ON songs(room_id);
CREATE INDEX IF NOT EXISTS idx_songs_room_position ON songs(room_id, position);

CREATE TABLE IF NOT EXISTS playback_state (
	room_id TEXT PRIMARY KEY REFERENCES rooms(id) ON DELETE CASCADE,
	current_song_id TEXT,
	is_playing INTEGER DEFAULT 0,
	position_ms INTEGER DEFAULT 0,
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS room_users (
	id TEXT NOT NULL,
	room_id TEXT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
	nickname TEXT,
	is_admin INTEGER DEFAULT 0,
	is_active_listener INTEGER DEFAULT 0,
	is_cast_receiver INTEGER DEFAULT 0,
	cast_owner_id TEXT,
	joined_at TIMESTAMPTZ DEFAULT NOW(),
	last_seen_at TIMESTAMPTZ DEFAULT NOW(),
	PRIMARY KEY (id, room_id)
);

CREATE INDEX IF NOT EXISTS idx_room_users_room_id ON room_users(room_id);
CREATE INDEX IF NOT EXISTS idx_room_users_last_seen ON room_users(last_seen_at);

CREATE TABLE IF NOT EXISTS skip_votes (
	room_id TEXT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
	song_id TEXT NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
	user_id TEXT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	PRIMARY KEY (room_id, song_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_skip_votes_song ON skip_votes(room_id, song_id);

CREATE TABLE IF NOT EXISTS song_votes (
	room_id TEXT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
	song_id TEXT NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
	user_id TEXT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	PRIMARY KEY (room_id, song_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_song_votes_song ON song_votes(room_id, song_id);

CREATE TABLE IF NOT EXISTS auth_tokens (
	user_id TEXT NOT NULL,
	provider TEXT NOT NULL,
	code TEXT NOT NULL,
	state TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	PRIMARY KEY (user_id, provider)
);

CREATE TABLE IF NOT EXISTS access_tokens (
	user_id TEXT NOT NULL,
	provider TEXT NOT NULL,
	access_token TEXT NOT NULL,
	refresh_token TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL,
	refresh_expires_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	last_checked TIMESTAMPTZ DEFAULT NULL,
	PRIMARY KEY (user_id, provider)
);

CREATE TABLE IF NOT EXISTS pending_oauth_state (
	user_id TEXT NOT NULL,
	state TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL,
	code_verifier TEXT,
	PRIMARY KEY (user_id, state)
);
