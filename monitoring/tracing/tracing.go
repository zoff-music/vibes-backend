package tracing

import (
	"context"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
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

	r := resource.NewWithAttributes(semconv.SchemaURL, resourceAttributes(conf)...)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(
			exporter,
			sdktrace.WithBatchTimeout(batchInterval),
			sdktrace.WithMaxExportBatchSize(batchSize),
		),
		sdktrace.WithResource(r),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplerParam(conf.OtelSamplerParam)))),
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

func resourceAttributes(conf *config.Config) []attribute.KeyValue {
	serviceName := conf.OtelServiceName
	if serviceName == "" {
		serviceName = defaultServiceName
	}

	attributes := []attribute.KeyValue{
		semconv.ServiceName(serviceName),
	}

	for _, pair := range strings.Split(conf.OtelResourceAttrs, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}

		attributes = append(attributes, resourceAttribute(key, value))
	}

	return attributes
}

func resourceAttribute(key string, value string) attribute.KeyValue {
	boolValue, err := strconv.ParseBool(value)
	if err == nil {
		return attribute.Bool(key, boolValue)
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err == nil {
		return attribute.Int64(key, intValue)
	}

	floatValue, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return attribute.Float64(key, floatValue)
	}

	return attribute.String(key, value)
}

func samplerParam(param float64) float64 {
	if param < 0 {
		return 0
	}
	if param > 1 {
		return 1
	}
	return param
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

const instrumentationName = "github.com/zoff-music/vibes-backend"

const defaultServiceName = "vibes-backend"
