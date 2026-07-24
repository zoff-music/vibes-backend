package vibe

import "context"

type GeneratedPlaylistRequest struct {
	Prompt string `json:"prompt"`
}

type GeneratedTrack struct {
	Artist         string `json:"artist"`
	Title          string `json:"title"`
	YouTubeVideoID string `json:"youtubeVideoId,omitempty"`
	ThumbnailURL   string `json:"thumbnailUrl,omitempty"`
	Duration       int    `json:"duration,omitempty"`
}

type GeneratedPlaylist struct {
	Tracks []GeneratedTrack `json:"tracks"`
}

type GeneratedRoom struct {
	Room            *Room            `json:"room"`
	Tracks          []GeneratedTrack `json:"tracks"`
	AddedTrackCount int              `json:"addedTrackCount"`
}

type PlaylistGenerator interface {
	GeneratePlaylist(ctx context.Context, prompt string) (*GeneratedPlaylist, error)
}

type GeneratedPlaylistVerifier interface {
	VerifyGeneratedPlaylist(ctx context.Context, playlist *GeneratedPlaylist) error
}

type GeneratedRoomCreator interface {
	RoomNameSuggester
	CreateGeneratedRoom(
		ctx context.Context,
		room *Room,
		playlist *GeneratedPlaylist,
	) (*GeneratedRoom, error)
}

const GeneratedPlaylistTrackCount = 50

const GeneratedPlaylistSystemInstruction = "You curate real, released songs for a YouTube queue. Given a listener description, return exactly 50 distinct songs that fit it. Use canonical artist and track titles and include the best-known YouTube music or official audio video ID for each song. Choose songs, not interviews, podcasts, compilations, mixes, or ordinary videos. Exclude live versions, remixes, covers, duplicates, and anything over 20 minutes; use at most two songs per artist unless explicitly requested. Only include songs you are confident exist. Return no commentary; follow the supplied JSON schema."
