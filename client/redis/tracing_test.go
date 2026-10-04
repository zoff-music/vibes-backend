package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestStreamReadTraceBoundary(t *testing.T) {
	tests := []struct {
		name   string
		cancel bool
	}{
		{name: "finished subscription parent"},
		{name: "cancelled subscription", cancel: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			previous := otel.GetTracerProvider()
			otel.SetTracerProvider(provider)
			t.Cleanup(func() {
				otel.SetTracerProvider(previous)
				err := provider.Shutdown(context.Background())
				if err != nil {
					t.Error(err)
				}
			})

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			ctx, parent := provider.Tracer("fixture").Start(ctx, "SubscribeFrom")
			parent.End()
			deadline, _ := ctx.Deadline()
			if tt.cancel {
				cancel()
			}

			unavailable := errors.New("synthetic redis unavailable")
			pool := &redigo.Pool{DialContext: func(dialCtx context.Context) (redigo.Conn, error) {
				actualDeadline, ok := dialCtx.Deadline()
				if !ok || !actualDeadline.Equal(deadline) {
					t.Error("read lost subscription deadline")
				}
				if tt.cancel && !errors.Is(dialCtx.Err(), context.Canceled) {
					t.Error("read lost subscription cancellation")
				}
				if dialCtx.Err() != nil {
					return nil, dialCtx.Err()
				}
				return nil, unavailable
			}}
			defer pool.Close()
			subscription := &streamSubscription{pool: pool}
			for range 2 {
				_, err := subscription.read(ctx, "0-0")
				if tt.cancel {
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("cancellation lost: %v", err)
					}
				} else if !errors.Is(err, unavailable) {
					t.Fatalf("unexpected read error: %v", err)
				}
			}
			spans := recorder.Ended()
			if len(spans) != 3 {
				t.Fatalf("got %d spans, want parent and two reads", len(spans))
			}
			for _, span := range spans[1:] {
				if span.Name() != "readEventStream" || span.Parent().IsValid() {
					t.Error("read is not an independent root span")
				}
				if span.SpanContext().TraceID() == parent.SpanContext().TraceID() {
					t.Error("read reused subscription trace")
				}
			}
			if spans[1].SpanContext().TraceID() == spans[2].SpanContext().TraceID() {
				t.Error("successive reads reused a trace")
			}
		})
	}
}
