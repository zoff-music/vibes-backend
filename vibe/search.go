package vibe

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"strings"
)

func IsLiveVideo(source string, duration int) bool {
	return source == SourceTypeYouTube && duration <= 0
}

type SearchUsage struct {
	RoomID    string
	Provider  string
	QueryHash string
	Cached    bool
}

func GenerateSearchUsage(
	provider string,
	query string,
	cached bool,
) (*SearchUsage, error) {
	normalizedQuery := NormalizeSearch(query)
	hash := sha256.Sum256([]byte(normalizedQuery))

	return &SearchUsage{
		Provider:  provider,
		QueryHash: hex.EncodeToString(hash[:]),
		Cached:    cached,
	}, nil
}

func NormalizeSearch(query string) string {
	value := strings.ToLower(html.UnescapeString(query))
	value = strings.ReplaceAll(value, "\u200b", "")
	value = strings.ReplaceAll(value, "\ufeff", "")
	value = strings.Join(strings.Fields(value), " ")
	normalizedQuery := value
	// Quoted searches are intentional. Outside quotes, remove only complete
	// bracketed metadata labels, never arbitrary parenthetical song versions.
	if !strings.ContainsAny(value, "\"“”") {
		var cleaned strings.Builder
		for index := 0; index < len(value); {
			closing := byte(0)
			if value[index] == '(' {
				closing = ')'
			}
			if value[index] == '[' {
				closing = ']'
			}
			if closing != 0 {
				end := strings.IndexByte(value[index+1:], closing)
				if end >= 0 {
					end += index + 1
					label := strings.TrimSpace(value[index+1 : end])
					if isSearchMetadata(label) {
						cleaned.WriteByte(' ')
					} else {
						cleaned.WriteString(value[index : end+1])
					}
					index = end + 1
					continue
				}
			}
			cleaned.WriteByte(value[index])
			index++
		}
		normalizedQuery = strings.Join(strings.Fields(cleaned.String()), " ")
	}
	// Preserve artist punctuation and version qualifiers; only drop separate
	// trailing punctuation tokens, not punctuation attached to a title.
	for {
		separator := strings.LastIndex(normalizedQuery, " ")
		if separator < 0 || strings.Trim(normalizedQuery[separator+1:], ".,;:!?…|–—-") != "" {
			break
		}
		normalizedQuery = normalizedQuery[:separator]
	}
	normalizedQuery = strings.TrimSpace(strings.TrimRight(normalizedQuery, ",;"))
	if normalizedQuery == "" {
		return value
	}

	return normalizedQuery
}

func isSearchMetadata(label string) bool {
	switch label {
	case "official", "vevo", "official video", "official music video", "official audio", "official hd video", "official 4k video":
		return true
	default:
		return false
	}
}

func (t GeneratedPlaylistItem) ToProviderItem() (*ProviderItem, error) {
	return &ProviderItem{
		CategoryID:          t.CategoryID,
		ID:                  t.YouTubeID,
		Source:              SourceTypeYouTube,
		ProviderURL:         fmt.Sprintf("https://www.youtube.com/watch?v=%s", t.YouTubeID),
		Title:               t.Title,
		Publisher:           t.Publisher,
		ThumbnailURL:        t.ThumbnailURL,
		Duration:            durationISO8601(t.Duration),
		DurationSeconds:     t.Duration,
		ViewCount:           t.ViewCount,
		LikeCount:           t.LikeCount,
		PlaybackRestriction: t.PlaybackRestriction,
	}, nil
}

func (t ProviderItem) ToGeneratedPlaylistItem(query string) (*GeneratedPlaylistItem, error) {
	return &GeneratedPlaylistItem{
		CategoryID:          t.CategoryID,
		Publisher:           t.Publisher,
		Title:               t.Title,
		YouTubeID:           t.ID,
		ThumbnailURL:        t.ThumbnailURL,
		Duration:            t.DurationSeconds,
		ViewCount:           t.ViewCount,
		LikeCount:           t.LikeCount,
		SearchQuery:         query,
		PlaybackRestriction: t.PlaybackRestriction,
	}, nil
}

func durationISO8601(seconds int) string {
	hours := seconds / (60 * 60)
	minutes := (seconds % (60 * 60)) / 60
	remainingSeconds := seconds % 60
	duration := "PT"
	if hours > 0 {
		duration += fmt.Sprintf("%dH", hours)
	}
	if minutes > 0 {
		duration += fmt.Sprintf("%dM", minutes)
	}
	if remainingSeconds > 0 || duration == "PT" {
		duration += fmt.Sprintf("%dS", remainingSeconds)
	}

	return duration
}

type SearchUsageCreator interface {
	CreateSearchUsages(ctx context.Context, usages []SearchUsage) error
}

type SearchClaim struct {
	Key   string
	Token string
}

func (c *SearchClaim) IsEmpty() bool {
	return c.Token == ""
}

type SearchClaimFetcherReleaser interface {
	GetOrClaimProviderSearch(ctx context.Context, source string, query string, roomType RoomType) (*ProviderSearchResult, error)
	ReleaseSearch(ctx context.Context, claim SearchClaim) error
}

type ProviderSearchResult struct {
	Searches []CachedProviderSearch
	Claim    SearchClaim
}

const SourceTypeYouTube = "youtube"
const SourceTypeSoundCloud = "soundcloud"
