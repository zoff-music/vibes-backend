package vibe

import (
	"context"
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
}

func (g *GeneratedTrack) IsEmpty() bool {
	return g.YouTubeID == ""
}

type GeneratedPlaylist []GeneratedTrack

type RoomGenerationStatus string

type RoomGenerationUpdate struct {
	Status RoomGenerationStatus `json:"status"`
	Error  string               `json:"error,omitempty"`
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
	SearchGeneratedPlaylist(ctx context.Context, playlist GeneratedPlaylist) (*GeneratedPlaylist, error)
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
	GeneratedSongAdder
	PlaybackController
	PlaybackFetcher
}

const GeneratedPlaylistTrackCount = 100

const GeneratedPlaylistSelectedTrackCount = 30

const RoomGenerationMaxAttempts = 5

const RoomGenerationMaxDailyCount = 2

const RoomGenerationMaxExistingSongs = 5

const RoomGenerationRetention = 24 * time.Hour

const RoomGenerationGenerating RoomGenerationStatus = "generating"

const RoomGenerationCompleted RoomGenerationStatus = "completed"

const RoomGenerationFailed RoomGenerationStatus = "failed"

const RoomGenerationFailure = "Could not finish generating this playlist. You can try again."

const RoomGenerationYouTubeQuotaFailure = "YouTube search has reached its daily limit. Try again after midnight Pacific time."

const GeneratedPlaylistSystemInstruction = `
You generate playlists from a listener's natural-language request.

Generate 100 distinct, real, publicly released songs that closely match the listener's request. Return as many high-quality matches as you can find if there are not enough strong matches for all 100. Never invent or include weakly related songs simply to reach the limit.

Interpret the request using any stated genres, moods, themes, eras, languages, artists, activities, energy levels, lyrical topics, or exclusions.

Requirements:

1. Return only a valid JSON array.
2. The array must contain between 1 and 100 objects.
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

- The array contains between 1 and 100 objects.
- Every song is distinct.
- Every song is real and publicly released.
- Every object contains only the permitted fields.
- Every included YouTube ID is one you are highly confident is correct.
- The output is valid JSON.
`
