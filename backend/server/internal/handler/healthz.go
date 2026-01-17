// Package handler contains HTTP handlers.
//
//	Routes:
// 		GET /api/v1/example
// 		GET /_healthz
package handler

import "net/http"

// Healthz is used for our readiness and liveness probes.
// 		GET /_healthz
// 		Responds: 200
func Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(http.StatusText(http.StatusOK)))
}
