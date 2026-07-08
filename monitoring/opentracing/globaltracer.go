package opentracing

import (
	"context"
	"fmt"
	"time"

	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/monitoring/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Initialize the global tracer with a no-op tracer provider, so that we can
// request the tracer before it has been initialized without having to
// handle errors
var globalTracer = &Tracer{
	OtelTracer: noop.Tracer{},
	provider:   noop.NewTracerProvider(),
	closer:     noopCloser{},
}

func StartGlobalTracer(ctx context.Context, cfg *config.Config, appName string) (*Tracer, error) {
	// Validate the GRPC request timeout setting
	exporterTimeout := cfg.OtelExporterTimeout
	if exporterTimeout <= 0 {
		exporterTimeout = 1 * time.Second // Default to 1 second if not set
	}

	// Validate the span batch interval setting
	batchInterval := cfg.OtelBatchInterval
	if batchInterval <= 0 {
		batchInterval = tracesdk.DefaultScheduleDelay * time.Millisecond // Default to 5 seconds if not set
	}

	// Validate the span batch size setting
	batchSize := cfg.OtelBatchSize
	if batchSize <= 0 {
		batchSize = tracesdk.DefaultMaxExportBatchSize // Default to 512 spans per batch if not set
	}

	exporter := &telemetry.ExporterWithLogging{
		Endpoint:   cfg.OtelEndpoint,
		LogSuccess: false, // Enable this to log successful exports and TLS handshakes
		Insecure:   true,  // Disable TLS, use plain-text messages
		Timeout:    exporterTimeout,
	}

	err := exporter.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating otel exporter: %w", err)
	}

	tracerProvider, err := telemetry.OtelTraceProvider(exporter, cfg.OtelSamplerParam, appName, batchInterval, batchSize)
	if err != nil {
		return nil, fmt.Errorf("error creating otel trace provider: %w", err)
	}

	otel.SetTracerProvider(tracerProvider) // Another global, but one we can mostly ignore
	tracer := &Tracer{
		OtelTracer:  otel.Tracer(appName),
		Propagators: []propagation.TextMapPropagator{propagation.TraceContext{}},
		provider:    tracerProvider,
		closer: &telemetry.OtelCloser{
			Ctx:      ctx,
			Provider: tracerProvider,
		},
	}

	RegisterGlobalTracer(tracer)

	return tracer, nil
}

func RegisterGlobalTracer(t *Tracer) {
	globalTracer = t
}

func GlobalTracer() *Tracer {
	return globalTracer
}

type noopCloser struct{}

func (n noopCloser) Close() error {
	return nil
}
