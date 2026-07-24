package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) prepareCreateGeneratedRoomStmt() error {
	stmt, err := c.DB.Prepare(`
		WITH created_room_q AS (
			INSERT INTO rooms (id, name, mode, host_id, admin_password_hash, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		),
		created_settings_q AS (
			INSERT INTO room_settings (
				room_id,
				skip_allowed,
				democratic_skip,
				skip_vote_threshold,
				max_continuous_adds,
				remove_on_play,
				loop_queue,
				allow_duplicates,
				enabled_sources,
				only_admin_add_songs
			)
			SELECT id, $7, $8, $9, $10, $11, $12, $13, $14, $15
			FROM created_room_q
		),
		input_songs_q AS (
			SELECT *
			FROM UNNEST(
				$16::text[],
				$17::text[],
				$18::text[],
				$19::text[],
				$20::text[],
				$21::integer[]
			) WITH ORDINALITY AS input_songs(
				song_id,
				source_id,
				title,
				artist,
				thumbnail_url,
				duration,
				song_order
			)
		),
		inserted_songs_q AS (
			INSERT INTO songs (
				id,
				room_id,
				source_type,
				source_id,
				title,
				artist,
				thumbnail_url,
				duration,
				added_by,
				added_at,
				duplicate_guard
			)
			SELECT
				a.song_id,
				b.id,
				'youtube',
				a.source_id,
				a.title,
				a.artist,
				a.thumbnail_url,
				a.duration,
				$22,
				NOW() + ((a.song_order - 1) * INTERVAL '1 millisecond'),
				TRUE
			FROM input_songs_q a
			CROSS JOIN created_room_q b
			RETURNING id
		),
		created_playback_q AS (
			INSERT INTO playback_state (
				room_id,
				current_song_id,
				is_playing,
				position_ms,
				updated_at
			)
			SELECT
				a.id,
				($16::text[])[1],
				TRUE,
				0,
				NOW()
			FROM created_room_q a
		)
		SELECT COUNT(*) FROM inserted_songs_q
	`)
	if err != nil {
		return fmt.Errorf("error preparing CreateGeneratedRoomStatement: %w", err)
	}

	c.CreateGeneratedRoomStatement = stmt
	return nil
}

func (c *Client) CreateGeneratedRoom(
	ctx context.Context,
	room *vibe.Room,
	playlist *vibe.GeneratedPlaylist,
) (*vibe.GeneratedRoom, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateGeneratedRoom")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	songIDs := []string{}
	sourceIDs := []string{}
	titles := []string{}
	artists := []string{}
	thumbnailURLs := []string{}
	durations := []int{}
	for _, track := range playlist.Tracks {
		if track.YouTubeVideoID == "" {
			continue
		}
		songIDs = append(songIDs, uuid.New().String())
		sourceIDs = append(sourceIDs, track.YouTubeVideoID)
		titles = append(titles, track.Title)
		artists = append(artists, track.Artist)
		thumbnailURLs = append(thumbnailURLs, track.ThumbnailURL)
		durations = append(durations, track.Duration)
	}
	if len(songIDs) == 0 {
		return nil, fmt.Errorf("error generated playlist has no verified youtube songs")
	}

	row := c.CreateGeneratedRoomStatement.QueryRowContext(
		cctx,
		room.ID,
		room.Name,
		room.Mode,
		room.HostID,
		room.AdminPasswordHash,
		room.CreatedAt,
		room.Settings.SkipAllowed,
		room.Settings.DemocraticSkip,
		room.Settings.SkipVoteThreshold,
		room.Settings.MaxContinuousAdds,
		room.Settings.RemoveOnPlay,
		room.Settings.LoopQueue,
		room.Settings.AllowDuplicates,
		strings.Join(room.Settings.EnabledSources, ","),
		room.Settings.OnlyAdminAddSongs,
		songIDs,
		sourceIDs,
		titles,
		artists,
		thumbnailURLs,
		durations,
		room.HostID,
	)

	var addedTrackCount int
	err := row.Scan(&addedTrackCount)
	if err != nil {
		return nil, fmt.Errorf("error creating generated room: %w", err)
	}

	return &vibe.GeneratedRoom{
		Room:            room,
		Tracks:          playlist.Tracks,
		AddedTrackCount: addedTrackCount,
	}, nil
}
