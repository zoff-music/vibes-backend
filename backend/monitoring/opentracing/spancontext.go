package opentracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type SpanContext struct {
	traceCtx trace.SpanContext
	ctx      context.Context
}

func NewSpanContext(ctx context.Context) *SpanContext {
	return &SpanContext{
		traceCtx: trace.SpanContextFromContext(ctx),
		ctx:      ctx,
	}
}

// Combine combines the current span context with another span context.
// Since a span is a struct that represents a single span, we can't mix their data.
// So in practice, this function simply chooses the "best" span out of the two.
func (sc *SpanContext) Combine(csc *SpanContext) {
	// If the current span context doesn't have a trace ID, but the combined
	// span context does, use the combined span context's trace ID.
	if !sc.traceCtx.HasTraceID() && csc.traceCtx.HasTraceID() {
		sc.traceCtx = csc.traceCtx
		return
	}

	// Prioritize remote spans over local spans.
	if !sc.traceCtx.IsRemote() && csc.traceCtx.IsRemote() {
		sc.traceCtx = csc.traceCtx
		return
	}
}

func (sc *SpanContext) TraceID() string {
	if !sc.traceCtx.HasTraceID() {
		return ""
	}

	return sc.traceCtx.TraceID().String()
}
