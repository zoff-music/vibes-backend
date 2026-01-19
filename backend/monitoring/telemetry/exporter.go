package telemetry

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
		otlptracegrpc.WithDialOption(grpc.WithUnaryInterceptor(d.LoggingInterceptor)),
	}

	if d.Insecure {
		options = append(options, d.insecureCreds())
	}

	exporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return fmt.Errorf("otlptracegrpc.New: %w", err)
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

// LoggingInterceptor is a unary client interceptor for logging requests and responses.
func (d *ExporterWithLogging) LoggingInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	if err != nil {
		log.Printf("RPC error method=%s duration=%v error=%v", method, time.Since(start), err)
		return err
	}

	if d.LogSuccess {
		log.Printf("RPC call method=%s duration=%v", method, time.Since(start))
	}

	return nil
}

// insecureCreds returns a custom insecure credential option that logs the success or failure of a handshake.
func (d *ExporterWithLogging) insecureCreds() otlptracegrpc.Option {
	return otlptracegrpc.WithTLSCredentials(
		&loggingCreds{insecure.NewCredentials(), d.LogSuccess},
	)
}

// loggingCreds is a wrapper around a TransportCredentials value, which logs the success or failure of a handshake.
type loggingCreds struct {
	credentials.TransportCredentials
	logSuccess bool
}

func (c *loggingCreds) ClientHandshake(ctx context.Context, addr string, rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	conn, auth, err := c.TransportCredentials.ClientHandshake(ctx, addr, rawConn)
	if err != nil {
		log.Printf("Client handshake failed: %v", err)
	}
	if err == nil && c.logSuccess {
		log.Println("Client handshake succeeded")
	}
	return conn, auth, err
}
