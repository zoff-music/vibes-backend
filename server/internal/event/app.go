package event

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/monitoring/metrics"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
)

// AppEvents contains a slice of AppEvent.
type AppEvents []AppEvent

// AppEvent contains the data for an in-app event type.
type AppEvent struct {
	Name     string
	Rate     time.Duration
	IdleRate time.Duration
	Handler  Handler
}

// SubscribeAndListen subscribes to an AppEvent.
func (e *AppEvent) SubscribeAndListen(ctx context.Context) {
	timer := time.NewTimer(e.Rate)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-timer.C:
			span, handlerCtx := tracing.StartSpanFromContext(ctx, e.Name)
			start := time.Now()
			status := http.StatusOK
			nextRate := e.Rate

			var errExpected internalerror.ErrExpected
			err := e.Handler.Handle(handlerCtx, nil)
			nextRate = e.nextRate(err)
			if err != nil && !errors.As(err, &errExpected) {
				status = http.StatusInternalServerError
				log.Printf("%v: %s", t, err.Error())
			}
			if err != nil && errors.As(err, &errExpected) {
				status = http.StatusAccepted
			}

			metrics.ObserveTaskDuration(e.Name, time.Since(start).Seconds())
			metrics.ProcessedTask(status, e.Name)
			span.End()
			timer.Reset(nextRate)
		}
	}
}

func (e *AppEvent) nextRate(err error) time.Duration {
	var errExpected internalerror.ErrExpected
	if e.IdleRate > 0 && errors.As(err, &errExpected) {
		return e.IdleRate
	}

	return e.Rate
}
