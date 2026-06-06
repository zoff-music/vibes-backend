package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zoff-music/vibes-backend/server/internal/handler"
	"github.com/zoff-music/vibes-backend/server/internal/middleware"
)

const v1API string = "/api/v1"

// setupRoutes - the root route function.
func (s *Server) setupRoutes() {
	api := s.Router.PathPrefix(v1API).Subrouter()

	// Room routes
	api.HandleFunc("/rooms", handler.CreateRoom(s.DB, s.InternalPubSub)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRoom")
	api.HandleFunc("/rooms/{id}", handler.GetRoom(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetRoom")
	api.HandleFunc("/rooms/{id}/settings", handler.UpdateRoomSettings(s.DB, s.InternalPubSub)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateRoomSettings")
	api.HandleFunc("/rooms/{id}/skips", handler.SkipSong(s.DB, s.InternalPubSub)).Methods(http.MethodPost, http.MethodOptions).Name("SkipSong")
	api.HandleFunc("/rooms/{id}/states", handler.GetPlaybackState(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetPlaybackState")
	api.HandleFunc("/rooms/{id}/states", handler.UpdatePlaybackState(s.DB, s.InternalPubSub)).Methods(http.MethodPut, http.MethodOptions).Name("UpdatePlaybackState")
	api.HandleFunc("/rooms/{id}/sessions", handler.CreateSession(s.DB, s.InternalPubSub)).Methods(http.MethodPost, http.MethodOptions).Name("CreateSession")

	// Song routes
	api.HandleFunc("/rooms/{id}/songs", handler.GetSongs(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetSongs")
	api.HandleFunc("/rooms/{id}/songs", handler.AddSong(s.DB, s.InternalPubSub)).Methods(http.MethodPost, http.MethodOptions).Name("AddSong")
	api.HandleFunc("/rooms/{id}/songs/{songId}", handler.RemoveSong(s.DB, s.InternalPubSub)).Methods(http.MethodDelete, http.MethodOptions).Name("RemoveSong")
	api.HandleFunc("/rooms/{id}/songs/{songId}", handler.VoteSong(s.DB, s.InternalPubSub)).Methods(http.MethodPost, http.MethodOptions).Name("VoteSong")

	// SSE route
	api.HandleFunc("/rooms/{id}/events", handler.RoomEvents(s.InternalPubSub, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("RoomEvents")

	// YouTube routes
	api.HandleFunc("/youtube/search", handler.SearchMusic(s.YouTube)).Methods(http.MethodGet, http.MethodOptions).Name("SearchMusic")
	api.HandleFunc("/youtube/videos/{id}", handler.GetMusicTrack(s.YouTube)).Methods(http.MethodGet, http.MethodOptions).Name("GetMusicTrack")

	// SoundCloud routes
	api.HandleFunc("/soundcloud/search", handler.SearchSoundCloud(s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("SearchSoundCloud")
	api.HandleFunc("/soundcloud/tracks/{id}", handler.GetSoundCloudTrack(s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("GetSoundCloudTrack")

	// Spotify search routes
	api.HandleFunc("/spotify/search", handler.SearchSpotify(s.Spotify)).Methods(http.MethodGet, http.MethodOptions).Name("SearchSpotify")
	api.HandleFunc("/spotify/tracks/{id}", handler.GetSpotifyTrack(s.Spotify)).Methods(http.MethodGet, http.MethodOptions).Name("GetSpotifyTrack")
	api.HandleFunc("/tokens/spotify", handler.GetToken(s.DB, s.Spotify, "spotify")).Methods(http.MethodGet, http.MethodOptions).Name("GetSpotifyToken")
	api.HandleFunc("/tokens/soundcloud", handler.GetToken(s.DB, s.SoundCloud, "soundcloud")).Methods(http.MethodGet, http.MethodOptions).Name("GetSoundCloudToken")
	api.HandleFunc("/tokens/youtube", handler.GetToken(s.DB, s.YouTube, "youtube")).Methods(http.MethodGet, http.MethodOptions).Name("GetYouTubeToken")

	// Authorization routes
	api.HandleFunc("/authorizations/spotify", handler.Authorize(s.DB, s.Spotify)).Methods(http.MethodGet, http.MethodOptions).Name("Authorize")
	api.HandleFunc("/authorizations/soundcloud", handler.Authorize(s.DB, s.SoundCloud)).Methods(http.MethodGet, http.MethodOptions).Name("Authorize")
	api.HandleFunc("/authorizations/youtube", handler.Authorize(s.DB, s.YouTube)).Methods(http.MethodGet, http.MethodOptions).Name("Authorize")
	// Callbacks
	api.HandleFunc("/callbacks/spotify", handler.OAuthCallback(s.DB, s.Spotify, "spotify")).Methods(http.MethodGet, http.MethodOptions).Name("SpotifyCallback")
	api.HandleFunc("/callbacks/soundcloud", handler.OAuthCallback(s.DB, s.SoundCloud, "soundcloud")).Methods(http.MethodGet, http.MethodOptions).Name("SoundCloudCallback")
	api.HandleFunc("/callbacks/youtube", handler.OAuthCallback(s.DB, s.YouTube, "youtube")).Methods(http.MethodGet, http.MethodOptions).Name("YouTubeCallback")

	// Config routes
	api.HandleFunc("/providers", handler.GetProviders(s.Config)).Methods(http.MethodGet, http.MethodOptions).Name("GetProviders")

	// Cast token endpoint (cookie-auth only)
	api.HandleFunc("/casting/tokens", handler.CreateCastingToken(s.DB, s.Config.CastTokenSecret)).Methods(http.MethodPost, http.MethodOptions).Name("CreateCastingToken")

	// Admin routes (disabled when ADMIN_PASSWORD is not configured)
	if s.Config.AdminPassword != "" {
		api.HandleFunc("/admin/sessions", handler.AdminLogin(&s.Config.AdminPassword, s.Config.CookieSecret)).Methods(http.MethodPost, http.MethodOptions).Name("AdminLogin")
		api.HandleFunc("/admin/sessions", handler.AdminLogout()).Methods(http.MethodDelete, http.MethodOptions).Name("AdminLogout")
		api.HandleFunc("/admin/rooms", handler.AdminRooms(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminRooms")
		api.HandleFunc("/admin/events", handler.AdminEvents(s.InternalPubSub, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("AdminEvents")
	}

	s.addSessionMiddleware(api)
	s.addPermissionMiddleware(api)
	if s.Config.AdminPassword != "" {
		s.addAdminMiddleware(api)
	}
	s.addTracingAndMetrics(api)
	s.addCORSMiddleware(s.Router)
}

func (s *Server) setupInternalRoutes() {
	s.InternalRouter.HandleFunc("/_healthz", handler.Healthz).Methods(http.MethodGet).Name("Health")
	s.InternalRouter.Handle("/metrics", promhttp.Handler()).Name("Metrics")
}

func (s *Server) addSessionMiddleware(routers ...*mux.Router) {
	sm := middleware.SessionMiddleware{
		Secret:          s.Config.CookieSecret,
		CastTokenSecret: s.Config.CastTokenSecret,
	}
	for _, r := range routers {
		r.Use(sm.Middleware)
	}
}

// addTracingAndMetrics - Adds tracing and metrics to a router.
func (s *Server) addTracingAndMetrics(routers ...*mux.Router) {
	tm := middleware.TraceMetrics{}

	for _, r := range routers {
		r.Use(tm.TraceMiddleware)
		r.Use(tm.MetricsMiddleware)
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
			"UpdateRoomSettings": true,
		},
	}

	for _, r := range routers {
		r.Use(am.Middleware)
	}
}

func (s *Server) addAdminMiddleware(routers ...*mux.Router) {
	am := middleware.AdminMiddleware{
		AdminPassword: &s.Config.AdminPassword,
		CookieSecret:  s.Config.CookieSecret,
		ProtectedRoutes: map[string]bool{
			"AdminRooms":  true,
			"AdminEvents": true,
		},
	}

	for _, r := range routers {
		r.Use(am.Middleware)
	}
}
