package tracing

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/zoff-music/vibes-backend/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/credentials/insecure"
)

const instrumentationName = "github.com/zoff-music/vibes-backend"

const serviceName = "vibes"

type closer struct {
	provider *sdktrace.TracerProvider
	timeout  time.Duration
}

func Init(conf *config.Config) (io.Closer, error) {
	exporterTimeout := conf.OtelExporterTimeout
	if exporterTimeout <= 0 {
		exporterTimeout = time.Second
	}

	batchInterval := conf.OtelBatchInterval
	if batchInterval <= 0 {
		batchInterval = tracesdkDefaultScheduleDelay()
	}

	batchSize := conf.OtelBatchSize
	if batchSize <= 0 {
		batchSize = sdktrace.DefaultMaxExportBatchSize
	}

	ctx, cancel := context.WithTimeout(context.Background(), exporterTimeout)
	defer cancel()

	exporter, err := newExporter(ctx, conf.OtelEndpoint, exporterTimeout)
	if err != nil {
		return nil, fmt.Errorf("error creating otel exporter: %w", err)
	}

	r := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(
			exporter,
			sdktrace.WithBatchTimeout(batchInterval),
			sdktrace.WithMaxExportBatchSize(batchSize),
		),
		sdktrace.WithResource(r),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(conf.OtelSamplerParam))),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return &closer{
		provider: provider,
		timeout:  exporterTimeout,
	}, nil
}

func StartSpanFromContext(ctx context.Context, name string, opts ...trace.SpanStartOption) (trace.Span, context.Context) {
	cctx, span := otel.Tracer(instrumentationName).Start(ctx, name, opts...)
	span.SetAttributes(
		attribute.String("code.function", name),
	)
	return span, cctx
}

func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

func (c *closer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	err := c.provider.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("error shutting down OpenTelemetry provider: %w", err)
	}
	return nil
}

func newExporter(ctx context.Context, endpoint string, timeout time.Duration) (sdktrace.SpanExporter, error) {
	if endpoint == "" {
		return noopSpanExporter{}, nil
	}

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithTimeout(timeout),
		otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating otel trace grpc exporter: %w", err)
	}

	return exporterWithLogging{
		exporter: exporter,
	}, nil
}

func tracesdkDefaultScheduleDelay() time.Duration {
	return sdktrace.DefaultScheduleDelay * time.Millisecond
}

type exporterWithLogging struct {
	exporter sdktrace.SpanExporter
}

func (e exporterWithLogging) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	err := e.exporter.ExportSpans(ctx, spans)
	if err != nil {
		log.Printf("error exporting %d spans: %v", len(spans), err)
		return err
	}
	return nil
}

func (e exporterWithLogging) Shutdown(ctx context.Context) error {
	return e.exporter.Shutdown(ctx)
}

type noopSpanExporter struct{}

func (noopSpanExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return nil
}

func (noopSpanExporter) Shutdown(context.Context) error {
	return nil
}
