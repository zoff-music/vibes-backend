// Package middleware provides HTTP middleware.
package middleware

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/monitoring/metrics"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
)

// TraceMetrics is the configuration for trace and metrics middleware.
type TraceMetrics struct{}

// TraceMiddleware handles tracing of our HTTPS requests.
func (tm *TraceMetrics) TraceMiddleware(next http.Handler) http.Handler {
	middleware := otelhttp.NewMiddleware(
		"http.server",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return routeName(r)
		}),
	)

	return middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := tracing.SpanFromContext(r.Context())
		span.SetAttributes(attribute.String("path", r.RequestURI))
		next.ServeHTTP(w, r)
	}))
}

// MetricsMiddleware collects HTTP request metrics for Prometheus.
// Collects request duration and response code.
func (tm *TraceMetrics) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeName := routeName(r)

		crw := customResponseWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(&crw, r)

		duration := time.Since(start)

		metrics.ObserveTimeToProcess(routeName, duration.Seconds())
		metrics.ReceivedRequest(crw.status, routeName)
	})
}

type customResponseWriter struct {
	http.ResponseWriter
	status int
}

func (crw *customResponseWriter) WriteHeader(status int) {
	crw.status = status
	crw.ResponseWriter.WriteHeader(status)
}

func (crw *customResponseWriter) Flush() {
	flusher, ok := crw.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}

	flusher.Flush()
}

func routeName(r *http.Request) string {
	route := mux.CurrentRoute(r)
	if route == nil {
		return "http.server"
	}

	name := route.GetName()
	if name == "" {
		return "http.server"
	}

	return name
}
