package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zoff-music/vibes/monitoring/opentracing"
	"github.com/zoff-music/vibes/vibe"
)

// prepareGetSkipVotesStmt prepares the GetSkipVotesStatement.
func (c *Client) prepareGetSkipVotesStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT song_id, user_id
		FROM skip_votes
		WHERE room_id = ?1 AND song_id = ?2
	`)
	if err != nil {
		return fmt.Errorf("error preparing GetSkipVotesStatement: %w", err)
	}

	c.GetSkipVotesStatement = stmt

	return nil
}

// GetSkipVotes fetches all skip votes for a song.
func (c *Client) GetSkipVotes(ctx context.Context, roomID, songID string) ([]vibe.SkipVote, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetSkipVotes")
	defer span.Finish()

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

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating skip vote rows: %w", err)
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
		WHERE room_id = ?1 AND song_id = ?2 AND user_id = ?3
	`)
	if err != nil {
		return fmt.Errorf("error preparing HasUserVotedStatement: %w", err)
	}

	c.HasUserVotedStatement = stmt

	return nil
}

// HasUserVoted checks if a user has already voted to skip a song.
func (c *Client) HasUserVoted(ctx context.Context, roomID, songID, userID string) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "HasUserVoted")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := c.HasUserVotedStatement.QueryRowContext(cctx, roomID, songID, userID)

	var scanned skipVoteCountRow

	err := scanned.scan(row)
	if err != nil {
		return false, fmt.Errorf("error checking user vote: %w", err)
	}

	return scanned.toHasUserVoted(), nil
}

type skipVoteCountRow struct {
	Count sql.NullInt64
}

func (r *skipVoteCountRow) scan(row *sql.Row) error {
	return row.Scan(&r.Count)
}

func (r *skipVoteCountRow) toHasUserVoted() bool {
	if !r.Count.Valid {
		return false
	}

	return r.Count.Int64 > 0
}

// prepareAddSkipVoteStmt prepares the AddSkipVoteStatement.
func (c *Client) prepareAddSkipVoteStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT OR IGNORE INTO skip_votes (room_id, song_id, user_id)
		VALUES (?1, ?2, ?3)
	`)
	if err != nil {
		return fmt.Errorf("error preparing AddSkipVoteStatement: %w", err)
	}

	c.AddSkipVoteStatement = stmt

	return nil
}

// AddSkipVote adds a skip vote for a song.
func (c *Client) AddSkipVote(ctx context.Context, roomID, songID, userID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "AddSkipVote")
	defer span.Finish()

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
		DELETE FROM skip_votes WHERE room_id = ?1 AND song_id = ?2
	`)
	if err != nil {
		return fmt.Errorf("error preparing ClearSkipVotesStatement: %w", err)
	}

	c.ClearSkipVotesStatement = stmt

	return nil
}

// ClearSkipVotes clears all skip votes for a song.
func (c *Client) ClearSkipVotes(ctx context.Context, roomID, songID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ClearSkipVotes")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.ClearSkipVotesStatement.ExecContext(cctx, roomID, songID)
	if err != nil {
		return fmt.Errorf("error clearing skip votes: %w", err)
	}

	return nil
}

// VoteToSkip adds a user's vote to skip the current track and skips if threshold is met.
func (c *Client) VoteToSkip(ctx context.Context, roomID, userID string) (*vibe.PlaybackState, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "VoteToSkip")
	defer span.Finish()

	state, err := c.GetPlaybackState(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("vote to skip: get playback state: %w", err)
	}

	if state.CurrentSongID == nil {
		return state, nil
	}
	songID := *state.CurrentSongID

	voted, err := c.HasUserVoted(ctx, roomID, songID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking if user voted: %w", err)
	}

	if voted {
		return state, nil // Already voted
	}

	err = c.AddSkipVote(ctx, roomID, songID, userID)
	if err != nil {
		return nil, fmt.Errorf("error adding skip vote: %w", err)
	}

	// Check threshold
	room, err := c.GetRoom(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("error fetching room for threshold: %w", err)
	}

	votes, err := c.GetSkipVotes(ctx, roomID, songID)
	if err != nil {
		return nil, fmt.Errorf("error fetching skip votes count: %w", err)
	}

	// Count active participants for threshold (anyone connected to SSE)
	activeParticipants, err := c.GetActiveParticipants(ctx, roomID, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error counting active participants: %w", err)
	}
	participantCount := len(activeParticipants)

	voteCount := len(votes)
	requiredVotes := int(float64(participantCount) * room.Settings.SkipVoteThreshold)

	// If there is only 1 participant, allow them to skip alone
	if participantCount == 1 {
		requiredVotes = 1
	} else if requiredVotes < 2 {
		// Otherwise, enforce minimum 2 votes to prevent single-vote skips in groups
		requiredVotes = 2
	}

	if voteCount >= requiredVotes {
		// Threshold met, skip the track
		newState, err := c.skipTrack(ctx, roomID)
		if err != nil {
			return nil, fmt.Errorf("error skipping track after vote: %w", err)
		}

		err = c.ClearSkipVotes(ctx, roomID, songID)
		if err != nil {
			// Log error but don't fail operation as skip already happened
		}

		return newState, nil
	}

	return state, nil
}
