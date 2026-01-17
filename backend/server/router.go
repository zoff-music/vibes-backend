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
	// Add CORS middleware to handle cross-origin requests
	s.Router.Use(middleware.CORSMiddleware)

	s.Router.Handle("/metrics", promhttp.Handler()).Name("Metrics")
	s.Router.HandleFunc("/_healthz", handler.Healthz).Methods(http.MethodGet).Name("Health")

	api := s.Router.PathPrefix(v1API).Subrouter()

	// Room routes
	api.HandleFunc("/rooms", handler.CreateRoom(s.DB)).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/rooms/{id}", handler.GetRoom(s.DB)).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/rooms/{id}", handler.UpdateRoom(s.DB)).Methods(http.MethodPatch, http.MethodOptions)
	api.HandleFunc("/rooms/{id}/sessions", handler.CreateSession(s.DB)).Methods(http.MethodPost, http.MethodOptions)

	// Song routes
	api.HandleFunc("/rooms/{id}/songs", handler.GetSongs(s.DB)).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/rooms/{id}/songs", handler.AddSong(s.DB)).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/rooms/{id}/songs/{songId}", handler.RemoveSong(s.DB)).Methods(http.MethodDelete, http.MethodOptions)
	api.HandleFunc("/rooms/{id}/songs/{songId}", handler.ReorderSongs(s.DB)).Methods(http.MethodPatch, http.MethodOptions)

	// Action route
	api.HandleFunc("/rooms/{id}/action", handler.RoomAction(s.DB, s.InternalPubSub)).Methods(http.MethodPost, http.MethodOptions)

	// SSE route
	api.HandleFunc("/rooms/{id}/events", handler.RoomEvents(s.InternalPubSub, s.DB)).Methods(http.MethodGet, http.MethodOptions)

	// YouTube routes
	// YouTube routes
	api.HandleFunc("/youtube/search", handler.SearchMusic(s.YouTube)).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/youtube/videos/{id}", handler.GetMusicTrack(s.YouTube)).Methods(http.MethodGet, http.MethodOptions)

	s.addTracingAndMetrics(api)
}

// addTracingAndMetrics - Adds tracing and metrics to a router.
func (s *Server) addTracingAndMetrics(routers ...*mux.Router) {
	tm := middleware.TraceMetrics{}

	for _, r := range routers {
		r.Use(tm.TraceMiddleware)
		r.Use(tm.MetricsMiddleware)
	}
}
