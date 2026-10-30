package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

func CreateActivities(analytics vibe.AnalyticsEventCreator, domain string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Header.Get("DNT") == "1" || r.Header.Get("Sec-GPC") == "1" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, analyticsMaxBodyBytes))
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error reading analytics event: %w", err),
				http.StatusRequestEntityTooLarge,
				false,
			)
			return
		}
		var event vibe.AnalyticsEvent
		err = json.Unmarshal(body, &event)
		if err == nil {
			err = event.Validate(domain)
		}
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating analytics event: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}
		visitor := vibe.AnalyticsVisitor{IP: helper.ClientIP(r), UserAgent: r.UserAgent()}
		result, err := analytics.CreateAnalyticsEvent(r.Context(), event, visitor)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error creating analytics event: %w", err),
				http.StatusBadGateway,
				true,
			)
			return
		}
		if result.Dropped {
			w.Header().Set("X-Plausible-Dropped", "1")
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

const analyticsMaxBodyBytes = 16 * 1024
