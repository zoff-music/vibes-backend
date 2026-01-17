// Package middleware provides HTTP middleware.
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/monitoring/metrics"

	"github.com/zoff-music/vibes/monitoring/opentracing"
)

// TraceMetrics is the configuration for trace and metrics middleware.
type TraceMetrics struct{}

// TraceMiddleware handles tracing of our HTTPS requests.
func (tm *TraceMetrics) TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeName := mux.CurrentRoute(r).GetName()

		span, ctx := initRootSpan(r, routeName)
		defer span.Finish()

		next.ServeHTTP(w, r.WithContext(ctx))

		span.SetTag("path", r.RequestURI)
	})
}

// MetricsMiddleware collects HTTP request metrics for Prometheus.
// Collects request duration and response code.
func (tm *TraceMetrics) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeName := mux.CurrentRoute(r).GetName()

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

func initRootSpan(r *http.Request, operationName string) (opentracing.Span, context.Context) {
	wireContext, err := opentracing.GlobalTracer().Extract(
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(r.Header),
	)
	if err != nil {
		return opentracing.StartSpanFromContext(r.Context(), operationName)
	}

	return opentracing.StartSpanFromContext(r.Context(), operationName, opentracing.ChildOf(wireContext))
}
