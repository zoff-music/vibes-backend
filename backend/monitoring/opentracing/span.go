package opentracing

import (
	"context"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// The Span interface exposes the Span methods that are used outside the
// opentracing package. We're using an interface because the old opentracing
// package did, which affected the syntax of the implementation. In order to
// offer a drop-in replacement, we have to do the same.
type Span interface {
	Context() *SpanContext
	Finish()
	SetTag(key string, value any)
	LogKV(alternatingKeyValues ...any)
	CombineContexts(extraContexts ...*SpanContext)
}

// SpanFromContext is a drop-in replacement for opentracing.SpanFromContext
func SpanFromContext(ctx context.Context) Span {
	span := trace.SpanFromContext(ctx)

	return &OtSpan{
		otelSpan: span,
		context: &SpanContext{
			ctx:      ctx,
			traceCtx: span.SpanContext(),
		},
	}
}

// StartSpanFromContext is a drop-in replacement for opentracing.StartSpanFromContext
func StartSpanFromContext(ctx context.Context, name string, extraContexts ...*SpanContext) (Span, context.Context) {
	return globalTracer.StartFromContext(ctx, name, extraContexts...)
}

// OtSpan implements the pretend-opentracing Span interface.
type OtSpan struct {
	otelSpan trace.Span
	context  *SpanContext
	ended    atomic.Bool
}

func (s *OtSpan) Finish() {
	if s.ended.Swap(true) {
		return
	}
	s.otelSpan.End()
}

// Such a messy function, but we seem to be using it
func (s *OtSpan) LogKV(alternatingKeyValues ...any) {
	for i := 0; i < len(alternatingKeyValues); i += 2 {
		key := alternatingKeyValues[i].(string)
		value := alternatingKeyValues[i+1]
		s.SetTag(key, value)
	}
}

// Inefficient, but it saves us from having to change a lot of code
func (s *OtSpan) SetTag(key string, value any) {
	switch v := value.(type) {
	case string:
		s.SetStringTag(key, v)
	case bool:
		s.SetBoolTag(key, v)
	case error:
		s.SetStringTag(key, v.Error())
	default:
		log.Errorf("trace.SetTag: unsupported tag type: %T (value: %v)", v, v)
	}
}

func (s *OtSpan) SetStringTag(key string, value string) {
	s.otelSpan.SetAttributes(attribute.String(key, value))
}

func (s *OtSpan) SetBoolTag(key string, value bool) {
	s.otelSpan.SetAttributes(attribute.Bool(key, value))
}

func (s *OtSpan) Context() *SpanContext {
	return s.context
}

func (s *OtSpan) CombineContexts(extraContexts ...*SpanContext) {
	ctx := s.Context()
	for _, ectx := range extraContexts {
		ctx.Combine(ectx)
	}

	s.context = ctx
}

// ChildOf is a drop-in replacement for opentracing.ChildOf.
// opentelemetry has done away with the concept of span references, so we just
// return the original span
func ChildOf(sc *SpanContext) *SpanContext {
	return sc
}
