package vibe

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type MusicPlaylist struct {
	ID        string        `json:"id"`
	Source    string        `json:"source"`
	Title     string        `json:"title,omitempty"`
	Tracks    []*MusicTrack `json:"tracks"`
	Truncated bool          `json:"truncated"`
}

func (p *MusicPlaylist) GetMusicTracks() []MusicTrack {
	tracks := make([]MusicTrack, 0, len(p.Tracks))
	for _, track := range p.Tracks {
		tracks = append(tracks, *track)
	}

	return tracks
}

type AddPlaylistRequest struct {
	Songs []*AddSongRequest `json:"songs"`
}

type AddPlaylistResult struct {
	Results []*AddSongResult `json:"results"`
}

func ResolveSoundCloudPlaylistURL(value string) (string, error) {
	providerURL, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("error parsing soundcloud playlist URL: %w", err)
	}

	hostname := strings.ToLower(providerURL.Hostname())
	isSoundCloudHost := hostname == "soundcloud.com" ||
		strings.HasSuffix(hostname, ".soundcloud.com")
	pathSegments := strings.Split(strings.Trim(providerURL.Path, "/"), "/")
	isShortLink := hostname == "on.soundcloud.com" && len(pathSegments) >= 1
	isPlaylist := len(pathSegments) >= 3 && pathSegments[1] == "sets"
	if providerURL.Scheme != "https" ||
		!isSoundCloudHost ||
		providerURL.User != nil ||
		(!isShortLink && !isPlaylist) {
		return "", fmt.Errorf("error validating soundcloud playlist URL")
	}

	providerURL.RawQuery = ""
	providerURL.Fragment = ""

	url := providerURL.String()
	return url, nil
}

type MusicPlaylistFetcher interface {
	GetPlaylist(ctx context.Context, id string) (*MusicPlaylist, error)
}

type MusicPlaylistResolver interface {
	ResolvePlaylist(ctx context.Context, providerURL string) (*MusicPlaylist, error)
}
