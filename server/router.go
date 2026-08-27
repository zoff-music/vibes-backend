package server

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/zoff-music/vibes-backend/server/internal/handler"
	"github.com/zoff-music/vibes-backend/server/internal/middleware"
	_ "github.com/zoff-music/vibes-backend/swaggerdocs"
	"github.com/zoff-music/vibes-backend/vibe"
)

// setupRoutes - the root route function.
func (s *Server) setupRoutes() {
	api := s.Router.PathPrefix(v1API).Subrouter()
	s.Router.Handle(swaggerAPI, http.RedirectHandler(swaggerAPI+"/", http.StatusPermanentRedirect)).Methods(http.MethodGet)
	s.Router.PathPrefix(swaggerAPI + "/").Handler(httpSwagger.WrapHandler)

	// Room routes
	api.HandleFunc("/rooms", handler.CreateRoom(s.DB)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRoom")
	api.HandleFunc("/rooms/reservations", handler.ReserveRoomName(s.DB)).Methods(http.MethodPost, http.MethodOptions).Name("ReserveRoomName")
	api.HandleFunc("/rooms/suggestions", handler.SuggestRoomName(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("SuggestRoomName")
	api.HandleFunc("/rooms/public", handler.GetPublicRooms(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetPublicRooms")
	api.HandleFunc("/rooms/generation", handler.CreateGeneratedRoom(s.DB)).Methods(http.MethodPost, http.MethodOptions).Name("CreateGeneratedRoom")
	api.HandleFunc("/rooms/{id}/generations", handler.CreateRoomGeneration(s.DB, s.Config.RoomGenerationMaxExistingSongs)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRoomGeneration")
	api.HandleFunc("/rooms/{id}", handler.RoomExists(s.DB)).Methods(http.MethodHead).Name("RoomExists")
	api.HandleFunc("/rooms/{id}", handler.GetRoom(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetRoom")
	api.HandleFunc("/rooms/{id}/settings", handler.UpdateRoomSettings(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateRoomSettings")
	api.HandleFunc("/rooms/{id}/skips", handler.SkipSong(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("SkipSong")
	api.HandleFunc("/rooms/{id}/states", handler.GetPlaybackState(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetPlaybackState")
	api.HandleFunc("/rooms/{id}/states", handler.UpdatePlaybackState(s.DB, s.Redis, s.Redis)).Methods(http.MethodPut, http.MethodOptions).Name("UpdatePlaybackState")
	api.HandleFunc("/rooms/{id}/playbackfailures", handler.ReportPlaybackFailure(s.DB, s.YouTube, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("ReportPlaybackFailure")
	api.HandleFunc("/rooms/{id}/sessions", handler.CreateSession(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("CreateSession")
	api.HandleFunc("/rooms/{id}/sessions", handler.DeleteRoomAdminSession(s.DB)).Methods(http.MethodDelete, http.MethodOptions).Name("DeleteRoomAdminSession")
	api.HandleFunc("/sessions", handler.GetSessionProfile(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetSessionProfile")
	api.HandleFunc("/sessions", handler.UpdateSessionProfile(s.DB)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateSessionProfile")

	// Song routes
	api.HandleFunc("/rooms/{id}/songs", handler.GetSongs(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetSongs")
	api.HandleFunc("/rooms/{id}/songs", handler.AddSong(s.DB, s.Redis, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("AddSong")
	api.HandleFunc("/rooms/{id}/playlists", handler.AddPlaylist(s.DB, s.Redis, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("AddPlaylist")
	api.HandleFunc("/rooms/{id}/songs/{songId}", handler.RemoveSong(s.DB, s.Redis)).Methods(http.MethodDelete, http.MethodOptions).Name("RemoveSong")
	api.HandleFunc("/rooms/{id}/songs/{songId}", handler.VoteSong(s.DB, s.Redis)).Methods(http.MethodPost, http.MethodOptions).Name("VoteSong")

	// SSE route
	api.HandleFunc("/rooms/{id}/events", handler.RoomEvents(s.Redis, s.DB, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("RoomEvents")

	// YouTube routes
	api.HandleFunc("/youtube/search", handler.SearchMusic(s.YouTube, s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("SearchMusic")
	api.HandleFunc("/youtube/videos/{id}", handler.GetMusicTrack(s.YouTube, s.Redis, vibe.SourceTypeYouTube)).Methods(http.MethodGet, http.MethodOptions).Name("GetMusicTrack")
	api.HandleFunc("/youtube/playlists/{id}", handler.GetMusicPlaylist(s.YouTube, s.Redis, vibe.SourceTypeYouTube)).Methods(http.MethodGet, http.MethodOptions).Name("GetYouTubePlaylist")

	// SoundCloud routes
	api.HandleFunc("/soundcloud/search", handler.SearchSoundCloud(s.SoundCloud, s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("SearchSoundCloud")
	api.HandleFunc("/soundcloud/tracks", handler.ResolveSoundCloudTrack(s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("ResolveSoundCloudTrack")
	api.HandleFunc("/soundcloud/tracks/{id}", handler.GetSoundCloudTrack(s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("GetSoundCloudTrack")
	api.HandleFunc("/soundcloud/playlists", handler.ResolveSoundCloudPlaylist(s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("ResolveSoundCloudPlaylist")

	api.HandleFunc("/tokens/soundcloud", handler.GetToken(s.DB, s.SoundCloud, "soundcloud")).Methods(http.MethodGet, http.MethodOptions).Name("GetSoundCloudToken")
	api.HandleFunc("/tokens/youtube", handler.GetToken(s.DB, s.YouTube, "youtube")).Methods(http.MethodGet, http.MethodOptions).Name("GetYouTubeToken")

	// Authorization routes
	api.HandleFunc("/authorizations/soundcloud", handler.Authorize(s.DB, s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("Authorize")
	api.HandleFunc("/authorizations/youtube", handler.Authorize(s.DB, s.YouTube)).Methods(http.MethodGet, http.MethodOptions).Name("Authorize")
	// Callbacks
	api.HandleFunc("/callbacks/soundcloud", handler.OAuthCallback(s.DB, s.SoundCloud, "soundcloud")).Methods(http.MethodGet, http.MethodOptions).Name("SoundCloudCallback")
	api.HandleFunc("/callbacks/youtube", handler.OAuthCallback(s.DB, s.YouTube, "youtube")).Methods(http.MethodGet, http.MethodOptions).Name("YouTubeCallback")

	// Config routes
	api.HandleFunc("/providers", handler.GetProviders(s.Config)).Methods(http.MethodGet, http.MethodOptions).Name("GetProviders")

	// Stats routes
	api.HandleFunc("/stats", handler.GetStats(s.DB, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("GetStats")

	// Cast token endpoint (cookie-auth only)
	api.HandleFunc("/tokens/casting", handler.CreateCastingToken(s.DB, s.Config.CastTokenSecret)).Methods(http.MethodPost, http.MethodOptions).Name("CreateCastingToken")

	// Remote control routes
	api.HandleFunc("/remotes", handler.CreateRemoteControl(s.DB, s.Redis, s.Config.CookieSecret, s.Config.RemotePairingTTL)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRemoteControl")
	api.HandleFunc("/remotes", handler.GetOwnedRemoteControl(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetOwnedRemoteControl")
	api.HandleFunc("/remotes/{id}", handler.GetRemoteControl(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetRemoteControl")
	api.HandleFunc("/remotes/{id}", handler.UpdateRemoteControl(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateRemoteControl")
	api.HandleFunc("/remotes/{id}", handler.DeleteRemoteControl(s.DB)).Methods(http.MethodDelete, http.MethodOptions).Name("DeleteRemoteControl")
	api.HandleFunc("/remotes/{id}/sessions", handler.PairRemoteControl(s.DB, s.Redis, s.Config.CookieSecret)).Methods(http.MethodPost, http.MethodOptions).Name("PairRemoteControl")
	api.HandleFunc("/remotes/{id}/events", handler.RemoteEvents(s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("RemoteEvents")

	// Admin routes
	if s.Config.AdminPasswordPepper != "" {
		api.HandleFunc("/admin/sessions", handler.AdminLogin(s.DB, s.Config.AdminPasswordPepper, s.Config.CookieSecret)).Methods(http.MethodPost, http.MethodOptions).Name("AdminLogin")
		api.HandleFunc("/admin/sessions", handler.AdminSession()).Methods(http.MethodGet, http.MethodOptions).Name("AdminSession")
		api.HandleFunc("/admin/sessions", handler.AdminLogout()).Methods(http.MethodDelete, http.MethodOptions).Name("AdminLogout")
		api.HandleFunc("/admin/users", handler.AdminUsers(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminUsers")
		api.HandleFunc("/admin/users", handler.AdminCreateUser(s.DB, s.Config.AdminPasswordPepper)).Methods(http.MethodPost, http.MethodOptions).Name("AdminCreateUser")
		api.HandleFunc("/admin/users/{id}", handler.AdminUpdateUser(s.DB, s.Config.AdminPasswordPepper)).Methods(http.MethodPatch, http.MethodOptions).Name("AdminUpdateUser")
		api.HandleFunc("/admin/users/{id}", handler.AdminDeleteUser(s.DB)).Methods(http.MethodDelete, http.MethodOptions).Name("AdminDeleteUser")
		api.HandleFunc("/admin/rooms", handler.AdminRooms(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminRooms")
		api.HandleFunc("/admin/searches/usage", handler.AdminSearchUsage(s.DB, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("AdminSearchUsage")
		api.HandleFunc("/admin/listeners/usage", handler.AdminListenerUsage(s.DB, s.Redis)).Methods(http.MethodGet, http.MethodOptions).Name("AdminListenerUsage")
		api.HandleFunc("/admin/rooms/{id}", handler.AdminUpdateRoom(s.DB, s.Redis)).Methods(http.MethodPatch, http.MethodOptions).Name("AdminUpdateRoom")
		api.HandleFunc("/admin/rooms/{id}", handler.AdminDeleteRoom(s.DB, s.Redis)).Methods(http.MethodDelete, http.MethodOptions).Name("AdminDeleteRoom")
		api.HandleFunc("/admin/events", handler.AdminEvents(s.Redis, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminEvents")
	}

	s.addSessionMiddleware(api)
	if s.Config.RateLimitEnabled {
		s.addRateLimitMiddleware(api)
	}
	s.addPermissionMiddleware(api)
	if s.Config.AdminPasswordPepper != "" {
		s.addAdminMiddleware(api)
	}
	s.addTracingAndMetrics(api)
	s.addCORSMiddleware(s.Router)
}

func (s *Server) addRateLimitMiddleware(routers ...*mux.Router) {
	rm := middleware.RateLimitMiddleware{
		Checker: s.Redis,
		Policies: map[string]vibe.RateLimitPolicy{
			"CreateRoom": {Rate: time.Minute, Limit: 10},
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
			"RoomExists":            {Rate: time.Minute, Limit: 60},
			"GetRoom":               {Rate: time.Minute, Limit: 120},
			"GetPublicRooms":        {Rate: time.Minute, Limit: 120},
			"UpdateRoomSettings":    {Rate: time.Minute, Limit: 30},
			"SkipSong":              {Rate: time.Minute, Limit: 60},
			"GetPlaybackState":      {Rate: time.Minute, Limit: 240},
			"UpdatePlaybackState":   {Rate: time.Minute, Limit: 240},
			"ReportPlaybackFailure": {Rate: time.Minute, Limit: 30},
			"CreateSession":         {Rate: time.Minute, Limit: 30},
			"GetSongs":              {Rate: time.Minute, Limit: 120},
			"AddSong":               {Rate: time.Minute, Limit: 60},
			"AddPlaylist":           {Rate: time.Minute, Limit: 6},
			"RemoveSong":            {Rate: time.Minute, Limit: 60},
			"VoteSong":              {Rate: time.Minute, Limit: 120},
			"RoomEvents":            {Rate: time.Minute, Limit: 30},
			"SearchMusic": {
				Bucket:  "YouTubeSearch",
				Rate:    time.Second,
				Limit:   1,
				IPLimit: 3,
			},
			"GetMusicTrack":      {Rate: time.Minute, Limit: 120},
			"GetYouTubePlaylist": {Rate: time.Minute, Limit: 10},
			"SearchSoundCloud": {
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
			"CreateRoomGeneration": {
				Bucket:      "RoomGeneration",
				Rate:        10 * time.Minute,
				Limit:       1,
				IPLimit:     2,
				GlobalRate:  time.Minute,
				GlobalLimit: 1,
			},
			"Authorize":             {Rate: 10 * time.Minute, Limit: 20},
			"SoundCloudCallback":    {Rate: 10 * time.Minute, Limit: 30},
			"YouTubeCallback":       {Rate: 10 * time.Minute, Limit: 30},
			"GetProviders":          {Rate: time.Minute, Limit: 120},
			"GetStats":              {Rate: time.Minute, Limit: 600},
			"CreateCastingToken":    {Rate: time.Minute, Limit: 30},
			"CreateRemoteControl":   {Rate: time.Minute, Limit: 10},
			"GetOwnedRemoteControl": {Rate: time.Minute, Limit: 120},
			"GetRemoteControl":      {Rate: time.Minute, Limit: 120},
			"UpdateRemoteControl":   {Rate: time.Minute, Limit: 180},
			"DeleteRemoteControl":   {Rate: time.Minute, Limit: 10},
			"PairRemoteControl": {
				Rate:    10 * time.Minute,
				Limit:   5,
				IPLimit: 10,
			},
			"RemoteEvents":       {Rate: time.Minute, Limit: 30},
			"AdminLogin":         {Rate: 10 * time.Minute, Limit: 5},
			"AdminSession":       {Rate: time.Minute, Limit: 120},
			"AdminLogout":        {Rate: time.Minute, Limit: 20},
			"AdminUsers":         {Rate: time.Minute, Limit: 60},
			"AdminCreateUser":    {Rate: 10 * time.Minute, Limit: 10},
			"AdminUpdateUser":    {Rate: 10 * time.Minute, Limit: 10},
			"AdminDeleteUser":    {Rate: 10 * time.Minute, Limit: 10},
			"AdminRooms":         {Rate: time.Minute, Limit: 120},
			"AdminSearchUsage":   {Rate: time.Minute, Limit: 120},
			"AdminListenerUsage": {Rate: time.Minute, Limit: 120},
			"AdminUpdateRoom":    {Rate: time.Minute, Limit: 30},
			"AdminDeleteRoom":    {Rate: time.Minute, Limit: 30},
			"AdminEvents":        {Rate: time.Minute, Limit: 20},
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
	for _, r := range routers {
		r.Use(middleware.TraceMiddleware)
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
			"AdminUsers":         true,
			"AdminCreateUser":    true,
			"AdminUpdateUser":    true,
			"AdminDeleteUser":    true,
			"AdminRooms":         true,
			"AdminSearchUsage":   true,
			"AdminListenerUsage": true,
			"AdminUpdateRoom":    true,
			"AdminDeleteRoom":    true,
			"AdminEvents":        true,
		},
	}

	for _, r := range routers {
		r.Use(am.Middleware)
	}
}

const v1API string = "/api/v1"
const swaggerAPI string = "/api/swagger"
