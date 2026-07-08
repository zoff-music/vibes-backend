package tracing_test

import (
	"context"
	"testing"
	"time"

	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
)

func TestStartSpanFromContextKeepsTraceContext(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "child span uses parent trace",
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

			parentSpan, ctx := tracing.StartSpanFromContext(context.Background(), "parent")
			defer parentSpan.End()

			childSpan, _ := tracing.StartSpanFromContext(ctx, "child")
			defer childSpan.End()

			parentTraceID := parentSpan.SpanContext().TraceID()
			childTraceID := childSpan.SpanContext().TraceID()
			if parentTraceID != childTraceID {
				t.Fatalf("expected child trace id %s to match parent trace id %s", childTraceID, parentTraceID)
			}
		})
	}
}
