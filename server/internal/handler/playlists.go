package handler

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/internalerror"
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
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/youtube/playlists/{id} [get]
//
// Deprecated: Use GET /api/v2/youtube/playlists/{id}. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use GET /api/v2/youtube/playlists/{id} for the playlist-item contract. This endpoint retains its existing payloads.
func GetMusicPlaylist(
	fetcher vibe.ProviderPlaylistFetcher,
	cache vibe.CachedProviderItemsCreator,
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

		playlist, err := fetcher.GetProviderPlaylist(ctx, id)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting provider playlist: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = cache.CacheProviderItems(ctx, provider, playlist.Items)
		if err != nil {
			log.Printf("error caching provider playlist track metadata: %v", err)
		}

		legacy := playlist.ToMusicPlaylist()

		body, err := json.Marshal(legacy)
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
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/soundcloud/playlists [get]
//
// Deprecated: Use GET /api/v2/soundcloud/playlists. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use GET /api/v2/soundcloud/playlists for the playlist-item contract. This endpoint retains its existing payloads.
func ResolveSoundCloudPlaylist(
	resolver vibe.ProviderPlaylistResolver,
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

		playlist, err := resolver.ResolveProviderPlaylist(ctx, providerURL)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error resolving soundcloud playlist: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		legacy := playlist.ToMusicPlaylist()

		body, err := json.Marshal(legacy)
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
//	@Success	202		{object}	vibe.AddPlaylistResult
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	403		{object}	vibe.ErrorResponse
//	@Failure	404		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v1/rooms/{id}/playlists [post]
//
// Deprecated: Use POST /api/v2/rooms/{id}/playlists. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use POST /api/v2/rooms/{id}/playlists for the playlist-item contract. This endpoint retains its existing payloads.
func AddPlaylist(
	db vibe.PlaylistImportRoomCreator,
	cache vibe.CachedProviderItemsFetcher,
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
		if len(req.Songs) == 0 || len(req.Songs) > playlistImportItemLimit {
			handleError(
				w,
				fmt.Errorf(
					"error playlist must contain between 1 and %d songs",
					playlistImportItemLimit,
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
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error importing playlist: missing session"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "playlist_session_required",
						Message:   "Rejoin the room before importing a playlist.",
						Propagate: true,
					},
					StatusCode: http.StatusUnauthorized,
				},
				http.StatusUnauthorized,
				false,
			)
			return
		}

		room, err := db.GetRoomV2(ctx, roomID, session.UserID)
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
		if room.Settings.OnlyAdminAddPlaylistItems && !room.IsAdmin {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error only admins can import playlists in this room"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "song_room_admin_required",
						Message:   "Only room admins can import playlists here. Log in as a room admin in room settings.",
						Propagate: true,
					},
					StatusCode: http.StatusForbidden,
				},
				http.StatusForbidden,
				false,
			)
			return
		}

		cacheKeys := make([]vibe.CachedProviderItemKey, 0, len(req.Songs))
		for _, requestedSong := range req.Songs {
			if requestedSong.SourceID == "" || requestedSong.SourceType == "" {
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
			if vibe.IsLiveVideo(requestedSong.SourceType, requestedSong.Duration) {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: fmt.Errorf("error playlist contains a live video"),
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "youtube_live_video_not_supported",
							Message:   "Live videos cannot be added to rooms.",
							Propagate: true,
						},
						StatusCode: http.StatusBadRequest,
					},
					http.StatusBadRequest,
					false,
				)
				return
			}

			cacheKeys = append(cacheKeys, vibe.CachedProviderItemKey{
				Provider: requestedSong.SourceType,
				ID:       requestedSong.SourceID,
			})
		}

		cachedItems, err := cache.GetCachedProviderItems(ctx, cacheKeys)
		if err != nil {
			log.Printf("error getting cached provider playlist track metadata: %v", err)
			cachedItems = nil
		}

		metadata := make(map[vibe.CachedProviderItemKey]vibe.ProviderItem, len(cachedItems))
		for _, item := range cachedItems {
			key := vibe.CachedProviderItemKey{Provider: item.Source, ID: item.ID}
			metadata[key] = item
		}

		playlistItems := make([]vibe.PlaylistItem, 0, len(req.Songs))
		for _, requestedSong := range req.Songs {

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
			key := vibe.CachedProviderItemKey{Provider: requestedSong.SourceType, ID: requestedSong.SourceID}
			cachedTrack := metadata[key]
			if requestedSong.SourceType == vibe.SourceTypeYouTube && cachedTrack.IsEmpty() {
				handleError(w, client.ErrorCodeWrapper{
					Err: fmt.Errorf("error importing youtube playlist: verified metadata is unavailable"),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "youtube_track_verification_required",
						Message:   "This playlist's availability needs to be checked again. Paste its link again before importing it.",
						Propagate: true,
					},
					StatusCode: http.StatusBadRequest,
				}, http.StatusBadRequest, false)
				return
			}

			if !cachedTrack.IsEmpty() {
				if vibe.IsLiveVideo(
					requestedSong.SourceType,
					cachedTrack.DurationSeconds,
				) {
					handleError(
						w,
						client.ErrorCodeWrapper{
							Err: fmt.Errorf("error playlist contains a live video"),
							ResponseBody: client.ErrorCodeResponseBody{
								Namespace: "vibes-backend",
								Error:     "youtube_live_video_not_supported",
								Message:   "Live videos cannot be added to rooms.",
								Propagate: true,
							},
							StatusCode: http.StatusBadRequest,
						},
						http.StatusBadRequest,
						false,
					)
					return
				}
				playbackRestriction = cachedTrack.PlaybackRestriction
				if requestedSong.SourceType == vibe.SourceTypeYouTube {
					requestedSong.Title = cachedTrack.Title
					requestedSong.Artist = cachedTrack.Publisher
					requestedSong.Thumbnail = cachedTrack.ThumbnailURL
					requestedSong.Duration = cachedTrack.DurationSeconds
				}
			}

			playlistItems = append(playlistItems, vibe.PlaylistItem{
				ID:                  uuid.New().String(),
				RoomID:              roomID,
				SourceType:          requestedSong.SourceType,
				SourceID:            requestedSong.SourceID,
				ProviderURL:         providerURL,
				PlaybackRestriction: playbackRestriction,
				Title:               requestedSong.Title,
				Publisher:           requestedSong.Artist,
				ThumbnailURL:        requestedSong.Thumbnail,
				Duration:            requestedSong.Duration,
				AddedBySessionID:    session.UserID,
				AddedAt:             time.Now(),
			})
		}

		importID := uuid.NewString()
		// Staged items are invisible to the worker until the import is published.
		// Bound the entire enqueue so disconnected requests cannot leave work running.
		importCtx, cancelImport := context.WithTimeout(ctx, 2*time.Minute)
		defer cancelImport()

		published := false
		defer func() {
			if published {
				return
			}

			cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
			defer cancelCleanup()

			cleanupErr := db.DeletePlaylistImport(cleanupCtx, importID)
			if cleanupErr != nil {
				log.Printf("error cleaning incomplete playlist import %s: %v", importID, cleanupErr)
			}
		}()

		for position, playlistItem := range playlistItems {
			err = db.CreatePlaylistImportItem(importCtx, importID, position, playlistItem)
			if err != nil {
				handleError(
					w,
					client.ErrorCodeWrapper{
						Err: fmt.Errorf("error staging playlist item %d: %w", position, err),
						ResponseBody: client.ErrorCodeResponseBody{
							Namespace: "vibes-backend",
							Error:     "playlist_import_failed",
							Message:   "The playlist could not be queued. Please try again.",
							Propagate: true,
						},
						StatusCode: http.StatusInternalServerError,
					},
					http.StatusInternalServerError,
					true,
				)
				return
			}
		}

		err = db.CreatePlaylistImport(importCtx, importID, roomID, session.UserID, len(playlistItems))
		if err != nil {
			handleError(
				w,
				client.ErrorCodeWrapper{
					Err: fmt.Errorf("error creating playlist import: %w", err),
					ResponseBody: client.ErrorCodeResponseBody{
						Namespace: "vibes-backend",
						Error:     "playlist_import_failed",
						Message:   "The playlist could not be queued. Please try again.",
						Propagate: true,
					},
					StatusCode: http.StatusInternalServerError,
				},
				http.StatusInternalServerError,
				true,
			)
			return
		}

		published = true

		response := vibe.AddPlaylistResult{
			ImportID:    importID,
			QueuedCount: len(playlistItems),
		}
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(body)
	}
}

const playlistImportItemLimit = 500

type MetaRefresh struct {
	DB           vibe.PlaylistItemMetadataRefreshStorage
	Events       vibe.RoomEventV3Notifier
	Provider     vibe.ProviderItemFetcher
	ProviderName string
}

func (h *MetaRefresh) Handle(ctx context.Context, _ []byte) error {
	refresh, err := h.DB.ClaimPlaylistItemMetadataRefresh(ctx, h.ProviderName, metadataRefreshRetry)
	if err != nil {
		return fmt.Errorf(
			"error claiming %s metadata in MetaRefresh.Handle: %w",
			h.ProviderName,
			err,
		)
	}

	providerItem, err := h.Provider.GetProviderItem(ctx, refresh.SourceID)
	if err != nil {
		var quotaError internalerror.ErrProviderQuotaExceeded
		if errors.As(err, &quotaError) {
			err = h.DB.DeferPlaylistItemMetadataRefresh(ctx, refresh.PlaylistItemID, metadataQuotaRetry)
			if err != nil {
				return fmt.Errorf(
					"error deferring %s metadata refresh in MetaRefresh.Handle: %w",
					h.ProviderName,
					err,
				)
			}

			return fmt.Errorf(
				"error fetching rate-limited %s metadata in MetaRefresh.Handle: %w",
				h.ProviderName,
				quotaError,
			)
		}

		var notFoundError internalerror.ErrProviderItemNotFound
		var liveVideoError internalerror.ErrLiveVideo
		var madeForKidsError internalerror.ErrMadeForKids
		if !errors.As(err, &notFoundError) &&
			!errors.As(err, &liveVideoError) &&
			!errors.As(err, &madeForKidsError) {
			return fmt.Errorf(
				"error fetching %s metadata in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}

		err = h.DB.RemovePlaylistItem(ctx, refresh.RoomID, refresh.PlaylistItemID)
		if err != nil {
			return fmt.Errorf(
				"error removing unavailable %s playlist item in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}

		playlistItems, err := h.DB.GetPlaylistItems(ctx, refresh.RoomID)
		if err != nil {
			return fmt.Errorf(
				"error fetching playlist items after removing unavailable %s playlist item in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}

		payload, err := json.Marshal(playlistItems)
		if err != nil {
			return fmt.Errorf(
				"error marshaling playlist items after removing unavailable %s playlist item in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}
		compactPayload, err := json.Marshal(vibe.PlaylistItemIDUpdate{ID: refresh.PlaylistItemID})
		if err != nil {
			return fmt.Errorf(
				"error marshaling compact unavailable %s playlist item event in MetaRefresh.Handle: %w",
				h.ProviderName,
				err,
			)
		}
		compactEvent := &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemRemoved, Payload: compactPayload}

		err = h.Events.NotifyRoomUpdateV3(ctx, refresh.RoomID, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: payload,
			Compact: compactEvent,
		})
		if err != nil {
			log.Printf(
				"failed to notify room after removing unavailable %s playlist item: %v",
				h.ProviderName,
				err,
			)
		}

		playback, err := h.DB.GetPlaybackStateV2(ctx, refresh.RoomID)
		if err != nil {
			return fmt.Errorf("error fetching playback after metadata removal in MetaRefresh.Handle: %w", err)
		}

		playbackPayload, err := json.Marshal(playback)
		if err != nil {
			return fmt.Errorf("error marshaling playback after metadata removal in MetaRefresh.Handle: %w", err)
		}

		err = h.Events.NotifyRoomUpdateV3(ctx, refresh.RoomID, vibe.RoomEventV3{
			Type:    vibe.PlaybackUpdate,
			Payload: playbackPayload,
		})
		if err != nil {
			return fmt.Errorf("error notifying playback after metadata removal in MetaRefresh.Handle: %w", err)
		}

		return nil
	}

	err = h.DB.RefreshPlaylistItemMetadata(ctx, *refresh, *providerItem, metadataRefreshInterval)
	if err != nil {
		return fmt.Errorf(
			"error refreshing %s metadata in MetaRefresh.Handle: %w",
			h.ProviderName,
			err,
		)
	}

	playlistItems, err := h.DB.GetPlaylistItems(ctx, refresh.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching refreshed queue in MetaRefresh.Handle: %w", err)
	}

	for position, playlistItem := range playlistItems {
		if playlistItem.ID != refresh.PlaylistItemID {
			continue
		}

		payload, err := json.Marshal(playlistItems)
		if err != nil {
			return fmt.Errorf("error marshaling refreshed queue in MetaRefresh.Handle: %w", err)
		}

		compactPayload, err := json.Marshal(vibe.PlaylistItemPositionUpdate{PlaylistItem: playlistItem, Position: position})
		if err != nil {
			return fmt.Errorf("error marshaling refreshed playlist item in MetaRefresh.Handle: %w", err)
		}

		err = h.Events.NotifyRoomUpdateV3(ctx, refresh.RoomID, vibe.RoomEventV3{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: payload,
			Compact: &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemUpdated, Payload: compactPayload},
		})
		if err != nil {
			return fmt.Errorf("error notifying refreshed playlist item in MetaRefresh.Handle: %w", err)
		}

		break
	}

	playback, err := h.DB.GetPlaybackStateV2(ctx, refresh.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching refreshed playback in MetaRefresh.Handle: %w", err)
	}

	if playback.CurrentPlaylistItem != nil && playback.CurrentPlaylistItem.ID == refresh.PlaylistItemID {
		payload, err := json.Marshal(playback)
		if err != nil {
			return fmt.Errorf("error marshaling refreshed playback in MetaRefresh.Handle: %w", err)
		}

		err = h.Events.NotifyRoomUpdateV3(ctx, refresh.RoomID, vibe.RoomEventV3{
			Type:    vibe.PlaybackUpdate,
			Payload: payload,
		})
		if err != nil {
			return fmt.Errorf("error notifying refreshed playback in MetaRefresh.Handle: %w", err)
		}
	}

	return nil
}

type ExpirePlaylistItemMetadata struct {
	DB     vibe.PlaylistItemMetadataExpiryFetcher
	Events vibe.RoomBatchEventV3Notifier
}

func (h *ExpirePlaylistItemMetadata) Handle(ctx context.Context, _ []byte) error {
	expiry, err := h.DB.ExpirePlaylistItemMetadata(ctx)
	if err != nil {
		return fmt.Errorf("error expiring metadata in ExpirePlaylistItemMetadata.Handle: %w", err)
	}

	playlistItems, err := h.DB.GetPlaylistItems(ctx, expiry.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching queue in ExpirePlaylistItemMetadata.Handle: %w", err)
	}

	payload, err := json.Marshal(playlistItems)
	if err != nil {
		return fmt.Errorf("error marshaling queue in ExpirePlaylistItemMetadata.Handle: %w", err)
	}

	playback, err := h.DB.GetPlaybackStateV2(ctx, expiry.RoomID)
	if err != nil {
		return fmt.Errorf("error fetching playback in ExpirePlaylistItemMetadata.Handle: %w", err)
	}

	playbackPayload, err := json.Marshal(playback)
	if err != nil {
		return fmt.Errorf("error marshaling playback in ExpirePlaylistItemMetadata.Handle: %w", err)
	}

	err = h.Events.NotifyRoomUpdatesV3(ctx, expiry.RoomID, []vibe.RoomEventV3{
		{
			Type:    vibe.PlaylistItemsUpdate,
			Payload: payload,
			Compact: &vibe.RoomEventV3Payload{Type: vibe.PlaylistItemsSnapshot, Payload: payload},
		},
		{
			Type:    vibe.PlaybackUpdate,
			Payload: playbackPayload,
		},
	})
	if err != nil {
		return fmt.Errorf("error notifying metadata expiry in ExpirePlaylistItemMetadata.Handle: %w", err)
	}

	return nil
}

const metadataRefreshInterval = 21 * 24 * time.Hour
const metadataRefreshRetry = 5 * time.Minute
const metadataQuotaRetry = 24 * time.Hour
