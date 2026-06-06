package opentracing

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"io"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Tracer pretends to be opentracing.Tracer, and fulfils the parts of the
// interface we need
type Tracer struct {
	OtelTracer  trace.Tracer
	Propagators []propagation.TextMapPropagator
	provider    trace.TracerProvider
	closer      io.Closer
}

// Start starts a new span with the given name and returns it along with the
// new context
func (t *Tracer) Start(ctx context.Context, name string) (Span, context.Context) {
	cctx, span := t.OtelTracer.Start(ctx, name)

	sc := span.SpanContext()
	// Put the IDs on every span so they’re searchable in your backend.
	span.SetAttributes(
		attribute.String("trace.id", sc.TraceID().String()),
		attribute.String("span.id", sc.SpanID().String()),
	)

	return &OtSpan{
		otelSpan: span,
		context: &SpanContext{
			traceCtx: sc,
			ctx:      cctx,
		},
	}, cctx
}

func (t *Tracer) StartFromContext(ctx context.Context, name string, extraContexts ...*SpanContext) (Span, context.Context) {
	pc := ctx

	var sc SpanContext
	for _, e := range extraContexts {
		if e == nil {
			continue
		}

		sc.Combine(e)
	}

	switch {
	case sc.ctx != nil:
		pc = sc.ctx
	case sc.traceCtx.IsValid():
		pc = trace.ContextWithSpanContext(pc, sc.traceCtx)
	}

	return t.Start(pc, name)
}

// HttpTransport returns an HTTP transport that instruments HTTP requests, by
// adding trace context to outgoing requests and extracting trace context from
// incoming requests.
// Not used right now, because we're sticking with the old opentracing interface.
func (t *Tracer) HttpTransport(rt http.RoundTripper) *otelhttp.Transport {
	p := t.compositePropagator()

	return otelhttp.NewTransport(rt,
		otelhttp.WithTracerProvider(t.provider),
		otelhttp.WithPropagators(p),
		otelhttp.WithFilter(func(*http.Request) bool { return false }),
	)
}

func (t *Tracer) GetCloser() io.Closer {
	return t.closer
}

// Extract gets the span context from the carrier (usually HTTP headers)
func (t *Tracer) Extract(format int, carrier propagation.TextMapCarrier) (*SpanContext, error) {
	// We lose the original context (and any associated spans) here because we
	// need to conform to the opentracing interface, which doesn't accept a
	// context as an argument
	ctx := t.compositePropagator().Extract(context.Background(), carrier)

	return &SpanContext{
		// Make a trace context, and mark it as remote so that the combiner
		// can give it priority over local span contexts
		traceCtx: trace.SpanContextFromContext(ctx).WithRemote(true),
		ctx:      ctx,
	}, nil
}

func (t *Tracer) compositePropagator() propagation.TextMapPropagator {
	if len(t.Propagators) == 1 {
		return t.Propagators[0]
	}
	if len(t.Propagators) > 1 {
		return propagation.NewCompositeTextMapPropagator(t.Propagators...)
	}

	return propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})
}

func (t *Tracer) Inject(sc *SpanContext, format int, carrier propagation.TextMapCarrier) error {
	ctx := sc.ctx
	if ctx == nil {
		ctx = trace.ContextWithSpanContext(context.Background(), sc.traceCtx)
	}
	t.compositePropagator().Inject(ctx, carrier)

	return nil
}
