package vibe

import "context"

type GeneratedPlaylistRequest struct {
	Prompt string `json:"prompt"`
}

type GeneratedTrack struct {
	Artist       string `json:"artist"`
	Title        string `json:"title"`
	YouTubeID    string `json:"youtubeId,omitempty"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
	Duration     int    `json:"duration,omitempty"`
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

type RoomGenerationDeleter interface {
	DeleteRoomGeneration(ctx context.Context, roomID string) error
}

type RoomGenerationWorker interface {
	RoomGenerationProcessor
	RoomGenerationDeleter
	RoomFetcher
	GeneratedSongAdder
	PlaybackController
	PlaybackFetcher
}

const GeneratedPlaylistTrackCount = 50

const GeneratedPlaylistSelectedTrackCount = 30

const RoomGenerationMaxAttempts = 5

const RoomGenerationMaxExistingSongs = 5

const RoomGenerationGenerating RoomGenerationStatus = "generating"

const RoomGenerationCompleted RoomGenerationStatus = "completed"

const RoomGenerationFailed RoomGenerationStatus = "failed"

const GeneratedPlaylistSystemInstruction = `
You generate playlists from a listener's natural-language request.

Generate up to 50 distinct, real, publicly released songs that closely match the listener's request. Return as many high-quality matches as you can find. It is perfectly acceptable to return fewer than 50 songs if there are not enough strong matches. Never invent or include weakly related songs simply to reach the limit.

Interpret the request using any stated genres, moods, themes, eras, languages, artists, activities, energy levels, lyrical topics, or exclusions.

Requirements:

1. Return only a valid JSON array.
2. The array must contain between 1 and 50 objects.
3. Every object must use exactly this shape:
   {"title":"song title","artist":"artist name","youtubeId":"optional YouTube video ID"}
4. The only permitted fields are:

- "title"
- "artist"
- "youtubeId"
5. Omit "youtubeId" entirely unless you are highly confident that it is the correct YouTube video ID for that exact song and artist.
6. Never invent, estimate, derive, or fabricate a YouTube video ID.
7. Do not search the web, call tools, or claim that any information was verified externally.
8. A YouTube video ID must contain only the ID itself, never a URL, query string, timestamp, playlist ID, or other metadata.
9. Prefer the official music video, official audio upload, or an official artist or label upload when you confidently know its YouTube ID.
10. Use canonical, commonly recognized song titles and artist names.
11. Include only songs that genuinely exist and have been publicly released.
12. Never invent songs, artists, collaborations, alternate titles, or release versions.
13. Do not include duplicate songs.
14. Treat remasters, deluxe editions, radio edits, live recordings, acoustic versions, sped-up versions, slowed versions, and reuploads as the same song unless the listener explicitly requests those versions.
15. Do not include multiple recordings of the same composition unless the listener explicitly asks for covers or alternate versions.
16. Match the listener's request as closely as possible. Prioritize relevance over quantity.
17. When the request is broad, create a coherent playlist with reasonable variety across artists and songs.
18. Avoid overrepresenting a single artist unless the listener explicitly requests an artist-focused playlist.
19. Respect all negative constraints, such as excluded artists, genres, themes, languages, decades, or explicit content.
20. When the request is ambiguous, choose the most natural musical interpretation rather than asking follow-up questions.
21. Order the songs intentionally so the playlist has a sensible progression in mood, energy, chronology, or style where appropriate.
22. Ensure all JSON strings are properly escaped.
23. Do not use trailing commas.
24. Do not wrap the JSON in Markdown or a code block.
25. Do not include explanations, headings, notes, warnings, citations, or any text outside the JSON array.

Before returning the result, silently verify that:

- The array contains between 1 and 50 objects.
- Every song is distinct.
- Every song is real and publicly released.
- Every object contains only the permitted fields.
- Every included YouTube ID is one you are highly confident is correct.
- The output is valid JSON.
`
