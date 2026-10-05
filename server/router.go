package server

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zoff-music/vibes-backend/server/internal/handler"
	"github.com/zoff-music/vibes-backend/server/internal/middleware"
	_ "github.com/zoff-music/vibes-backend/swaggerdocs"
	"github.com/zoff-music/vibes-backend/vibe"
)

// setupRoutes - the root route function.
func (s *Server) setupRoutes() {
	apiV1 := s.Router.PathPrefix(v1API).Subrouter()
	apiV2 := s.Router.PathPrefix(v2API).Subrouter()
	apiV3 := s.Router.PathPrefix(v3API).Subrouter()
	apiV3.HandleFunc("/rooms/public", handler.GetPublicRoomsV3(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetPublicRoomsV3")
	apiV3.HandleFunc("/rooms/{id}/events", handler.RoomEventsV3(s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("RoomEventsV3")
	s.Router.Handle(swaggerAPI, http.RedirectHandler(swaggerAPI+"/", http.StatusPermanentRedirect)).Methods(http.MethodGet)

	docs := s.Router.PathPrefix(swaggerAPI + "/").Subrouter()

	if s.Config.AdminPasswordPepper != "" {
		docs.HandleFunc("/admin.json", handler.AdminDocumentation()).Methods(http.MethodGet).Name("AdminDocumentation")
		s.addSessionMiddleware(docs)
		s.addAdminMiddleware(docs)
	}

	docs.PathPrefix("/").Handler(handler.Documentation(s.Config.DocumentationDirectory)).Methods(http.MethodGet)

	// Room routes
	apiV1.HandleFunc("/rooms", handler.CreateRoom(s.DB)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRoom")
	apiV2.HandleFunc("/rooms", handler.CreateRoomV2(s.DB)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRoomV2")
	apiV1.HandleFunc("/rooms/reservations", handler.ReserveRoomName(s.DB)).Methods(http.MethodPost, http.MethodOptions).Name("ReserveRoomName")
	apiV1.HandleFunc("/rooms/suggestions", handler.SuggestRoomName(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("SuggestRoomName")
	apiV1.HandleFunc("/rooms/public", handler.GetPublicRooms(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetPublicRooms")
	apiV2.HandleFunc("/rooms/public", handler.GetPublicRoomsV2(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetPublicRoomsV2")
	apiV1.HandleFunc("/rooms/generation", handler.CreateGeneratedRoom(s.DB)).Methods(http.MethodPost, http.MethodOptions).Name("CreateGeneratedRoom")
	apiV2.HandleFunc("/rooms/generation", handler.CreateGeneratedRoomV2(s.DB)).Methods(http.MethodPost, http.MethodOptions).Name("CreateGeneratedRoomV2")
	apiV1.HandleFunc("/rooms/{id}/generations", handler.CreateRoomGeneration(s.DB, s.Config.RoomGenerationMaxExistingPlaylistItems)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRoomGeneration")
	apiV1.HandleFunc("/rooms/{id}", handler.RoomExists(s.DB)).Methods(http.MethodHead).Name("RoomExists")
	apiV1.HandleFunc("/rooms/{id}", handler.GetRoom(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetRoom")
	apiV2.HandleFunc("/rooms/{id}", handler.GetRoomV2(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetRoomV2")
	apiV1.HandleFunc("/rooms/{id}/settings", handler.UpdateRoomSettings(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateRoomSettings")
	apiV2.HandleFunc("/rooms/{id}/settings", handler.UpdateRoomSettingsV2(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateRoomSettingsV2")
	apiV1.HandleFunc("/rooms/{id}/skips", handler.SkipSong(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("SkipSong")
	apiV2.HandleFunc("/rooms/{id}/skips", handler.SkipPlaylistItem(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("SkipPlaylistItem")
	apiV1.HandleFunc("/rooms/{id}/states", handler.GetPlaybackState(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetPlaybackState")
	apiV2.HandleFunc("/rooms/{id}/states", handler.GetPlaybackStateV2(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetPlaybackStateV2")
	apiV1.HandleFunc("/rooms/{id}/states", handler.UpdatePlaybackState(s.DB, s.Redis)).Methods(http.MethodPut, http.MethodOptions).Name("UpdatePlaybackState")
	apiV2.HandleFunc("/rooms/{id}/states", handler.UpdatePlaybackStateV2(s.DB, s.Redis)).Methods(http.MethodPut, http.MethodOptions).Name("UpdatePlaybackStateV2")
	apiV1.HandleFunc("/rooms/{id}/playbackfailures", handler.ReportPlaybackFailure(s.DB, s.YouTube, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("ReportPlaybackFailure")
	apiV2.HandleFunc("/rooms/{id}/failures", handler.ReportPlaybackFailureV2(s.DB, s.YouTube, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("ReportPlaybackFailureV2")
	apiV1.HandleFunc("/rooms/{id}/sessions", handler.CreateSession(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("CreateSession")
	apiV1.HandleFunc("/rooms/{id}/sessions", handler.DeleteRoomAdminSession(s.DB)).Methods(http.MethodDelete, http.MethodOptions).Name("DeleteRoomAdminSession")
	apiV2.HandleFunc("/rooms/{id}/sessions", handler.CreateSessionV2(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("CreateSessionV2")
	apiV2.HandleFunc("/rooms/{id}/sessions", handler.DeleteRoomAdminSessionV2(s.DB)).Methods(http.MethodDelete, http.MethodOptions).Name("DeleteRoomAdminSessionV2")
	apiV1.HandleFunc("/sessions", handler.GetSessionProfile(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetSessionProfile")
	apiV1.HandleFunc("/sessions", handler.UpdateSessionProfile(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateSessionProfile")

	// Playlist item routes and legacy song contracts.
	apiV2.HandleFunc("/rooms/{id}/playlist-items", handler.GetPlaylistItems(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetPlaylistItems")
	apiV2.HandleFunc("/rooms/{id}/playlist-items", handler.AddPlaylistItem(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("AddPlaylistItem")
	apiV2.HandleFunc("/rooms/{id}/playlist-items/{playlistItemId}", handler.RemovePlaylistItem(s.DB, s.Redis)).Methods(http.MethodDelete, http.MethodOptions).Name("RemovePlaylistItem")
	apiV2.HandleFunc("/rooms/{id}/playlist-items/{playlistItemId}", handler.VotePlaylistItem(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("VotePlaylistItem")
	apiV1.HandleFunc("/rooms/{id}/songs", handler.GetSongs(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetSongs")
	apiV1.HandleFunc("/rooms/{id}/songs", handler.AddSong(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("AddSong")
	apiV1.HandleFunc("/rooms/{id}/playlists", handler.AddPlaylist(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("AddPlaylist")
	apiV2.HandleFunc("/rooms/{id}/playlists", handler.AddPlaylistV2(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("AddPlaylistV2")
	apiV1.HandleFunc("/rooms/{id}/songs/{songId}", handler.RemoveSong(s.DB, s.Redis)).Methods(http.MethodDelete, http.MethodOptions).Name("RemoveSong")
	apiV1.HandleFunc("/rooms/{id}/songs/{songId}", handler.VoteSong(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("VoteSong")

	// SSE route
	apiV1.HandleFunc("/rooms/{id}/messages", handler.CreateMessages(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("CreateMessages")
	apiV1.HandleFunc("/rooms/{id}/messages", handler.Messages(s.DB, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("Messages")
	apiV1.HandleFunc("/rooms/{id}/events", handler.RoomEvents(s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("RoomEvents")
	apiV2.HandleFunc("/rooms/{id}/events", handler.RoomEventsV2(s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("RoomEventsV2")

	// YouTube routes
	apiV2.HandleFunc("/rooms/{id}/search/{provider:youtube}", handler.SearchYouTubeV2(s.YouTube, s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("SearchYouTubeV2")
	apiV2.HandleFunc("/rooms/{id}/search/{provider:soundcloud}", handler.SearchSoundCloudV2(s.SoundCloud, s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("SearchSoundCloudV2")
	apiV1.HandleFunc("/youtube/search", handler.SearchMusic(s.YouTube, s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("SearchMusic")
	apiV1.HandleFunc("/youtube/videos/{id}", handler.GetMusicTrack(s.YouTube, s.Redis, vibe.SourceTypeYouTube)).Methods(http.MethodGet, http.MethodOptions).Name("GetMusicTrack")
	apiV2.HandleFunc("/youtube/videos/{id}", handler.GetYouTubeItemV2(s.YouTube, s.Redis, vibe.SourceTypeYouTube)).Methods(http.MethodGet, http.MethodOptions).Name("GetYouTubeItemV2")
	apiV1.HandleFunc("/youtube/playlists/{id}", handler.GetMusicPlaylist(s.YouTube, s.Redis, vibe.SourceTypeYouTube)).Methods(http.MethodGet, http.MethodOptions).Name("GetYouTubePlaylist")
	apiV2.HandleFunc("/youtube/playlists/{id}", handler.GetYouTubePlaylistV2(s.YouTube, s.Redis, vibe.SourceTypeYouTube)).Methods(http.MethodGet, http.MethodOptions).Name("GetYouTubePlaylistV2")

	// SoundCloud routes
	apiV1.HandleFunc("/soundcloud/search", handler.SearchSoundCloud(s.SoundCloud, s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("SearchSoundCloud")
	apiV1.HandleFunc("/soundcloud/tracks", handler.ResolveSoundCloudTrack(s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("ResolveSoundCloudTrack")
	apiV2.HandleFunc("/soundcloud/items", handler.ResolveSoundCloudItemV2(s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("ResolveSoundCloudItemV2")
	apiV1.HandleFunc("/soundcloud/tracks/{id}", handler.GetSoundCloudTrack(s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("GetSoundCloudTrack")
	apiV2.HandleFunc("/soundcloud/items/{id}", handler.GetSoundCloudItemV2(s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("GetSoundCloudItemV2")
	apiV1.HandleFunc("/soundcloud/playlists", handler.ResolveSoundCloudPlaylist(s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("ResolveSoundCloudPlaylist")
	apiV2.HandleFunc("/soundcloud/playlists", handler.ResolveSoundCloudPlaylistV2(s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("ResolveSoundCloudPlaylistV2")

	apiV1.HandleFunc("/tokens/soundcloud", handler.GetToken(s.DB, s.SoundCloud, "soundcloud")).Methods(http.MethodGet, http.MethodOptions).Name("GetSoundCloudToken")
	apiV1.HandleFunc("/tokens/youtube", handler.GetToken(s.DB, s.YouTube, "youtube")).Methods(http.MethodGet, http.MethodOptions).Name("GetYouTubeToken")

	// Authorization routes
	apiV1.HandleFunc("/authorizations/soundcloud", handler.Authorize(s.DB, s.SoundCloud, "soundcloud")).Methods(http.MethodGet, http.MethodOptions).Name("Authorize")
	apiV1.HandleFunc("/authorizations/youtube", handler.Authorize(s.DB, s.YouTube, "youtube")).Methods(http.MethodGet, http.MethodOptions).Name("Authorize")
	// Callbacks
	apiV1.HandleFunc("/callbacks/soundcloud", handler.OAuthCallback(s.DB, s.SoundCloud, "soundcloud")).Methods(http.MethodGet, http.MethodOptions).Name("SoundCloudCallback")
	apiV1.HandleFunc("/callbacks/youtube", handler.OAuthCallback(s.DB, s.YouTube, "youtube")).Methods(http.MethodGet, http.MethodOptions).Name("YouTubeCallback")

	// Config routes
	apiV1.HandleFunc("/providers", handler.GetProviders(s.Config)).Methods(http.MethodGet, http.MethodOptions).Name("GetProviders")

	// Stats routes
	apiV1.HandleFunc("/stats", handler.GetStats(s.DB, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("GetStats")
	apiV2.HandleFunc("/stats", handler.GetStatsV2(s.DB, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("GetStatsV2")

	// Cast token endpoint (cookie-auth only)
	apiV1.HandleFunc("/tokens/casting", handler.CreateCastingToken(s.DB, s.Config.CastTokenSecret)).Methods(http.MethodPost, http.MethodOptions).Name("CreateCastingToken")

	// Remote control routes
	apiV1.HandleFunc("/remotes", handler.CreateRemoteControl(s.DB, s.Redis, s.Config.CookieSecret, s.Config.RemotePairingTTL)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRemoteControl")
	apiV2.HandleFunc("/remotes", handler.CreateRemoteControlV2(s.DB, s.Redis, s.Config.CookieSecret, s.Config.RemotePairingTTL)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRemoteControlV2")
	apiV1.HandleFunc("/remotes", handler.GetOwnedRemoteControl(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetOwnedRemoteControl")
	apiV2.HandleFunc("/remotes", handler.GetOwnedRemoteControlV2(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetOwnedRemoteControlV2")
	apiV1.HandleFunc("/remotes/{id}", handler.GetRemoteControl(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetRemoteControl")
	apiV2.HandleFunc("/remotes/{id}", handler.GetRemoteControlV2(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetRemoteControlV2")
	apiV1.HandleFunc("/remotes/{id}", handler.UpdateRemoteControl(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateRemoteControl")
	apiV2.HandleFunc("/remotes/{id}", handler.UpdateRemoteControlV2(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateRemoteControlV2")
	apiV1.HandleFunc("/remotes/{id}", handler.DeleteRemoteControl(s.DB)).Methods(http.MethodDelete, http.MethodOptions).Name("DeleteRemoteControl")
	apiV1.HandleFunc("/remotes/{id}/sessions", handler.PairRemoteControl(s.DB, s.Redis, s.Config.CookieSecret)).Methods(http.MethodPost, http.MethodOptions).Name("PairRemoteControl")
	apiV2.HandleFunc("/remotes/{id}/sessions", handler.PairRemoteControlV2(s.DB, s.Redis, s.Config.CookieSecret)).Methods(http.MethodPost, http.MethodOptions).Name("PairRemoteControlV2")
	apiV1.HandleFunc("/remotes/{id}/events", handler.RemoteEvents(s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("RemoteEvents")
	apiV2.HandleFunc("/remotes/{id}/events", handler.RemoteEventsV2(s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("RemoteEventsV2")

	// Admin routes

	if s.Config.AdminPasswordPepper != "" {
		apiV1.HandleFunc("/admin/sessions", handler.AdminLogin(s.DB, s.Config.AdminPasswordPepper, s.Config.CookieSecret)).Methods(http.MethodPost, http.MethodOptions).Name("AdminLogin")
		apiV1.HandleFunc("/admin/sessions", handler.AdminSession()).Methods(http.MethodGet, http.MethodOptions).Name("AdminSession")
		apiV1.HandleFunc("/admin/sessions", handler.AdminLogout()).Methods(http.MethodDelete, http.MethodOptions).Name("AdminLogout")
		apiV1.HandleFunc("/admin/users", handler.AdminUsers(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminUsers")
		apiV1.HandleFunc("/admin/users", handler.AdminCreateUser(s.DB, s.Config.AdminPasswordPepper)).Methods(http.MethodPost, http.MethodOptions).Name("AdminCreateUser")
		apiV1.HandleFunc("/admin/users/{id}", handler.AdminUpdateUser(s.DB, s.Config.AdminPasswordPepper)).Methods(http.MethodPatch, http.MethodOptions).Name("AdminUpdateUser")
		apiV1.HandleFunc("/admin/users/{id}", handler.AdminDeleteUser(s.DB)).Methods(http.MethodDelete, http.MethodOptions).Name("AdminDeleteUser")
		apiV1.HandleFunc("/admin/rooms", handler.AdminRooms(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminRooms")
		apiV2.HandleFunc("/admin/rooms", handler.AdminRoomsV2(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminRoomsV2")
		apiV1.HandleFunc("/admin/messages/usage", handler.AdminMessageUsage(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminMessageUsage")
		apiV1.HandleFunc("/admin/searches/usage", handler.AdminSearchUsage(s.DB, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("AdminSearchUsage")
		apiV1.HandleFunc("/admin/listeners/usage", handler.AdminListenerUsage(s.DB, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("AdminListenerUsage")
		apiV1.HandleFunc("/admin/rooms/{id}", handler.AdminUpdateRoom(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("AdminUpdateRoom")
		apiV2.HandleFunc("/admin/rooms/{id}", handler.AdminUpdateRoomV2(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("AdminUpdateRoomV2")
		apiV1.HandleFunc("/admin/rooms/{id}", handler.AdminDeleteRoom(s.DB, s.Redis)).Methods(http.MethodDelete, http.MethodOptions).Name("AdminDeleteRoom")
		apiV2.HandleFunc("/admin/rooms/{id}", handler.AdminDeleteRoomV2(s.DB, s.Redis)).Methods(http.MethodDelete, http.MethodOptions).Name("AdminDeleteRoomV2")
		apiV1.HandleFunc("/admin/events", handler.AdminEvents(s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminEvents")
		apiV2.HandleFunc("/admin/events", handler.AdminEventsV2(s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminEventsV2")
	}

	s.addSessionMiddleware(apiV1, apiV2, apiV3)
	s.addRateLimitMiddleware(apiV1, apiV2, apiV3)
	s.addPermissionMiddleware(apiV1, apiV2, apiV3)

	if s.Config.AdminPasswordPepper != "" {
		s.addAdminMiddleware(apiV1, apiV2)
	}
	s.addTracingAndMetrics(apiV1, apiV2, apiV3)
	s.addCORSMiddleware(s.Router)
}

func (s *Server) addRateLimitMiddleware(routers ...*mux.Router) {
	rm := middleware.RateLimitMiddleware{
		Enabled: s.Config.RateLimitEnabled,
		RequiredRoutes: map[string]bool{
			"AdminLogin":     true,
			"CreateMessages": true,
		},
		Checker: s.Redis,
		Policies: map[string]vibe.RateLimitPolicy{
			"CreateRoom":   {Rate: time.Minute, Limit: 10},
			"CreateRoomV2": {Bucket: "CreateRoom", Rate: time.Minute, Limit: 10},
			"ReserveRoomName": {
				Bucket:  "RoomNameReservations",
				Rate:    time.Minute,
				Limit:   12,
				IPLimit: 30,
			},
			"SuggestRoomName": {
				Bucket:  "RoomNameReservations",
				Rate:    time.Minute,
				Limit:   12,
				IPLimit: 30,
			},
			"RoomExists":              {Rate: time.Minute, Limit: 60},
			"GetRoom":                 {Rate: time.Minute, Limit: 120},
			"GetRoomV2":               {Bucket: "GetRoom", Rate: time.Minute, Limit: 120},
			"GetPublicRooms":          {Rate: time.Minute, Limit: 120},
			"GetPublicRoomsV2":        {Rate: time.Minute, Limit: 120},
			"GetPublicRoomsV3":        {Bucket: "GetPublicRoomsV2", Rate: time.Minute, Limit: 120},
			"UpdateRoomSettings":      {Rate: time.Minute, Limit: 30},
			"UpdateRoomSettingsV2":    {Bucket: "UpdateRoomSettings", Rate: time.Minute, Limit: 30},
			"SkipSong":                {Rate: time.Minute, Limit: 60},
			"SkipPlaylistItem":        {Bucket: "SkipSong", Rate: time.Minute, Limit: 60},
			"GetPlaybackState":        {Rate: time.Minute, Limit: 240},
			"GetPlaybackStateV2":      {Bucket: "GetPlaybackState", Rate: time.Minute, Limit: 240},
			"UpdatePlaybackState":     {Rate: time.Minute, Limit: 240},
			"UpdatePlaybackStateV2":   {Bucket: "UpdatePlaybackState", Rate: time.Minute, Limit: 240},
			"ReportPlaybackFailure":   {Rate: time.Minute, Limit: 30},
			"ReportPlaybackFailureV2": {Bucket: "ReportPlaybackFailure", Rate: time.Minute, Limit: 30},
			"CreateSession":           {Rate: time.Minute, Limit: 30},
			"CreateSessionV2":         {Bucket: "CreateSession", Rate: time.Minute, Limit: 30},
			"GetSongs":                {Bucket: "GetSongs", Rate: time.Minute, Limit: 120},
			"GetPlaylistItems":        {Bucket: "GetSongs", Rate: time.Minute, Limit: 120},
			"AddSong":                 {Rate: time.Minute, Limit: 60},
			"AddPlaylistItem":         {Bucket: "AddSong", Rate: time.Minute, Limit: 60},
			"AddPlaylist":             {Rate: time.Minute, Limit: 6},
			"AddPlaylistV2":           {Bucket: "AddPlaylist", Rate: time.Minute, Limit: 6},
			"RemoveSong":              {Rate: time.Minute, Limit: 60},
			"RemovePlaylistItem":      {Bucket: "RemoveSong", Rate: time.Minute, Limit: 60},
			"VoteSong":                {Rate: time.Minute, Limit: 120},
			"VotePlaylistItem":        {Bucket: "VoteSong", Rate: time.Minute, Limit: 120},
			"CreateMessages":          {Rate: time.Second, Limit: 1},
			"Messages":                {Rate: time.Minute, Limit: 30},
			"RoomEvents":              {Rate: time.Minute, Limit: 30},
			"RoomEventsV2":            {Rate: time.Minute, Limit: 30},
			"RoomEventsV3":            {Bucket: "RoomEventsV2", Rate: time.Minute, Limit: 30},
			"SearchMusic": {
				Bucket:  "YouTubeSearch",
				Rate:    time.Second,
				Limit:   1,
				IPLimit: 3,
			},
			"GetMusicTrack":               {Rate: time.Minute, Limit: 120},
			"GetYouTubeItemV2":            {Bucket: "GetMusicTrack", Rate: time.Minute, Limit: 120},
			"GetYouTubePlaylistV2":        {Bucket: "GetYouTubePlaylist", Rate: time.Minute, Limit: 10},
			"GetSoundCloudItemV2":         {Bucket: "GetSoundCloudTrack", Rate: time.Minute, Limit: 120},
			"ResolveSoundCloudItemV2":     {Bucket: "ResolveSoundCloudTrack", Rate: time.Second, Limit: 3, IPLimit: 30},
			"ResolveSoundCloudPlaylistV2": {Bucket: "ResolveSoundCloudPlaylist", Rate: time.Minute, Limit: 10},
			"GetYouTubePlaylist":          {Rate: time.Minute, Limit: 10},
			"SearchSoundCloud": {
				Bucket:  "SearchSoundCloud",
				Rate:    time.Second,
				Limit:   1,
				IPLimit: 3,
			},
			"SearchYouTubeV2": {
				Bucket:  "YouTubeSearch",
				Rate:    time.Second,
				Limit:   1,
				IPLimit: 3,
			},
			"SearchSoundCloudV2": {
				Bucket:  "SearchSoundCloud",
				Rate:    time.Second,
				Limit:   1,
				IPLimit: 3,
			},
			"GetSoundCloudTrack": {Rate: time.Minute, Limit: 120},
			"ResolveSoundCloudTrack": {
				Rate:    time.Second,
				Limit:   3,
				IPLimit: 30,
			},
			"ResolveSoundCloudPlaylist": {Rate: time.Minute, Limit: 10},
			"GetSoundCloudToken":        {Rate: time.Minute, Limit: 60},
			"GetYouTubeToken":           {Rate: time.Minute, Limit: 60},
			"CreateGeneratedRoom": {
				Bucket:      "RoomGeneration",
				Rate:        10 * time.Minute,
				Limit:       1,
				IPLimit:     2,
				GlobalRate:  time.Minute,
				GlobalLimit: 1,
			},
			"CreateGeneratedRoomV2": {
				Bucket:      "RoomGeneration",
				Rate:        10 * time.Minute,
				Limit:       1,
				IPLimit:     2,
				GlobalRate:  time.Minute,
				GlobalLimit: 1,
			},
			"CreateRoomGeneration": {
				Bucket:      "RoomGeneration",
				Rate:        10 * time.Minute,
				Limit:       1,
				IPLimit:     2,
				GlobalRate:  time.Minute,
				GlobalLimit: 1,
			},
			"Authorize":               {Rate: 10 * time.Minute, Limit: 20},
			"SoundCloudCallback":      {Rate: 10 * time.Minute, Limit: 30},
			"YouTubeCallback":         {Rate: 10 * time.Minute, Limit: 30},
			"GetProviders":            {Rate: time.Minute, Limit: 120},
			"GetStats":                {Rate: time.Minute, Limit: 600},
			"GetStatsV2":              {Bucket: "GetStats", Rate: time.Minute, Limit: 600},
			"CreateCastingToken":      {Rate: time.Minute, Limit: 30},
			"CreateRemoteControl":     {Rate: time.Minute, Limit: 10},
			"CreateRemoteControlV2":   {Bucket: "CreateRemoteControl", Rate: time.Minute, Limit: 10},
			"GetOwnedRemoteControl":   {Rate: time.Minute, Limit: 120},
			"GetOwnedRemoteControlV2": {Bucket: "GetOwnedRemoteControl", Rate: time.Minute, Limit: 120},
			"GetRemoteControl":        {Rate: time.Minute, Limit: 120},
			"GetRemoteControlV2":      {Bucket: "GetRemoteControl", Rate: time.Minute, Limit: 120},
			"UpdateRemoteControl":     {Rate: time.Minute, Limit: 180},
			"UpdateRemoteControlV2":   {Bucket: "UpdateRemoteControl", Rate: time.Minute, Limit: 180},
			"DeleteRemoteControl":     {Rate: time.Minute, Limit: 10},
			"PairRemoteControl": {
				Rate:    10 * time.Minute,
				Limit:   5,
				IPLimit: 10,
			},
			"PairRemoteControlV2": {
				Bucket:  "PairRemoteControl",
				Rate:    10 * time.Minute,
				Limit:   5,
				IPLimit: 10,
			},
			"RemoteEvents":       {Rate: time.Minute, Limit: 30},
			"RemoteEventsV2":     {Bucket: "RemoteEvents", Rate: time.Minute, Limit: 30},
			"AdminLogin":         {Rate: 10 * time.Minute, Limit: 5},
			"AdminSession":       {Rate: time.Minute, Limit: 120},
			"AdminLogout":        {Rate: time.Minute, Limit: 20},
			"AdminUsers":         {Rate: time.Minute, Limit: 60},
			"AdminCreateUser":    {Rate: 10 * time.Minute, Limit: 10},
			"AdminUpdateUser":    {Rate: 10 * time.Minute, Limit: 10},
			"AdminDeleteUser":    {Rate: 10 * time.Minute, Limit: 10},
			"AdminRooms":         {Rate: time.Minute, Limit: 120},
			"AdminRoomsV2":       {Bucket: "AdminRooms", Rate: time.Minute, Limit: 120},
			"AdminMessageUsage":  {Rate: time.Minute, Limit: 120},
			"AdminSearchUsage":   {Rate: time.Minute, Limit: 120},
			"AdminListenerUsage": {Rate: time.Minute, Limit: 120},
			"AdminUpdateRoom":    {Rate: time.Minute, Limit: 30},
			"AdminUpdateRoomV2":  {Bucket: "AdminUpdateRoom", Rate: time.Minute, Limit: 30},
			"AdminDeleteRoom":    {Rate: time.Minute, Limit: 30},
			"AdminDeleteRoomV2":  {Bucket: "AdminDeleteRoom", Rate: time.Minute, Limit: 30},
			"AdminEvents":        {Rate: time.Minute, Limit: 20},
			"AdminEventsV2":      {Bucket: "AdminEvents", Rate: time.Minute, Limit: 20},
		},
	}

	for _, r := range routers {
		r.Use(rm.Middleware)
	}
}

func (s *Server) setupInternalRoutes() {
	s.InternalRouter.HandleFunc("/_healthz", handler.Healthz).Methods(http.MethodGet).Name("Health")
	s.InternalRouter.Handle("/metrics", promhttp.Handler()).Name("Metrics")
}

func (s *Server) addSessionMiddleware(routers ...*mux.Router) {
	sm := middleware.SessionMiddleware{
		CastRouteNames: map[string]bool{
			"ReportPlaybackFailureV2": true,
			"RoomEventsV3":            true,
			"GetPlaybackStateV2":      true,
			"GetRoomV2":               true,
			"GetProviders":            true,
			"GetRoom":                 true,
			"GetSongs":                true,
			"GetPlaylistItems":        true,
			"GetPlaybackState":        true,
			"ReportPlaybackFailure":   true,
			"RoomEvents":              true,
			"RoomEventsV2":            true,
		},
		RemoteRouteNames: map[string]bool{
			"AddPlaylistV2":               true,
			"RoomEventsV3":                true,
			"SkipPlaylistItem":            true,
			"GetPlaybackStateV2":          true,
			"UpdatePlaybackStateV2":       true,
			"AddPlaylistItem":             true,
			"RemovePlaylistItem":          true,
			"VotePlaylistItem":            true,
			"GetRoomV2":                   true,
			"UpdateRoomSettingsV2":        true,
			"GetRemoteControl":            true,
			"GetRemoteControlV2":          true,
			"UpdateRemoteControl":         true,
			"UpdateRemoteControlV2":       true,
			"RemoteEvents":                true,
			"RemoteEventsV2":              true,
			"GetRoom":                     true,
			"UpdateRoomSettings":          true,
			"SkipSong":                    true,
			"GetPlaybackState":            true,
			"UpdatePlaybackState":         true,
			"CreateSession":               true,
			"CreateSessionV2":             true,
			"GetSongs":                    true,
			"GetPlaylistItems":            true,
			"AddSong":                     true,
			"AddPlaylist":                 true,
			"RemoveSong":                  true,
			"VoteSong":                    true,
			"RoomEvents":                  true,
			"SearchMusic":                 true,
			"SearchYouTubeV2":             true,
			"SearchSoundCloudV2":          true,
			"GetMusicTrack":               true,
			"GetYouTubeItemV2":            true,
			"GetYouTubePlaylist":          true,
			"GetYouTubePlaylistV2":        true,
			"SearchSoundCloud":            true,
			"ResolveSoundCloudTrack":      true,
			"ResolveSoundCloudItemV2":     true,
			"GetSoundCloudTrack":          true,
			"GetSoundCloudItemV2":         true,
			"ResolveSoundCloudPlaylist":   true,
			"ResolveSoundCloudPlaylistV2": true,
			"GetProviders":                true,
		},
		RemoteRoomRouteNames: map[string]bool{
			"AddPlaylistV2":         true,
			"RoomEventsV3":          true,
			"SkipPlaylistItem":      true,
			"GetPlaybackStateV2":    true,
			"UpdatePlaybackStateV2": true,
			"AddPlaylistItem":       true,
			"RemovePlaylistItem":    true,
			"VotePlaylistItem":      true,
			"GetRoomV2":             true,
			"UpdateRoomSettingsV2":  true,
			"SearchYouTubeV2":       true,
			"SearchSoundCloudV2":    true,
			"GetRoom":               true,
			"UpdateRoomSettings":    true,
			"SkipSong":              true,
			"GetPlaybackState":      true,
			"UpdatePlaybackState":   true,
			"CreateSession":         true,
			"CreateSessionV2":       true,
			"GetSongs":              true,
			"GetPlaylistItems":      true,
			"AddSong":               true,
			"AddPlaylist":           true,
			"RemoveSong":            true,
			"VoteSong":              true,
			"RoomEvents":            true,
		},
		Secret:                     s.Config.CookieSecret,
		CookieMaxAge:               s.Config.SessionCookieMaxAge,
		CastTokenSecret:            s.Config.CastTokenSecret,
		EmbedBasePath:              s.Config.EmbedBasePath,
		RemoteControlAuthenticator: s.DB,
	}
	for _, r := range routers {
		r.Use(sm.Middleware)
	}
}

// addTracingAndMetrics - Adds tracing and metrics to a router.
func (s *Server) addTracingAndMetrics(routers ...*mux.Router) {
	tm := middleware.TraceMiddleware{
		ExemptRoutes: map[string]bool{
			"RoomEventsV3":   true,
			"RoomEvents":     true,
			"RoomEventsV2":   true,
			"Messages":       true,
			"RemoteEvents":   true,
			"RemoteEventsV2": true,
			"AdminEvents":    true,
			"AdminEventsV2":  true,
		},
	}
	for _, r := range routers {
		r.Use(tm.Middleware)
		r.Use(middleware.MetricsMiddleware)
	}
}

func (s *Server) addCORSMiddleware(routers ...*mux.Router) {
	cm := &middleware.CORSMiddleware{
		AllowedOriginsCSV: s.Config.CORSAllowedOrigins,
	}
	for _, r := range routers {
		r.Use(cm.Middleware)
	}
}

func (s *Server) addPermissionMiddleware(routers ...*mux.Router) {
	am := middleware.PermissionMiddleware{
		DB: s.DB,
		ProtectedRoutes: map[string]bool{
			"CreateRoomGeneration": true,
			"UpdateRoomSettings":   true,
			"UpdateRoomSettingsV2": true,
		},
	}

	for _, r := range routers {
		r.Use(am.Middleware)
	}
}

func (s *Server) addAdminMiddleware(routers ...*mux.Router) {
	am := middleware.AdminMiddleware{
		DB:           s.DB,
		CookieSecret: s.Config.CookieSecret,
		ProtectedRoutes: map[string]bool{
			"AdminSession":       true,
			"AdminDocumentation": true,
			"AdminUsers":         true,
			"AdminCreateUser":    true,
			"AdminUpdateUser":    true,
			"AdminDeleteUser":    true,
			"AdminRooms":         true,
			"AdminRoomsV2":       true,
			"AdminMessageUsage":  true,
			"AdminSearchUsage":   true,
			"AdminListenerUsage": true,
			"AdminUpdateRoom":    true,
			"AdminUpdateRoomV2":  true,
			"AdminDeleteRoom":    true,
			"AdminDeleteRoomV2":  true,
			"AdminEvents":        true,
			"AdminEventsV2":      true,
		},
	}

	for _, r := range routers {
		r.Use(am.Middleware)
	}
}

const v1API string = "/api/v1"
const v2API string = "/api/v2"
const v3API string = "/api/v3"
const swaggerAPI string = "/api/swagger"
