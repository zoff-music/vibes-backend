package vibe

// SkipVote tracks skip votes for a playlist item.
type SkipVote struct {
	PlaylistItemID string
	UserID         string
}

// SkipSongResult describes the result of a skip request.
//
// Deprecated: Use SkipPlaylistItemResult for new code. Retained for legacy API compatibility.
type SkipSongResult struct {
	Action         string         `json:"action"`
	Skipped        bool           `json:"skipped"`
	Voted          bool           `json:"voted"`
	AlreadyVoted   bool           `json:"alreadyVoted"`
	CurrentVotes   int            `json:"currentVotes"`
	RequiredVotes  int            `json:"requiredVotes"`
	NextSong       *Song          `json:"nextSong"`
	Playback       *PlaybackState `json:"playback"`
	PreviousSongID string         `json:"-"`
}

// SkipVoteUpdate describes a skip vote event.
//
// Deprecated: Use SkipVoteUpdateV2 for new code. Retained for legacy API compatibility.
type SkipVoteUpdate struct {
	UserID        string `json:"userId"`
	SongID        string `json:"songId"`
	CurrentVotes  int    `json:"currentVotes"`
	RequiredVotes int    `json:"requiredVotes"`
}
