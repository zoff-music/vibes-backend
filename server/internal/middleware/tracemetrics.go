// Package middleware provides HTTP middleware.
package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/monitoring/metrics"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
)

// TraceMiddleware handles tracing of our HTTPS requests.
type TraceMiddleware struct {
	ExemptRoutes map[string]bool
}

func (m *TraceMiddleware) Middleware(next http.Handler) http.Handler {
	middleware := otelhttp.NewMiddleware(
		"http.server",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return routeName(r)
		}),
	)

	tracedHandler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := tracing.SpanFromContext(r.Context())
		span.SetAttributes(attribute.String("path", r.RequestURI))
		next.ServeHTTP(w, r)
	}))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.ExemptRoutes[routeName(r)] {
			next.ServeHTTP(w, r)
			return
		}

		tracedHandler.ServeHTTP(w, r)
	})
}

// MetricsMiddleware collects HTTP request metrics for Prometheus.
// Collects request duration and response code.
func MetricsMiddleware(next http.Handler) http.Handler {
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
	status      int
	wroteHeader bool
}

func (crw *customResponseWriter) WriteHeader(status int) {
	if crw.wroteHeader {
		return
	}

	if status >= 100 && status < 200 && status != http.StatusSwitchingProtocols {
		crw.ResponseWriter.WriteHeader(status)
		return
	}

	crw.wroteHeader = true
	crw.status = status
	crw.ResponseWriter.WriteHeader(status)
}

func (crw *customResponseWriter) Write(body []byte) (int, error) {
	if !crw.wroteHeader {
		crw.status = http.StatusOK
		crw.wroteHeader = true
	}

	written, err := crw.ResponseWriter.Write(body)
	if err != nil {
		return written, fmt.Errorf("error writing HTTP response: %w", err)
	}

	return written, nil
}

func (crw *customResponseWriter) Flush() {
	flusher, ok := crw.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}

	if !crw.wroteHeader {
		crw.WriteHeader(http.StatusOK)
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
