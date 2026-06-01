package event

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/zoff-music/vibes/internalerror"
	"github.com/zoff-music/vibes/monitoring/opentracing"
)

// AppEvents contains a slice of AppEvent.
type AppEvents []AppEvent

// AppEvent contains the data for an in-app event type.
type AppEvent struct {
	Name    string
	Rate    time.Duration
	Handler Handler
}

// SubscribeAndListen subscribes to an AppEvent.
func (e *AppEvent) SubscribeAndListen(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SubscribeAndListen")
	defer span.Finish()

	ticker := time.NewTicker(e.Rate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			go func(t time.Time) {
				span, ctx := opentracing.StartSpanFromContext(ctx, e.Name)
				defer span.Finish()

				var errExpected internalerror.ErrExpected
				err := e.Handler.Handle(ctx, nil)
				if err != nil && !errors.As(err, &errExpected) {
					log.Printf("%v: %s", t, err.Error())
				}
			}(t)
		}
	}
}
