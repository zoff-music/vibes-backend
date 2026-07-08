package opentracing

import (
	"context"
	"log"
	"sync/atomic"

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
	LogFields(fields ...SpanField)
	SetBoolTag(key string, value bool)
	SetErrorTag(key string, value error)
	SetStringTag(key string, value string)
	CombineContexts(extraContexts ...*SpanContext)
}

type SpanFieldType string

const SpanFieldString SpanFieldType = "string"

const SpanFieldBool SpanFieldType = "bool"

const SpanFieldError SpanFieldType = "error"

type SpanField struct {
	Key         string
	StringValue string
	BoolValue   bool
	ErrorValue  error
	Type        SpanFieldType
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
	span, spanCtx := globalTracer.StartFromContext(ctx, name, extraContexts...)
	return span, spanCtx
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

func (s *OtSpan) LogFields(fields ...SpanField) {
	for _, field := range fields {
		switch field.Type {
		case SpanFieldString:
			s.SetStringTag(field.Key, field.StringValue)
		case SpanFieldBool:
			s.SetBoolTag(field.Key, field.BoolValue)
		case SpanFieldError:
			s.SetErrorTag(field.Key, field.ErrorValue)
		default:
			log.Printf("trace.LogFields: unsupported tag type: %s", field.Type)
		}
	}
}

func (s *OtSpan) SetStringTag(key string, value string) {
	s.otelSpan.SetAttributes(attribute.String(key, value))
}

func (s *OtSpan) SetBoolTag(key string, value bool) {
	s.otelSpan.SetAttributes(attribute.Bool(key, value))
}

func (s *OtSpan) SetErrorTag(key string, value error) {
	s.SetStringTag(key, value.Error())
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
