package trace

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/zoff-music/vibes/monitoring/opentracing"
)

// InjectIntoCarrier returns a textMapCarrier, basically a map[string]string,
// which can be used to transmit a span context to another service with ExtractFromCarrier.
func InjectIntoCarrier(ctx context.Context) opentracing.TextMapCarrier {
	carrier := opentracing.TextMapCarrier{}
	span := opentracing.SpanFromContext(ctx)

	if span == nil {
		return carrier
	}

	err := opentracing.GlobalTracer().Inject(span.Context(), opentracing.TextMap, carrier)
	if err != nil {
		log.WithError(err).Error("unexpected error while injecting span into carrier")
	}
	return carrier
}

// ExtractFromCarrier returns a span with context passed by the carrier.
// ctx should not already have span in it.
func ExtractFromCarrier(ctx context.Context, carrier opentracing.TextMapCarrier, spanName string) (opentracing.Span, context.Context) {
	tracer := opentracing.GlobalTracer()
	wireContext, err := tracer.Extract(opentracing.TextMap, carrier)
	if err != nil {
		// At the time of writing, the Extract method will always return nil on the error value
		log.WithError(err).Error("unexpected error while extracting span from carrier")
		return opentracing.SpanFromContext(ctx), ctx
	}

	// If there's no context, make one
	if ctx == nil {
		ctx = context.Background()
	}

	return tracer.StartFromContext(ctx, spanName, wireContext)
}
