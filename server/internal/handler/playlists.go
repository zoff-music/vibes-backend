package handler

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// GetMusicPlaylist handles provider playlist lookups by ID.
//
//	@Summary	Get a provider playlist
//	@Tags		providers
//	@Produce	json
//	@Param		id	path		string	true	"Playlist ID"
//	@Success	200	{object}	vibe.MusicPlaylist
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/youtube/playlists/{id} [get]
func GetMusicPlaylist(
	fetcher vibe.MusicPlaylistFetcher,
	cache vibe.CachedMusicTrackCreator,
	provider string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		id := vars["id"]
		validID := id != ""
		for _, character := range id {
			isLetter := character >= 'A' && character <= 'Z' ||
				character >= 'a' && character <= 'z'
			isNumber := character >= '0' && character <= '9'
			if !isLetter && !isNumber && character != '-' && character != '_' {
				validID = false
				break
			}
		}
		if !validID {
			handleError(
				w,
				fmt.Errorf("error invalid playlist id"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		playlist, err := fetcher.GetPlaylist(ctx, id)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting provider playlist: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = cache.CacheMusicTracks(ctx, provider, playlist.GetMusicTracks())
		if err != nil {
			log.Printf("error caching provider playlist track metadata: %v", err)
		}

		body, err := json.Marshal(playlist)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling provider playlist: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// ResolveSoundCloudPlaylist handles SoundCloud playlist URL lookups.
//
//	@Summary	Resolve a SoundCloud playlist URL
//	@Tags		providers
//	@Produce	json
//	@Param		url	query		string	true	"SoundCloud playlist URL"
//	@Success	200	{object}	vibe.MusicPlaylist
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/soundcloud/playlists [get]
func ResolveSoundCloudPlaylist(
	resolver vibe.MusicPlaylistResolver,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		providerURL, err := vibe.ResolveSoundCloudPlaylistURL(r.URL.Query().Get("url"))
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating soundcloud playlist URL: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		playlist, err := resolver.ResolvePlaylist(ctx, providerURL)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error resolving soundcloud playlist: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(playlist)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling soundcloud playlist: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// AddPlaylist handles importing resolved playlist tracks into a room.
//
//	@Summary	Add a playlist to a room
//	@Tags		playlists
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Room ID"
//	@Param		request	body		vibe.AddPlaylistRequest	true	"Playlist tracks"
//	@Success	200		{object}	vibe.AddPlaylistResult
//	@Success	201		{object}	vibe.AddPlaylistResult
//	@Failure	400		{object}	map[string]string
//	@Failure	401		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	404		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/rooms/{id}/playlists [post]
func AddPlaylist(
	db vibe.SongQueueAdder,
	notifier vibe.RoomBatchEventNotifier,
	cache vibe.CachedMusicTrackFetcher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]

		var req vibe.AddPlaylistRequest
		err := json.UnmarshalRead(r.Body, &req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding playlist request body: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}
		if len(req.Songs) == 0 || len(req.Songs) > playlistImportTrackLimit {
			handleError(
				w,
				fmt.Errorf(
					"error playlist must contain between 1 and %d songs",
					playlistImportTrackLimit,
				),
				http.StatusBadRequest,
				false,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		room, err := db.GetRoom(ctx, roomID, session.UserID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching room for playlist import: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if room.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}
		if !room.Settings.PlaylistImport {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error playlist import is disabled for this room"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "room_playlist_import_disabled",
						Message:   "Playlist importing is disabled in this room.",
						Propagate: true,
					},
					StatusCode: http.StatusForbidden,
				},
				http.StatusForbidden,
				false,
			)
			return
		}
		if room.Settings.OnlyAdminAddSongs && !room.IsAdmin {
			handleError(
				w,
				fmt.Errorf("error only admins can add songs in this room"),
				http.StatusForbidden,
				false,
			)
			return
		}

		queuedSongs, err := db.GetSongs(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching songs before playlist import: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		songs := make([]*vibe.Song, 0, len(req.Songs))
		for _, requestedSong := range req.Songs {
			if requestedSong == nil {
				handleError(
					w,
					fmt.Errorf("error playlist contains an empty song"),
					http.StatusBadRequest,
					false,
				)
				return
			}

			sourceEnabled := false
			for _, source := range room.Settings.EnabledSources {
				if requestedSong.SourceType == source {
					sourceEnabled = true
					break
				}
			}
			if !sourceEnabled {
				handleError(
					w,
					fmt.Errorf(
						"error source type %s is not enabled for this room",
						requestedSong.SourceType,
					),
					http.StatusBadRequest,
					false,
				)
				return
			}

			providerURL, err := requestedSong.CanonicalProviderURL()
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error getting canonical provider URL: %w", err),
					http.StatusBadRequest,
					false,
				)
				return
			}

			playbackRestriction := ""
			cachedTrack, err := cache.GetCachedMusicTrack(
				ctx,
				requestedSong.SourceType,
				requestedSong.SourceID,
			)
			if err != nil {
				log.Printf("error getting cached provider playlist track metadata: %v", err)
			}
			if err == nil && !cachedTrack.IsEmpty() {
				playbackRestriction = cachedTrack.PlaybackRestriction
			}

			songs = append(songs, &vibe.Song{
				ID:                  uuid.New().String(),
				RoomID:              roomID,
				SourceType:          requestedSong.SourceType,
				SourceID:            requestedSong.SourceID,
				ProviderURL:         providerURL,
				PlaybackRestriction: playbackRestriction,
				Title:               requestedSong.Title,
				Artist:              requestedSong.Artist,
				ThumbnailURL:        requestedSong.Thumbnail,
				Duration:            requestedSong.Duration,
				AddedBy:             session.UserID,
				AddedAt:             time.Now(),
			})
		}

		results := make([]*vibe.AddSongResult, 0, len(songs))
		var firstAddedSong *vibe.Song
		for _, song := range songs {
			result, err := db.AddSong(ctx, song)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error adding playlist song: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			results = append(results, result)
			if firstAddedSong == nil && result.Outcome == vibe.AddSongOutcomeAdded {
				addedSong := result.Song
				firstAddedSong = &addedSong
			}
		}

		updatedSongs, err := db.GetSongs(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching songs after playlist import: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		songsPayload, err := json.Marshal(updatedSongs)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling songs after playlist import: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		events := []vibe.RoomEvent{{
			Type:    vibe.QueueReordered,
			Payload: songsPayload,
			Origin:  session.EventOrigin,
		}}
		if len(queuedSongs) == 0 && firstAddedSong != nil {
			playbackState := &vibe.PlaybackState{
				RoomID:       roomID,
				CurrentSong:  firstAddedSong,
				IsPlaying:    true,
				PositionMs:   0,
				UpdatedAt:    time.Now(),
				ServerTimeMs: int(time.Now().UnixMilli()),
			}
			err = db.UpsertPlaybackState(ctx, playbackState)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error auto-playing imported playlist: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			playbackPayload, err := json.Marshal(playbackState)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshaling imported playlist playback: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			events = append(events, vibe.RoomEvent{
				Type:    vibe.PlaybackUpdate,
				Payload: playbackPayload,
				Origin:  session.EventOrigin,
			})
		}

		err = notifier.NotifyRoomUpdates(context.WithoutCancel(ctx), roomID, events)
		if err != nil {
			log.Printf("failed to notify room after playlist import: %v", err)
		}

		response := vibe.AddPlaylistResult{Results: results}
		body, err := json.Marshal(response)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling playlist import response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		status := http.StatusOK
		if firstAddedSong != nil {
			status = http.StatusCreated
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}
}

const playlistImportTrackLimit = 500
