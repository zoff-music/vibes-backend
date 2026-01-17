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
	api.HandleFunc("/example", handler.Example(s.API)).Methods(http.MethodGet).Name("Example")
	addTracingAndMetrics(api)
}

// addTracingAndMetrics - Adds tracing and metrics to a router.
func addTracingAndMetrics(r *mux.Router) {
	tm := middleware.TraceMetrics{}
	r.Use(tm.TraceMiddleware)
	r.Use(tm.MetricsMiddleware)
}
