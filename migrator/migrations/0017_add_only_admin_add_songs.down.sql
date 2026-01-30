-- Update rooms_view to remove only_admin_add_songs
DROP VIEW IF EXISTS rooms_view;
CREATE VIEW rooms_view AS
SELECT
    r.id,
    r.name,
    r.mode,
    r.host_id,
    r.admin_password_hash,
    r.created_at,
    rs.skip_allowed,
    rs.democratic_skip,
    rs.skip_vote_threshold,
    rs.max_continuous_adds,
    rs.remove_on_play,
    rs.loop_queue,
    rs.allow_duplicates,
    rs.enabled_sources
FROM rooms r
JOIN room_settings rs ON r.id = rs.room_id;

-- Revert rooms_view_insert trigger
DROP TRIGGER IF EXISTS rooms_view_insert;
CREATE TRIGGER rooms_view_insert
INSTEAD OF INSERT ON rooms_view
BEGIN
    INSERT INTO rooms (id, name, mode, host_id, admin_password_hash, created_at)
    VALUES (NEW.id, NEW.name, NEW.mode, NEW.host_id, NEW.admin_password_hash, NEW.created_at);

    INSERT INTO room_settings (
        room_id,
        skip_allowed,
        democratic_skip,
        skip_vote_threshold,
        max_continuous_adds,
        remove_on_play,
        loop_queue,
        allow_duplicates,
        enabled_sources
    )
    VALUES (
        NEW.id,
        NEW.skip_allowed,
        NEW.democratic_skip,
        NEW.skip_vote_threshold,
        NEW.max_continuous_adds,
        NEW.remove_on_play,
        NEW.loop_queue,
        NEW.allow_duplicates,
        COALESCE(NEW.enabled_sources, 'youtube,spotify,soundcloud')
    );
END;

-- Revert rooms_view_update trigger
DROP TRIGGER IF EXISTS rooms_view_update;
CREATE TRIGGER rooms_view_update
INSTEAD OF UPDATE ON rooms_view
BEGIN
    UPDATE rooms
    SET name = NEW.name,
        mode = NEW.mode,
        host_id = NEW.host_id,
        admin_password_hash = NEW.admin_password_hash
    WHERE id = OLD.id;

    UPDATE room_settings
    SET skip_allowed = NEW.skip_allowed,
        democratic_skip = NEW.democratic_skip,
        skip_vote_threshold = NEW.skip_vote_threshold,
        max_continuous_adds = NEW.max_continuous_adds,
        remove_on_play = NEW.remove_on_play,
        loop_queue = NEW.loop_queue,
        allow_duplicates = NEW.allow_duplicates,
        enabled_sources = NEW.enabled_sources
    WHERE room_id = OLD.id;
END;

-- Remove only_admin_add_songs column
ALTER TABLE room_settings DROP COLUMN only_admin_add_songs;
