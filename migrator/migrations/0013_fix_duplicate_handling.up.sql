-- Fix duplicate song handling by improving the ON CONFLICT behavior
-- When a duplicate song is added, we should return the existing song ID instead of failing

-- The current ON CONFLICT clause does a no-op update which doesn't help
-- Instead, we'll handle this in the application logic by checking for existing songs first
-- This migration just ensures the constraint is properly indexed

-- Ensure the unique constraint index exists (it should from migration 0012)
CREATE UNIQUE INDEX IF NOT EXISTS idx_songs_unique_source ON songs(room_id, source_type, source_id);