package vibe

import "fmt"

// PublicErrorKind selects a reviewed client-facing message. Never use dependency
// error text or request data as a public message; add a catalog entry instead.
type PublicErrorKind string

const PublicRoomGenerationBusy PublicErrorKind = "RoomGenerationBusy"

const PublicRoomNameUnavailable PublicErrorKind = "RoomNameUnavailable"

const PublicRoomGenerationDailyLimit PublicErrorKind = "RoomGenerationDailyLimit"

const PublicRoomGenerationSongLimit PublicErrorKind = "RoomGenerationSongLimit"

const PublicSkipSessionRequired PublicErrorKind = "SkipSessionRequired"

const PublicSkipHostRequired PublicErrorKind = "SkipHostRequired"

const PublicSkipRoomAdminRequired PublicErrorKind = "SkipRoomAdminRequired"

const PublicProfileNameInvalid PublicErrorKind = "ProfileNameInvalid"

const PublicRoomNameInvalid PublicErrorKind = "RoomNameInvalid"

const PublicRoomPasswordRequired PublicErrorKind = "PublicRoomPasswordRequired"

const PublicSongRequestInvalid PublicErrorKind = "SongRequestInvalid"

const PublicSongSessionRequired PublicErrorKind = "SongSessionRequired"

const PublicSongRoomNotFound PublicErrorKind = "SongRoomNotFound"

const PublicSongRoomAdminRequired PublicErrorKind = "SongRoomAdminRequired"

const PublicSongProviderDisabled PublicErrorKind = "SongProviderDisabled"

const PublicYouTubeLiveVideoNotSupported PublicErrorKind = "YoutubeLiveVideoNotSupported"

const PublicSongProviderURLInvalid PublicErrorKind = "SongProviderUrlInvalid"

const PublicSongVoteAlreadyExists PublicErrorKind = "SongVoteAlreadyExists"

const PublicPlaylistSessionRequired PublicErrorKind = "PlaylistSessionRequired"

const PublicRoomPlaylistImportDisabled PublicErrorKind = "RoomPlaylistImportDisabled"

const PublicPlaylistRoomAdminRequired PublicErrorKind = "SongRoomAdminRequiredPlaylist"

const PublicPlaylistImportFailed PublicErrorKind = "PlaylistImportFailed"

const PublicYouTubeEmbeddingNotAllowed PublicErrorKind = "YoutubeEmbeddingNotAllowed"

const PublicYouTubeSearchQuotaExhausted PublicErrorKind = "YoutubeSearchQuotaExhausted"

const PublicChatMessageInvalid PublicErrorKind = "ChatMessageInvalid"

// PublicError keeps the internal cause separate from its public representation.
// Limit is a public numeric limit used only by the playlist size message.
type PublicError struct {
	Err        error
	Kind       PublicErrorKind
	Limit      int
	StatusCode int
}

func (e PublicError) Error() string {
	return fmt.Sprintf("error handling request: %v", e.Err)
}

func (e PublicError) Unwrap() error {
	return e.Err
}

// Response resolves only locally defined messages. Unknown kinds fail closed.
func (e PublicError) Response() (*PublicErrorResponse, error) {
	response := PublicErrorResponse{
		Namespace: "vibes-backend",
		Propagate: true,
	}

	switch e.Kind {
	case PublicRoomGenerationBusy:
		response.Error = "room_generation_busy"
		response.Message = "A playlist is already being generated. Please wait and try again."

	case PublicRoomNameUnavailable:
		response.Error = "room_name_unavailable"
		response.Message = "This room name is unavailable or its reservation expired."

	case PublicRoomGenerationDailyLimit:
		response.Error = "room_generation_daily_limit"
		response.Message = "This room has reached its daily playlist generation limit."

	case PublicRoomGenerationSongLimit:
		response.Error = "room_generation_song_limit"
		response.Message = fmt.Sprintf(
			"Playlists can only be generated when the room has %d songs or fewer.",
			e.Limit,
		)

	case PublicSkipSessionRequired:
		response.Error = "skip_session_required"
		response.Message = "Rejoin the room before skipping songs."

	case PublicSkipHostRequired:
		response.Error = "skip_host_required"
		response.Message = "Only the host or a room admin can skip songs in host mode."

	case PublicSkipRoomAdminRequired:
		response.Error = "skip_room_admin_required"
		response.Message = "Only room admins can skip songs here. Log in as a room admin in room settings."

	case PublicProfileNameInvalid:
		response.Error = "profile_name_invalid"
		response.Message = fmt.Sprintf("Use between 1 and %d characters for your name.", SessionNameMaxLength)

	case PublicRoomNameInvalid:
		response.Error = "room_name_invalid"
		response.Message = fmt.Sprintf("Use between 1 and %d characters for the room name.", RoomNameMaxLength)

	case PublicRoomPasswordRequired:
		response.Error = "public_room_password_required"
		response.Message = "Add an admin password before making this room public."

	case PublicSongRequestInvalid:
		response.Error = "song_request_invalid"
		response.Message = "The song details could not be read. Search for the song again and retry."

	case PublicSongSessionRequired:
		response.Error = "song_session_required"
		response.Message = "Your room session is missing. Rejoin the room and try adding the song again."

	case PublicSongRoomNotFound:
		response.Error = "song_room_not_found"
		response.Message = "This room no longer exists. Join another room to add songs."

	case PublicSongRoomAdminRequired:
		response.Error = "song_room_admin_required"
		response.Message = "Only room admins can add songs here. Log in as a room admin in room settings, or ask an admin to allow everyone to add songs."

	case PublicSongProviderDisabled:
		response.Error = "song_provider_disabled"
		response.Message = "This music provider is not enabled for this room. Choose another provider or ask a room admin to enable it."

	case PublicYouTubeLiveVideoNotSupported:
		response.Error = "youtube_live_video_not_supported"
		response.Message = "Live videos cannot be added to rooms."

	case PublicSongProviderURLInvalid:
		response.Error = "song_provider_url_invalid"
		response.Message = "The song link is not valid for this provider. Search for the song again or paste a valid track link."

	case PublicSongVoteAlreadyExists:
		response.Error = "song_vote_already_exists"
		response.Message = "Your vote is already counted for this song."

	case PublicPlaylistSessionRequired:
		response.Error = "playlist_session_required"
		response.Message = "Rejoin the room before importing a playlist."

	case PublicRoomPlaylistImportDisabled:
		response.Error = "room_playlist_import_disabled"
		response.Message = "Playlist importing is disabled in this room."

	case PublicPlaylistRoomAdminRequired:
		response.Error = "song_room_admin_required"
		response.Message = "Only room admins can import playlists here. Log in as a room admin in room settings."

	case PublicPlaylistImportFailed:
		response.Error = "playlist_import_failed"
		response.Message = "The playlist could not be queued. Please try again."

	case PublicYouTubeEmbeddingNotAllowed:
		response.Error = "youtube_embedding_not_allowed"
		response.Message = "This video cannot play outside YouTube. Try another version of the song."

	case PublicYouTubeSearchQuotaExhausted:
		response.Error = "youtube_search_quota_exhausted"
		response.Message = RoomGenerationYouTubeQuotaFailure

	case PublicChatMessageInvalid:
		response.Error = "chat_message_invalid"
		response.Message = fmt.Sprintf("Use between 1 and %d characters for your message.", MessageMaxLength)

	default:
		return nil, fmt.Errorf("error resolving unknown public error kind")
	}

	return &response, nil
}

// PublicErrorResponse preserves the structured error contract used by clients.
// Only catalog messages should populate this response, never upstream bodies.
type PublicErrorResponse struct {
	Namespace string `json:"namespace"`
	Error     string `json:"error"`
	Message   string `json:"message"`
	Propagate bool   `json:"propagate,omitzero"`
}
