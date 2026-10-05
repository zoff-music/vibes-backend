package vibe

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Song represents a song in the queue
//
// Deprecated: Use PlaylistItem for new code. Retained for legacy API compatibility.
type Song struct {
	ID                  string    `json:"id"`
	RoomID              string    `json:"-"`
	SourceType          string    `json:"sourceType"`
	SourceID            string    `json:"sourceId"`
	ProviderURL         string    `json:"providerUrl,omitempty"`
	Title               string    `json:"title"`
	Artist              string    `json:"artist,omitempty"`
	ThumbnailURL        string    `json:"thumbnailUrl"`
	Duration            int       `json:"duration"`
	AddedBySessionID    string    `json:"-"`
	AddedBy             string    `json:"addedBy,omitempty"`
	AddedAt             time.Time `json:"addedAt"`
	VoteCount           int       `json:"voteCount"`
	PlaybackRestriction string    `json:"playbackRestriction,omitempty"`
}

// AddSongRequest is the request payload for adding a song.
//
// Deprecated: Use AddPlaylistItemRequest for new code. Retained for legacy API compatibility.
type AddSongRequest struct {
	SourceType  string `json:"sourceType"`
	SourceID    string `json:"sourceId"`
	ProviderURL string `json:"providerUrl,omitempty"`
	Title       string `json:"title"`
	Artist      string `json:"artist,omitempty"`
	Thumbnail   string `json:"thumbnailUrl"`
	Duration    int    `json:"duration"`
}

// Deprecated: Use AddPlaylistItemRequest.CanonicalProviderURL for new code.
func (r AddSongRequest) CanonicalProviderURL() (string, error) {
	request := AddPlaylistItemRequest{
		SourceType:  r.SourceType,
		SourceID:    r.SourceID,
		ProviderURL: r.ProviderURL,
	}

	providerURL, err := request.CanonicalProviderURL()
	if err != nil {
		return "", fmt.Errorf("error resolving legacy item provider URL: %w", err)
	}

	return providerURL, nil
}

func ResolveSoundCloudTrackURL(value string) (string, error) {
	providerURL, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("error parsing soundcloud track URL: %w", err)
	}

	hostname := strings.ToLower(providerURL.Hostname())
	isSoundCloudHost := hostname == "soundcloud.com" ||
		strings.HasSuffix(hostname, ".soundcloud.com")
	pathSegments := strings.Split(strings.Trim(providerURL.Path, "/"), "/")
	isShortLink := hostname == "on.soundcloud.com" && len(pathSegments) >= 1
	if providerURL.Scheme != "https" ||
		!isSoundCloudHost ||
		providerURL.User != nil ||
		(!isShortLink && len(pathSegments) < 2) {
		return "", fmt.Errorf("error validating soundcloud track URL")
	}

	providerURL.RawQuery = ""
	providerURL.Fragment = ""

	url := providerURL.String()
	return url, nil
}

// AddSongResult is the result of adding a song or voting on an existing duplicate.
//
// Deprecated: Use AddPlaylistItemResult for new code. Retained for legacy API compatibility.
type AddSongResult struct {
	Song    Song   `json:"song"`
	Outcome string `json:"outcome"`
}

// IsEmpty returns true if the song is empty/not found
func (s *Song) IsEmpty() bool {
	return s.ID == ""
}

// SongsFetcher fetches songs from the queue
type SongsFetcher interface {
	GetSongs(ctx context.Context, roomID string) ([]Song, error)
}

type SongFetcher interface {
	GetSong(ctx context.Context, roomID, songID string) (*Song, error)
}

// SongAdder adds songs to the queue
type SongAdder interface {
	AddSong(ctx context.Context, song *Song) (*AddSongResult, error)
}

// SongRemover removes songs from the queue
type SongRemover interface {
	RemoveSong(ctx context.Context, roomID, songID string) error
}

// SongVoter votes for a song
type SongVoter interface {
	VoteSong(ctx context.Context, roomID, songID, userID string) error
}

// SongQueueAdder defines the exact operations used when adding a song.
type SongQueueAdder interface {
	SessionProfileFetcherCreator
	SongAdder
	SongsFetcher
	RoomFetcher
	PlaybackController
}

// CachedMusicTrackRoomEventNotifier defines the Redis capabilities used while
// adding one song.
type CachedMusicTrackRoomEventNotifier interface {
	CachedMusicTrackFetcher
	RoomEventNotifier
}

// SongQueueRemover defines the exact operations used when removing a song.
type SongQueueRemover interface {
	SessionProfileFetcherCreator
	SongFetcher
	SongRemover
	SongsFetcher
	RoomFetcher
}

// SongQueueVoter defines the exact operations used when voting for a song.
type SongQueueVoter interface {
	MessageAuthorFetcher
	SongVoter
	SongsFetcher
}

const AddSongOutcomeAdded = "added"
const AddSongOutcomeDuplicateVoted = "duplicate_voted"
const AddSongOutcomeDuplicateAlreadyVoted = "duplicate_already_voted"

const PlaybackRestrictionAge = "age"

const PlaybackRestrictionRegion = "region"

const PlaybackRestrictionEmbedding = "embedding"
