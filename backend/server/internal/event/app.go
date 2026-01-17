package event

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
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
	for t := range time.Tick(e.Rate) {
		go func(t time.Time) {
			span, ctx := opentracing.StartSpanFromContext(context.Background(), e.Name)
			defer span.Finish()

			var errExpected internalerror.ErrExpected
			err := e.Handler.Handle(ctx, nil)
			if err != nil && !errors.As(err, &errExpected) {
				log.Error(t, err.Error())
			}
		}(t)
	}
}
