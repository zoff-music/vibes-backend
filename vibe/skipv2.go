package vibe

import "context"

// RequiresSkipVote applies after skip permission has been checked.
func (r *RoomV2) RequiresSkipVote(userID string) bool {
	if r.Mode == RoomModeHost && r.HostID == userID {
		return false
	}

	return r.Settings.DemocraticSkip && (r.Settings.SkipAllowed || !r.IsAdmin)
}

type SkipPlaylistItemRequest struct {
	PlaylistItemID string `json:"playlistItemId,omitempty"`
	ErrorCode      string `json:"errorCode,omitempty"`
}

type SkipPlaylistItemResult struct {
	Action                 string           `json:"action"`
	Skipped                bool             `json:"skipped"`
	Voted                  bool             `json:"voted"`
	AlreadyVoted           bool             `json:"alreadyVoted"`
	CurrentVotes           int              `json:"currentVotes"`
	RequiredVotes          int              `json:"requiredVotes"`
	NextPlaylistItem       *PlaylistItem    `json:"nextPlaylistItem"`
	Playback               *PlaybackStateV2 `json:"playback"`
	PreviousPlaylistItemID string           `json:"-"`
}

type RoomV2Skipper interface {
	SchedulePlaylistItemMetadataRefresh(ctx context.Context, roomID, playlistItemID string) error
	RoomV2Fetcher
	SessionProfileFetcherCreator
	PlaybackV2Fetcher
	PlaylistItemPositionFetcher
	SkipPlaylistItem(ctx context.Context, roomID, userID string) (*SkipPlaylistItemResult, error)
}
