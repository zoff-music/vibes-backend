package database

import (
	"context"
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
		WHERE room_id = ? AND song_id = ?
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
		var vote vibe.SkipVote

		err := rows.Scan(&vote.SongID, &vote.UserID)
		if err != nil {
			return nil, fmt.Errorf("error scanning skip vote row: %w", err)
		}

		votes = append(votes, vote)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating skip vote rows: %w", err)
	}

	return votes, nil
}

// prepareHasUserVotedStmt prepares the HasUserVotedStatement.
func (c *Client) prepareHasUserVotedStmt() error {
	stmt, err := c.DB.Prepare(`
		SELECT COUNT(*) FROM skip_votes
		WHERE room_id = ? AND song_id = ? AND user_id = ?
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

	var count int

	err := c.HasUserVotedStatement.QueryRowContext(cctx, roomID, songID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking user vote: %w", err)
	}

	return count > 0, nil
}

// prepareAddSkipVoteStmt prepares the AddSkipVoteStatement.
func (c *Client) prepareAddSkipVoteStmt() error {
	stmt, err := c.DB.Prepare(`
		INSERT OR IGNORE INTO skip_votes (room_id, song_id, user_id)
		VALUES (?, ?, ?)
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
		DELETE FROM skip_votes WHERE room_id = ? AND song_id = ?
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
