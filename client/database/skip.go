package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

// prepareGetSkipVotesStmt prepares the GetSkipVotesStatement.
func (c *Client) prepareGetSkipVotesStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT song_id, user_id
		FROM skip_votes
		WHERE room_id = $1 AND song_id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetSkipVotesStatement: %w", err)
	}

	c.GetSkipVotesStatement = stmt

	return nil
}

// GetSkipVotes fetches all skip votes for a song.
func (c *Client) GetSkipVotes(ctx context.Context, roomID, songID string) ([]vibe.SkipVote, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetSkipVotes")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.GetSkipVotesStatement.QueryContext(cctx, roomID, songID)
	if err != nil {
		return nil, fmt.Errorf("error fetching skip votes: %w", err)
	}
	defer rows.Close()

	var votes []vibe.SkipVote
	for rows.Next() {
		var row skipVoteRow
		err := row.scanRows(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning skip vote row: %w", err)
		}

		votes = append(votes, row.toSkipVote())
	}

	return votes, nil
}

type skipVoteRow struct {
	SongID sql.NullString
	UserID sql.NullString
}

func (r *skipVoteRow) scanRows(rows *sql.Rows) error {
	return rows.Scan(
		&r.SongID,
		&r.UserID,
	)
}

func (r *skipVoteRow) toSkipVote() vibe.SkipVote {
	return vibe.SkipVote{
		SongID: r.SongID.String,
		UserID: r.UserID.String,
	}
}

// prepareHasUserVotedStmt prepares the HasUserVotedStatement.
func (c *Client) prepareHasUserVotedStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT COUNT(*) FROM skip_votes
		WHERE room_id = $1 AND song_id = $2 AND user_id = $3
	`)
	if err != nil {
		return fmt.Errorf("error preparing HasUserVotedStatement: %w", err)
	}

	c.HasUserVotedStatement = stmt

	return nil
}

// HasUserVoted checks if a user has already voted to skip a song.
func (c *Client) HasUserVoted(ctx context.Context, roomID, songID, userID string) (bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "HasUserVoted")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r := c.HasUserVotedStatement.QueryRowContext(cctx, roomID, songID, userID)

	var row skipVoteCountRow
	err := row.scan(r)
	if err != nil {
		return false, fmt.Errorf("error checking user vote: %w", err)
	}

	return row.toHasUserVoted(), nil
}

type skipVoteCountRow struct {
	Count sql.NullInt64
}

func (r *skipVoteCountRow) scan(row *sql.Row) error {
	return row.Scan(&r.Count)
}

func (r *skipVoteCountRow) toHasUserVoted() bool {
	return int(r.Count.Int64) > 0
}

// prepareAddSkipVoteStmt prepares the AddSkipVoteStatement.
func (c *Client) prepareAddSkipVoteStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT INTO skip_votes (room_id, song_id, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT(room_id, song_id, user_id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("error preparing AddSkipVoteStatement: %w", err)
	}

	c.AddSkipVoteStatement = stmt

	return nil
}

// AddSkipVote adds a skip vote for a song.
func (c *Client) AddSkipVote(ctx context.Context, roomID, songID, userID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "AddSkipVote")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.AddSkipVoteStatement.ExecContext(cctx, roomID, songID, userID)
	if err != nil {
		return fmt.Errorf("error adding skip vote: %w", err)
	}

	return nil
}

// prepareClearSkipVotesStmt prepares the ClearSkipVotesStatement.
func (c *Client) prepareClearSkipVotesStmt() error {
	stmt, err := c.DB.Prepare(`
		DELETE FROM skip_votes WHERE room_id = $1 AND song_id = $2
	`)
	if err != nil {
		return fmt.Errorf("error preparing ClearSkipVotesStatement: %w", err)
	}

	c.ClearSkipVotesStatement = stmt

	return nil
}

// clearSkipVotes clears all skip votes for a song.
func (c *Client) clearSkipVotes(ctx context.Context, roomID, songID string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "clearSkipVotes")
	defer span.End()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.ClearSkipVotesStatement.ExecContext(cctx, roomID, songID)
	if err != nil {
		return fmt.Errorf("error clearing skip votes: %w", err)
	}

	return nil
}

// SkipSong skips the current track to the next one in the queue, either immediately (if host/admin/forced) or by voting.
func (c *Client) SkipSong(ctx context.Context, roomID, userID string, isAdmin bool) (*vibe.SkipSongResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SkipSong")
	defer span.End()

	room, err := c.GetRoom(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("error fetching room in SkipSong: %w", err)
	}

	// Determine if user is host
	isHost := room.HostID == userID
	log.Printf("room.HostID: %s, userID: %s, isHost: %v", room.HostID, userID, isHost)

	// 1. Check if skipping is allowed at all
	if !room.Settings.SkipAllowed && !isHost && !isAdmin {
		return nil, internalerror.ErrSkipDisabled{
			Err: fmt.Errorf("skipping is disabled in this room"),
		}
	}

	// 4. Check host mode restrictions
	if room.Mode == vibe.RoomModeHost && !isHost && !isAdmin {
		return nil, internalerror.ErrHostModeSkipOnly{
			Err: fmt.Errorf("only hosts can skip in host mode"),
		}
	}

	// 5. Determine if this should be a forced skip
	shouldForce := isHost || isAdmin
	if !room.Settings.DemocraticSkip {
		shouldForce = true
	}

	// 6. Execute Force Skip
	if shouldForce {
		log.Printf("[DEBUG-SKIP] Room: %s, User: %s, Force Skip (Host: %v, Admin: %v, User: %s, HostID: %s)\n", roomID, userID, isHost, isAdmin, userID, room.HostID)

		state, err := c.skipTrack(ctx, roomID)
		if err != nil {
			return nil, fmt.Errorf("error skipping track in shouldForce: %w", err)
		}

		return &vibe.SkipSongResult{
			Action:   vibe.RoomActionSkip,
			Skipped:  true,
			NextSong: state.CurrentSong,
			Playback: state,
		}, nil
	}

	// 7. Execute Vote Skip
	log.Printf("[DEBUG-SKIP] Room: %s, User: %s, Voting to Skip\n", roomID, userID)

	state, err := c.GetPlaybackState(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error skipping song: get playback state: %w", err)
	}

	log.Printf("[DEBUG-SKIP] Current-playing %+v", state.CurrentSong)

	if state.CurrentSong == nil {
		return &vibe.SkipSongResult{
			Action:   vibe.RoomActionSkip,
			Playback: state,
		}, nil
	}
	songID := state.CurrentSong.ID

	voted, err := c.HasUserVoted(ctx, roomID, songID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking if user voted: %w", err)
	}

	if voted {
		log.Printf("[DEBUG-SKIP] User has voted %+v", voted)
		votes, err := c.GetSkipVotes(ctx, roomID, songID)
		if err != nil {
			return nil, fmt.Errorf("error fetching skip votes for already voted result: %w", err)
		}

		requiredVotes, err := c.getRequiredSkipVotes(ctx, roomID, room.Settings.SkipVoteThreshold)
		if err != nil {
			return nil, fmt.Errorf("error getting required skip votes for already voted result: %w", err)
		}

		return &vibe.SkipSongResult{
			Action:        vibe.RoomActionSkip,
			AlreadyVoted:  true,
			CurrentVotes:  len(votes),
			RequiredVotes: requiredVotes,
			Playback:      state,
		}, nil
	}

	err = c.AddSkipVote(ctx, roomID, songID, userID)
	if err != nil {
		return nil, fmt.Errorf("error adding skip vote: %w", err)
	}

	votes, err := c.GetSkipVotes(ctx, roomID, songID)
	if err != nil {
		return nil, fmt.Errorf("error fetching skip votes count: %w", err)
	}

	voteCount := len(votes)
	requiredVotes, err := c.getRequiredSkipVotes(ctx, roomID, room.Settings.SkipVoteThreshold)
	if err != nil {
		return nil, fmt.Errorf("error getting required skip votes: %w", err)
	}

	if voteCount < requiredVotes {
		return &vibe.SkipSongResult{
			Action:        vibe.RoomActionSkip,
			Voted:         true,
			CurrentVotes:  voteCount,
			RequiredVotes: requiredVotes,
			Playback:      state,
		}, nil
	}

	// Threshold met, skip the track
	log.Printf("[DEBUG-SKIP] Room: %s, Threshold met. Skipping.\n", roomID)

	err = c.clearSkipVotes(ctx, roomID, songID)
	if err != nil {
		return nil, fmt.Errorf("error clearing skip votes: %w", err)
	}

	err = c.clearVotesSong(ctx, roomID, songID)
	if err != nil {
		return nil, fmt.Errorf("error clearing votes for song: %w", err)
	}

	newState, err := c.skipTrack(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error skipping track after vote: %w", err)
	}

	return &vibe.SkipSongResult{
		Action:        vibe.RoomActionSkip,
		Skipped:       true,
		Voted:         true,
		CurrentVotes:  voteCount,
		RequiredVotes: requiredVotes,
		NextSong:      newState.CurrentSong,
		Playback:      newState,
	}, nil
}

func (c *Client) getRequiredSkipVotes(ctx context.Context, roomID string, threshold float64) (int, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "getRequiredSkipVotes")
	defer span.End()

	counts, err := c.GetActiveListenerCounts(ctx, roomID, 5*time.Second)
	if err != nil {
		return 0, fmt.Errorf("error counting active participants: %w", err)
	}

	participantCount := counts.ActiveListeners
	if participantCount == 0 && counts.ActiveCastReceivers > 0 {
		participantCount = 1
	}

	requiredVotes := int(float64(participantCount) * threshold)
	log.Printf("[DEBUG-SKIP] Room: %s, Participants: %d, Threshold: %f, Required: %d\n", roomID, participantCount, threshold, requiredVotes)

	if participantCount == 1 {
		return 1, nil
	}
	if requiredVotes < 2 {
		return 2, nil
	}

	return requiredVotes, nil
}
