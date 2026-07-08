// Package handler contains HTTP handlers.
//
//	Routes:
//		GET /api/v1/example
//		GET /_healthz
package handler

import "net/http"

// Healthz is used for our readiness and liveness probes.
//
//	@Summary	Health check
//	@Tags		internal
//	@Produce	plain
//	@Success	200	{string}	string
//	@Router		/_healthz [get]
func Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(http.StatusText(http.StatusOK)))
}
