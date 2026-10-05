package vibe

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// PlaylistItem is one room queue entry, not the underlying provider resource.
type PlaylistItem struct {
	ID                  string    `json:"id"`
	RoomID              string    `json:"-"`
	SourceType          string    `json:"sourceType"`
	SourceID            string    `json:"sourceId"`
	ProviderURL         string    `json:"providerUrl,omitempty"`
	Title               string    `json:"title"`
	Publisher           string    `json:"publisher,omitempty"`
	ThumbnailURL        string    `json:"thumbnailUrl"`
	Duration            int       `json:"duration"`
	AddedBySessionID    string    `json:"-"`
	AddedBy             string    `json:"addedBy,omitempty"`
	AddedAt             time.Time `json:"addedAt"`
	VoteCount           int       `json:"voteCount"`
	PlaybackRestriction string    `json:"playbackRestriction,omitempty"`
}

func (p *PlaylistItem) IsEmpty() bool {
	return p.ID == ""
}

// ToSong preserves the original API representation for installed clients.
func (p PlaylistItem) ToSong() *Song {
	return &Song{
		ID:                  p.ID,
		RoomID:              p.RoomID,
		SourceType:          p.SourceType,
		SourceID:            p.SourceID,
		ProviderURL:         p.ProviderURL,
		Title:               p.Title,
		Artist:              p.Publisher,
		ThumbnailURL:        p.ThumbnailURL,
		Duration:            p.Duration,
		AddedBySessionID:    p.AddedBySessionID,
		AddedBy:             p.AddedBy,
		AddedAt:             p.AddedAt,
		VoteCount:           p.VoteCount,
		PlaybackRestriction: p.PlaybackRestriction,
	}
}

func (s Song) ToPlaylistItem() *PlaylistItem {
	return &PlaylistItem{
		ID:                  s.ID,
		RoomID:              s.RoomID,
		SourceType:          s.SourceType,
		SourceID:            s.SourceID,
		ProviderURL:         s.ProviderURL,
		Title:               s.Title,
		Publisher:           s.Artist,
		ThumbnailURL:        s.ThumbnailURL,
		Duration:            s.Duration,
		AddedBySessionID:    s.AddedBySessionID,
		AddedBy:             s.AddedBy,
		AddedAt:             s.AddedAt,
		VoteCount:           s.VoteCount,
		PlaybackRestriction: s.PlaybackRestriction,
	}
}

type PlaylistItemsFetcher interface {
	GetPlaylistItems(ctx context.Context, roomID string) ([]PlaylistItem, error)
}

type PlaylistItemFetcher interface {
	GetPlaylistItem(ctx context.Context, roomID string, playlistItemID string) (*PlaylistItem, error)
}

type AddPlaylistItemRequest struct {
	SourceType  string `json:"sourceType"`
	SourceID    string `json:"sourceId"`
	ProviderURL string `json:"providerUrl,omitempty"`
	Title       string `json:"title"`
	Publisher   string `json:"publisher,omitempty"`
	Thumbnail   string `json:"thumbnailUrl"`
	Duration    int    `json:"duration"`
}

type AddPlaylistItemResult struct {
	PlaylistItem PlaylistItem `json:"playlistItem"`
	Outcome      string       `json:"outcome"`
}

func (r AddPlaylistItemResult) ToAddSongResult() *AddSongResult {
	song := r.PlaylistItem.ToSong()

	return &AddSongResult{
		Song:    *song,
		Outcome: r.Outcome,
	}
}

const AddPlaylistItemOutcomeAdded = "added"
const AddPlaylistItemOutcomeDuplicateVoted = "duplicate_voted"
const AddPlaylistItemOutcomeDuplicateAlreadyVoted = "duplicate_already_voted"

func (r AddPlaylistItemRequest) CanonicalProviderURL() (string, error) {
	if r.SourceType == SourceTypeYouTube {
		providerURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", r.SourceID)
		return providerURL, nil
	}

	if r.SourceType != SourceTypeSoundCloud || r.ProviderURL == "" {
		return "", nil
	}

	providerURL, err := url.Parse(r.ProviderURL)
	if err != nil {
		return "", fmt.Errorf("error parsing soundcloud provider URL: %w", err)
	}

	hostname := strings.ToLower(providerURL.Hostname())
	isSoundCloudHost := hostname == "soundcloud.com" ||
		strings.HasSuffix(hostname, ".soundcloud.com")
	pathSegments := strings.Split(strings.Trim(providerURL.Path, "/"), "/")
	if providerURL.Scheme != "https" ||
		!isSoundCloudHost ||
		providerURL.User != nil ||
		len(pathSegments) < 2 {
		return "", fmt.Errorf("error validating soundcloud provider URL")
	}

	url := providerURL.String()
	return url, nil
}

type PlaylistItemAdder interface {
	AddPlaylistItem(ctx context.Context, item *PlaylistItem) (*AddPlaylistItemResult, error)
}

type PlaylistItemRemover interface {
	RemovePlaylistItem(ctx context.Context, roomID, playlistItemID string) error
}

type PlaylistItemVoter interface {
	VotePlaylistItem(ctx context.Context, roomID, playlistItemID, userID string) error
}

type PlaylistItemQueueAdder interface {
	SessionProfileFetcherCreator
	PlaylistItemAdder
	PlaylistItemsFetcher
	RoomV2Fetcher
	PlaybackV2Controller
}

type PlaylistItemQueueRemover interface {
	SessionProfileFetcherCreator
	PlaylistItemFetcher
	PlaylistItemRemover
	PlaylistItemsFetcher
	RoomV2Fetcher
}

type PlaylistItemQueueVoter interface {
	SessionProfileFetcherCreator
	RoomV2Fetcher
	PlaylistItemVoter
	PlaylistItemsFetcher
}
