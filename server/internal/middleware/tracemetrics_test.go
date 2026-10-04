package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTraceMiddlewareStreamBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		wantSpans int
	}{
		{name: "RoomEvents"},
		{name: "RoomEventsV2"},
		{name: "RemoteEvents"},
		{name: "AdminEvents"},
		{name: "GetRoom", wantSpans: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			previous := otel.GetTracerProvider()
			otel.SetTracerProvider(provider)
			t.Cleanup(func() { otel.SetTracerProvider(previous) })
			router := mux.NewRouter()
			router.Use(TraceMiddleware)
			router.HandleFunc("/fixture", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}).Name(tt.name)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/fixture", nil))
			if response.Code != http.StatusNoContent {
				t.Fatal("handler was not called")
			}
			if len(recorder.Ended()) != tt.wantSpans {
				t.Fatalf("got %d spans, want %d", len(recorder.Ended()), tt.wantSpans)
			}
		})
	}
}
