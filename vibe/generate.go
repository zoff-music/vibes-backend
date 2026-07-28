package vibe

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type GeneratedPlaylistRequest struct {
	Prompt string `json:"prompt"`
}

type GeneratedTrack struct {
	Artist       string `json:"artist"`
	Title        string `json:"title"`
	YouTubeID    string `json:"youtubeId,omitempty"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
	Duration     int    `json:"duration,omitempty"`
	ViewCount    uint64 `json:"-"`
	LikeCount    uint64 `json:"-"`
	SearchQuery  string `json:"-"`
}

func (g *GeneratedTrack) IsEmpty() bool {
	return g.YouTubeID == ""
}

type GeneratedPlaylist []GeneratedTrack

type generatedPlaylistPromptTrack struct {
	Artist string `json:"artist,omitempty"`
	Title  string `json:"title"`
}

type generatedPlaylistPrompt struct {
	CurrentSong *generatedPlaylistPromptTrack  `json:"currentlyPlaying,omitempty"`
	Prompt      string                         `json:"listenerRequest"`
	Songs       []generatedPlaylistPromptTrack `json:"existingSongs,omitempty"`
}

type AIModel struct {
	Provider string
	Name     string
}

func ParseAIModel(value string) (*AIModel, error) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		name := strings.TrimSpace(value)
		lowerName := strings.ToLower(name)
		if strings.HasPrefix(lowerName, "grok-") {
			model := AIModel{
				Provider: AIProviderGrok,
				Name:     name,
			}
			return &model, nil
		}
		if strings.HasPrefix(lowerName, "gemini-") {
			model := AIModel{
				Provider: AIProviderGemini,
				Name:     name,
			}
			return &model, nil
		}
		return nil, fmt.Errorf(
			"error parsing AI model %q: expected PROVIDER:model",
			value,
		)
	}

	provider := strings.ToUpper(strings.TrimSpace(parts[0]))
	name := strings.TrimSpace(parts[1])
	if name == "" {
		return nil, fmt.Errorf("error parsing AI model %q: model is required", value)
	}

	if provider != AIProviderGrok && provider != AIProviderGemini {
		return nil, fmt.Errorf(
			"error parsing AI model %q: provider must be GROK or GEMINI",
			value,
		)
	}

	model := AIModel{
		Provider: provider,
		Name:     name,
	}

	return &model, nil
}

type GeneratedPlaylistSearchResult struct {
	Playlist       GeneratedPlaylist
	CachedSearches []CachedSearch
	SearchUsages   []SearchUsage
}

type RoomGenerationUpdate struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type RoomGeneration struct {
	RoomID    string
	Prompt    string
	Attempt   int
	Exhausted bool
}

type PlaylistGenerator interface {
	GeneratePlaylist(ctx context.Context, prompt string) (*GeneratedPlaylist, error)
}

type GeneratedPlaylistSearcher interface {
	SearchGeneratedPlaylist(
		ctx context.Context,
		playlist GeneratedPlaylist,
		cachedSearches []CachedSearch,
	) (*GeneratedPlaylistSearchResult, error)
}

type GeneratedSongAdder interface {
	AddGeneratedSong(ctx context.Context, song *Song) (*Song, error)
}

type GeneratedRoomCreator interface {
	RoomNameSuggester
	RoomCreator
	RoomGenerationCreator
	RoomGenerationAvailabilityChecker
}

type RoomGenerationCreator interface {
	CreateRoomGeneration(ctx context.Context, roomID string, prompt string) error
}

type RoomGenerationAvailabilityChecker interface {
	HasActiveRoomGeneration(ctx context.Context) (bool, error)
}

type RoomGenerationProcessor interface {
	ProcessNextRoomGeneration(ctx context.Context) (*RoomGeneration, error)
}

type RoomGenerationCompleter interface {
	CompleteRoomGeneration(ctx context.Context, roomID string) error
}

type RoomGenerationFailer interface {
	FailRoomGeneration(ctx context.Context, roomID string, reason string) error
}

type RoomGenerationCleaner interface {
	DeleteExpiredRoomGenerations(ctx context.Context, olderThan time.Duration) (int64, error)
}

type RoomGenerationWorker interface {
	RoomGenerationProcessor
	RoomGenerationCompleter
	RoomGenerationFailer
	RoomFetcher
	SongsFetcher
	GeneratedSongAdder
	PlaybackController
	PlaybackFetcher
	SearchUsageCreator
}

func GeneratePlaylistPrompt(
	prompt string,
	currentSong *Song,
	songs []Song,
) (string, error) {
	if currentSong == nil && len(songs) == 0 {
		return prompt, nil
	}

	generatedPrompt := generatedPlaylistPrompt{
		Prompt: prompt,
		Songs:  make([]generatedPlaylistPromptTrack, 0, len(songs)),
	}
	if currentSong != nil {
		generatedPrompt.CurrentSong = &generatedPlaylistPromptTrack{
			Artist: currentSong.Artist,
			Title:  currentSong.Title,
		}
	}
	for _, song := range songs {
		generatedPrompt.Songs = append(
			generatedPrompt.Songs,
			generatedPlaylistPromptTrack{
				Artist: song.Artist,
				Title:  song.Title,
			},
		)
	}

	body, err := json.Marshal(generatedPrompt)
	if err != nil {
		return "", fmt.Errorf("error marshaling generated playlist prompt: %w", err)
	}

	return string(body), nil
}

func GeneratedPlaylistSystemInstruction(trackCount int) string {
	return strings.ReplaceAll(
		generatedPlaylistSystemInstruction,
		"{trackCount}",
		fmt.Sprintf("%d", trackCount),
	)
}

const AIProviderGrok = "GROK"
const AIProviderGemini = "GEMINI"
const RoomGenerationRetention = 24 * time.Hour
const RoomGenerationGenerating = "generating"
const RoomGenerationCompleted = "completed"
const RoomGenerationFailed = "failed"
const RoomGenerationFailure = "Could not finish generating this playlist. You can try again."
const RoomGenerationYouTubeQuotaFailure = "YouTube search has reached its daily limit. Try again after midnight Pacific time."
const generatedPlaylistSystemInstruction = `
You generate playlists from a listener's natural-language request.

Generate up to {trackCount} distinct, real, publicly released songs that closely match the listener's request. Return as many high-quality matches as you can find if there are not enough strong matches for all {trackCount}. Never invent or include weakly related songs simply to reach the limit.

Interpret the request using any stated genres, moods, themes, eras, languages, artists, activities, energy levels, lyrical topics, or exclusions.

The listener message may be a JSON object containing "listenerRequest", "currentlyPlaying", and "existingSongs". When that context is present, never suggest the currently playing song or any song already in the room. Treat alternate releases or versions of an existing song as the same song unless the listener explicitly requests them.

Requirements:

1. Return only a valid JSON array.
2. The array must contain between 1 and {trackCount} objects.
3. Every object must use exactly this shape:
   {"title":"song title","artist":"artist name","youtubeId":"optional YouTube video ID"}
4. The only permitted fields are:

- "title"
- "artist"
- "youtubeId"
5. Strongly prefer songs whose exact YouTube video ID you know. Maximize the number of objects containing "youtubeId" while preserving playlist quality and relevance.
6. When two songs are equally relevant, prefer the song whose exact YouTube video ID you know.
7. Omit "youtubeId" entirely unless you are highly confident that it is the correct YouTube video ID for that exact song and artist.
8. Never invent, estimate, derive, or fabricate a YouTube video ID.
9. Do not search the web, call tools, or claim that any information was verified externally.
10. A YouTube video ID must contain exactly 11 characters and only the ID itself, never a URL, query string, timestamp, playlist ID, or other metadata.
11. Prefer the official music video, official audio upload, or an official artist or label upload when you confidently know its YouTube ID.
12. Use canonical, commonly recognized song titles and artist names.
13. Include only songs that genuinely exist and have been publicly released.
14. Never invent songs, artists, collaborations, alternate titles, or release versions.
15. Do not include duplicate songs.
16. Treat remasters, deluxe editions, radio edits, live recordings, acoustic versions, sped-up versions, slowed versions, and reuploads as the same song unless the listener explicitly requests those versions.
17. Do not include multiple recordings of the same composition unless the listener explicitly asks for covers or alternate versions.
18. Match the listener's request as closely as possible. Prioritize relevance over quantity.
19. When the request is broad, create a coherent playlist with reasonable variety across artists and songs.
20. Avoid overrepresenting a single artist unless the listener explicitly requests an artist-focused playlist.
21. Respect all negative constraints, such as excluded artists, genres, themes, languages, decades, or explicit content.
22. When the request is ambiguous, choose the most natural musical interpretation rather than asking follow-up questions.
23. Order the songs intentionally so the playlist has a sensible progression in mood, energy, chronology, or style where appropriate.
24. Ensure all JSON strings are properly escaped.
25. Do not use trailing commas.
26. Do not wrap the JSON in Markdown or a code block.
27. Do not include explanations, headings, notes, warnings, citations, or any text outside the JSON array.

Before returning the result, silently verify that:

- The array contains between 1 and {trackCount} objects.
- Every song is distinct.
- Every song is real and publicly released.
- Every object contains only the permitted fields.
- Every included YouTube ID is one you are highly confident is correct.
- The output is valid JSON.
`
