package trace

import (
	"context"
	"fmt"
	"io"

	"github.com/zoff-music/cibes/config"
	"github.com/zoff-music/cibes/monitoring/opentracing"
)

// InitGlobalTracer starts the global tracer, while conforming to the old interface
func InitGlobalTracer(conf *config.Config) (io.Closer, error) {
	// We lose the context here because the original interface does not have
	// a context parameter
	ctx := context.Background()
	tracer, err := opentracing.StartGlobalTracer(ctx, conf, "template-api-go")
	if err != nil {
		return nil, fmt.Errorf("failed to start global tracer: %w", err)
	}

	return tracer.GetCloser(), nil
}
