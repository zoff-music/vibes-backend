package vibe

import "context"

type ProviderPlaylist struct {
	ID                      string         `json:"id"`
	Source                  string         `json:"source"`
	Title                   string         `json:"title,omitempty"`
	Items                   []ProviderItem `json:"items"`
	Truncated               bool           `json:"truncated"`
	SkippedEmbeddingCount   int            `json:"skippedEmbeddingCount"`
	SkippedMadeForKidsCount int            `json:"skippedMadeForKidsCount"`
}

func (p *ProviderPlaylist) IsEmpty() bool {
	return p.ID == ""
}

func (p ProviderPlaylist) ToMusicPlaylist() *MusicPlaylist {
	var tracks []MusicTrack
	if p.Items != nil {
		tracks = make([]MusicTrack, len(p.Items))
		for index, item := range p.Items {
			tracks[index] = *item.ToMusicTrack()
		}
	}

	return &MusicPlaylist{
		ID:                      p.ID,
		Source:                  p.Source,
		Title:                   p.Title,
		Tracks:                  tracks,
		Truncated:               p.Truncated,
		SkippedEmbeddingCount:   p.SkippedEmbeddingCount,
		SkippedMadeForKidsCount: p.SkippedMadeForKidsCount,
	}
}

type ProviderPlaylistFetcher interface {
	GetProviderPlaylist(ctx context.Context, id string) (*ProviderPlaylist, error)
}

type ProviderPlaylistResolver interface {
	ResolveProviderPlaylist(ctx context.Context, providerURL string) (*ProviderPlaylist, error)
}
