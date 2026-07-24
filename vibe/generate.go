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

type GeneratedPlaylist []GeneratedTrack

type GeneratedRoom struct {
	Room   Room              `json:"room"`
	Tracks GeneratedPlaylist `json:"tracks"`
}

type PlaylistGenerator interface {
	GeneratePlaylist(ctx context.Context, prompt string) (*GeneratedPlaylist, error)
}

type GeneratedPlaylistSearcher interface {
	SearchGeneratedPlaylist(ctx context.Context, playlist GeneratedPlaylist) (*GeneratedPlaylist, error)
}

type GeneratedRoomCreator interface {
	RoomNameSuggester
	CreateGeneratedRoom(
		ctx context.Context,
		room Room,
		playlist GeneratedPlaylist,
	) (*GeneratedRoom, error)
}

const GeneratedPlaylistTrackCount = 50

const GeneratedPlaylistSystemInstruction = "Generate exactly 50 distinct, real, released songs matching the listener's request. Return only a valid JSON array. Every item must use this exact shape: {\"title\":\"song title\",\"artist\":\"artist name\",\"youtubeId\":\"optional YouTube video ID\"}. Omit youtubeId when you are not confident it is correct. Use canonical song and artist names, no other fields, and no commentary or Markdown."
