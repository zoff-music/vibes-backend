package telemetry

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials/insecure"
)

type ExporterWithLogging struct {
	Endpoint   string        // An OTLP GRPC endpoint in the format "hostname:port"
	LogSuccess bool          // If true, successful export events will be logged
	Insecure   bool          // If true, TLS is disabled and messages are sent as plain-text
	Timeout    time.Duration // The maximum time to wait before sending a span batch

	traceExporter sdktrace.SpanExporter
}

func (d *ExporterWithLogging) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if d.traceExporter == nil {
		return nil
	}

	err := d.traceExporter.ExportSpans(ctx, spans)
	if err != nil {
		log.Printf("Failed to export %d spans: %v", len(spans), err)
		return err
	}

	if d.LogSuccess {
		log.Printf("Exported %d spans.", len(spans))
	}

	return nil
}

func (d *ExporterWithLogging) Shutdown(ctx context.Context) error {
	if d.traceExporter == nil {
		return nil
	}

	return d.traceExporter.Shutdown(ctx)
}

func (d *ExporterWithLogging) Init(ctx context.Context) error {
	if d.Timeout == 0 {
		d.Timeout = 100 * time.Millisecond
	}

	if d.Endpoint == "" {
		d.traceExporter = noopSpanExporter{}
		return nil
	}

	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(d.Endpoint),
		otlptracegrpc.WithTimeout(d.Timeout),
	}

	if d.Insecure {
		options = append(options, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()))
	}

	exporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return fmt.Errorf("error creating otel trace grpc exporter: %w", err)
	}

	d.traceExporter = exporter
	return nil
}

type noopSpanExporter struct{}

func (noopSpanExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return nil
}

func (noopSpanExporter) Shutdown(context.Context) error {
	return nil
}
