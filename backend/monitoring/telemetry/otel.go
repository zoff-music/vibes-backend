package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// OtelTraceProvider creates a new OpenTelemetry trace provider with the given
// exporter, samplerParam, application name, batch interval, and batch size.
func OtelTraceProvider(exp sdktrace.SpanExporter, samplerParam float64, appName string, batchInterval time.Duration, batchSize int) (*sdktrace.TracerProvider, error) {
	// Ensure default SDK resources and the required service name are set.
	r := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(appName),
	)

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(
			exp,
			sdktrace.WithBatchTimeout(batchInterval),
			sdktrace.WithMaxExportBatchSize(batchSize),
		),
		sdktrace.WithResource(r),
		// Inherit trace ID from the parent span if it exists.
		// If the parent span does not exist, generate a new trace ID for a
		// portion of the requests, based on the samplerParam.
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplerParam))),
	), nil
}

type OtelCloser struct {
	Ctx      context.Context
	Provider *sdktrace.TracerProvider
}

func (c *OtelCloser) Close() error {
	err := c.Provider.Shutdown(c.Ctx)
	if err != nil {
		return fmt.Errorf("error shutting down OpenTelemetry provider: %w", err)
	}
	return nil
}
