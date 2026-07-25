package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zoff-music/vibes-backend/vibe"
)

// GetStats handles GET /api/v1/stats.
//
//	@Summary	Get public service statistics
//	@Tags		stats
//	@Produce	json
//	@Success	200	{object}	vibe.Stats
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/stats [get]
func GetStats(sf vibe.StatsFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := sf.GetStats(r.Context())
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching stats in GetStats: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(stats)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling stats in GetStats: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}
}
