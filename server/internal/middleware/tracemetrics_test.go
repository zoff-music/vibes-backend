package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestTraceMiddlewareExtractsTraceparent(t *testing.T) {
	tests := []struct {
		name        string
		traceparent string
		traceID     string
	}{
		{
			name:        "request context uses incoming trace id",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			traceID:     "4bf92f3577b34da6a3ce929d0e0e4736",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				OtelSamplerParam:    1,
				OtelExporterTimeout: time.Second,
				OtelBatchInterval:   time.Millisecond,
				OtelBatchSize:       1,
			}

			closer, err := tracing.Init(&cfg)
			if err != nil {
				t.Fatalf("error initializing tracing: %v", err)
			}
			defer closer.Close()

			tm := TraceMetrics{}
			handler := tm.TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				spanContext := oteltrace.SpanContextFromContext(r.Context())
				if spanContext.TraceID().String() != tt.traceID {
					t.Fatalf("expected trace id %s, got %s", tt.traceID, spanContext.TraceID().String())
				}

				w.WriteHeader(http.StatusNoContent)
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/v1/rooms/electro", nil)
			req.Header.Set("traceparent", tt.traceparent)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusNoContent {
				t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
			}
		})
	}
}
