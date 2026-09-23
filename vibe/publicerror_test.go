package vibe

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
)

func TestPublicGenerationError(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		want   string
	}{
		{
			name:   "no failure",
			reason: "",
			want:   "",
		},
		{
			name:   "generic failure",
			reason: RoomGenerationFailure,
			want:   RoomGenerationFailure,
		},
		{
			name:   "quota failure",
			reason: RoomGenerationYouTubeQuotaFailure,
			want:   RoomGenerationYouTubeQuotaFailure,
		},
		{
			name:   "persisted SQL detail",
			reason: "sql: no rows in result set",
			want:   RoomGenerationFailure,
		},
		{
			name:   "persisted provider detail",
			reason: "upstream request failed: token=secret",
			want:   RoomGenerationFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PublicGenerationError(tt.reason)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// Preserve existing frontend error codes and messages while keeping causes private.
func TestPublicErrorContract(t *testing.T) {
	tests := []struct {
		kind    PublicErrorKind
		code    string
		message string
	}{
		{
			kind:    PublicRoomGenerationBusy,
			code:    "room_generation_busy",
			message: "A playlist is already being generated. Please wait and try again.",
		},
		{
			kind:    PublicRoomNameUnavailable,
			code:    "room_name_unavailable",
			message: "This room name is unavailable or its reservation expired.",
		},
		{
			kind:    PublicRoomGenerationDailyLimit,
			code:    "room_generation_daily_limit",
			message: "This room has reached its daily playlist generation limit.",
		},
		{
			kind:    PublicRoomGenerationSongLimit,
			code:    "room_generation_song_limit",
			message: "Playlists can only be generated when the room has 20 songs or fewer.",
		},
		{
			kind:    PublicSkipSessionRequired,
			code:    "skip_session_required",
			message: "Rejoin the room before skipping songs.",
		},
		{
			kind:    PublicSkipHostRequired,
			code:    "skip_host_required",
			message: "Only the host or a room admin can skip songs in host mode.",
		},
		{
			kind:    PublicSkipRoomAdminRequired,
			code:    "skip_room_admin_required",
			message: "Only room admins can skip songs here. Log in as a room admin in room settings.",
		},
		{
			kind:    PublicProfileNameInvalid,
			code:    "profile_name_invalid",
			message: fmt.Sprintf("Use between 1 and %d characters for your name.", SessionNameMaxLength),
		},
		{
			kind:    PublicRoomNameInvalid,
			code:    "room_name_invalid",
			message: fmt.Sprintf("Use between 1 and %d characters for the room name.", RoomNameMaxLength),
		},
		{
			kind:    PublicRoomPasswordRequired,
			code:    "public_room_password_required",
			message: "Add an admin password before making this room public.",
		},
		{
			kind:    PublicSongRequestInvalid,
			code:    "song_request_invalid",
			message: "The song details could not be read. Search for the song again and retry.",
		},
		{
			kind:    PublicSongSessionRequired,
			code:    "song_session_required",
			message: "Your room session is missing. Rejoin the room and try adding the song again.",
		},
		{
			kind:    PublicSongRoomNotFound,
			code:    "song_room_not_found",
			message: "This room no longer exists. Join another room to add songs.",
		},
		{
			kind:    PublicSongRoomAdminRequired,
			code:    "song_room_admin_required",
			message: "Only room admins can add songs here. Log in as a room admin in room settings, or ask an admin to allow everyone to add songs.",
		},
		{
			kind:    PublicSongProviderDisabled,
			code:    "song_provider_disabled",
			message: "This music provider is not enabled for this room. Choose another provider or ask a room admin to enable it.",
		},
		{
			kind:    PublicYouTubeLiveVideoNotSupported,
			code:    "youtube_live_video_not_supported",
			message: "Live videos cannot be added to rooms.",
		},
		{
			kind:    PublicSongProviderURLInvalid,
			code:    "song_provider_url_invalid",
			message: "The song link is not valid for this provider. Search for the song again or paste a valid track link.",
		},
		{
			kind:    PublicSongVoteAlreadyExists,
			code:    "song_vote_already_exists",
			message: "Your vote is already counted for this song.",
		},
		{
			kind:    PublicPlaylistSessionRequired,
			code:    "playlist_session_required",
			message: "Rejoin the room before importing a playlist.",
		},
		{
			kind:    PublicRoomPlaylistImportDisabled,
			code:    "room_playlist_import_disabled",
			message: "Playlist importing is disabled in this room.",
		},
		{
			kind:    PublicPlaylistRoomAdminRequired,
			code:    "song_room_admin_required",
			message: "Only room admins can import playlists here. Log in as a room admin in room settings.",
		},
		{
			kind:    PublicPlaylistImportFailed,
			code:    "playlist_import_failed",
			message: "The playlist could not be queued. Please try again.",
		},
		{
			kind:    PublicYouTubeEmbeddingNotAllowed,
			code:    "youtube_embedding_not_allowed",
			message: "This video cannot play outside YouTube. Try another version of the song.",
		},
		{
			kind:    PublicYouTubeSearchQuotaExhausted,
			code:    "youtube_search_quota_exhausted",
			message: RoomGenerationYouTubeQuotaFailure,
		},
		{
			kind:    PublicChatMessageInvalid,
			code:    "chat_message_invalid",
			message: fmt.Sprintf("Use between 1 and %d characters for your message.", MessageMaxLength),
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			public := PublicError{
				Err:   sql.ErrNoRows,
				Kind:  tt.kind,
				Limit: 20,
			}

			if !errors.Is(public, sql.ErrNoRows) {
				t.Fatal("internal cause was lost")
			}

			response, err := public.Response()
			if err != nil {
				t.Fatal(err)
			}

			if response.Error != tt.code || response.Message != tt.message || response.Namespace != "vibes-backend" || !response.Propagate {
				t.Fatalf("changed public contract: %+v", response)
			}
		})
	}
}
