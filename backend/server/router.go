package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zoff-music/vibes/server/internal/handler"
	"github.com/zoff-music/vibes/server/internal/middleware"
)

const v1API string = "/api/v1"

// setupRoutes - the root route function.
func (s *Server) setupRoutes() {
	s.Router.Handle("/metrics", promhttp.Handler()).Name("Metrics")
	s.Router.HandleFunc("/_healthz", handler.Healthz).Methods(http.MethodGet).Name("Health")

	api := s.Router.PathPrefix(v1API).Subrouter()

	// Room routes
	api.HandleFunc("/rooms", handler.CreateRoom(s.DB)).Methods(http.MethodPost, http.MethodOptions).Name("CreateRoom")
	api.HandleFunc("/rooms/{id}", handler.GetRoom(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetRoom")
	api.HandleFunc("/rooms/{id}/settings", handler.UpdateRoomSettings(s.DB, s.InternalPubSub)).Methods(http.MethodPatch, http.MethodOptions).Name("UpdateRoomSettings")
	api.HandleFunc("/rooms/{id}/skips", handler.SkipSong(s.DB, s.InternalPubSub)).Methods(http.MethodPost, http.MethodOptions).Name("SkipSong")
	api.HandleFunc("/rooms/{id}/states", handler.UpdatePlaybackState(s.DB, s.InternalPubSub)).Methods(http.MethodPut, http.MethodOptions).Name("UpdatePlaybackState")
	api.HandleFunc("/rooms/{id}/sessions", handler.CreateSession(s.DB)).Methods(http.MethodPost, http.MethodOptions).Name("CreateSession")

	// Song routes
	api.HandleFunc("/rooms/{id}/songs", handler.GetSongs(s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("GetSongs")
	api.HandleFunc("/rooms/{id}/songs", handler.AddSong(s.DB, s.InternalPubSub)).Methods(http.MethodPost, http.MethodOptions).Name("AddSong")
	api.HandleFunc("/rooms/{id}/songs/{songId}", handler.RemoveSong(s.DB, s.InternalPubSub)).Methods(http.MethodDelete, http.MethodOptions).Name("RemoveSong")
	api.HandleFunc("/rooms/{id}/songs/{songId}", handler.VoteSong(s.DB, s.InternalPubSub)).Methods(http.MethodPost, http.MethodOptions).Name("VoteSong")
	api.HandleFunc("/rooms/{id}/songs/{songId}", handler.ReorderSongs(s.DB, s.InternalPubSub)).Methods(http.MethodPatch, http.MethodOptions).Name("ReorderSongs")

	// SSE route
	api.HandleFunc("/rooms/{id}/events", handler.RoomEvents(s.InternalPubSub, s.DB, s.DB)).Methods(http.MethodGet, http.MethodOptions).Name("RoomEvents")

	// YouTube routes
	api.HandleFunc("/youtube/search", handler.SearchMusic(s.YouTube)).Methods(http.MethodGet, http.MethodOptions).Name("SearchMusic")
	api.HandleFunc("/youtube/videos/{id}", handler.GetMusicTrack(s.YouTube)).Methods(http.MethodGet, http.MethodOptions).Name("GetMusicTrack")

	s.addSessionMiddleware(api)
	s.addAuthMiddleware(api)
	s.addTracingAndMetrics(api)
	s.addCORSMiddleware(s.Router)
}

func (s *Server) addSessionMiddleware(routers ...*mux.Router) {
	for _, r := range routers {
		r.Use(middleware.SessionMiddleware)
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
	for _, r := range routers {
		r.Use(middleware.CORSMiddleware)
	}
}

func (s *Server) addAuthMiddleware(routers ...*mux.Router) {
	am := middleware.AuthMiddleware{
		Provider: s.DB,
		ProtectedRoutes: map[string]bool{
			"UpdateRoomSettings": true,
		},
	}

	for _, r := range routers {
		r.Use(am.Middleware)
	}
}
