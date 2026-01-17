package opentracing

import "net/http"

// TextMapCarrier is a copy of go.opentelemetry.io/otel/propagation.MapCarrier
// source: https://github.com/open-telemetry/opentelemetry-go/blob/main/propagation/propagation.go
// It implements the propagation.TextMapCarrier interface.
type TextMapCarrier map[string]string

// Get returns the value associated with the passed key.
func (c TextMapCarrier) Get(key string) string {
	return c[key]
}

// Set stores the key-value pair.
func (c TextMapCarrier) Set(key, value string) {
	c[key] = value
}

// Keys lists the keys stored in this carrier.
func (c TextMapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// HTTPHeadersCarrier is a straight up copy of go.opentelemetry.io/otel/propagation.HeaderCarrier
// It implements the propagation.TextMapCarrier interface.
type HTTPHeadersCarrier http.Header

// Get returns the value associated with the passed key.
func (hc HTTPHeadersCarrier) Get(key string) string {
	return http.Header(hc).Get(key)
}

// Set stores the key-value pair.
func (hc HTTPHeadersCarrier) Set(key string, value string) {
	http.Header(hc).Set(key, value)
}

// Keys lists the keys stored in this carrier.
func (hc HTTPHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(hc))
	for k := range hc {
		keys = append(keys, k)
	}
	return keys
}

const (
	// These consts were used by OpenTracing to specify the format of the
	// carrier when injecting or extracting span contexts. It's unneccessary,
	// since we only need a TextMapCarrier implementation and don't need to add
	// special cases that vary based on these implementations.
	// It might not even have been necessary in the original OpenTracing package.
	// However, we need to keep these two around for compatibility just like
	// all the other OpenTracing interface cruft.
	HTTPHeaders = 1
	TextMap     = 2
)
