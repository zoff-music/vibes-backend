package handler

import (
	"encoding/json"
	"fmt"
	"log"
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
func GetStats(
	sf vibe.StatsFetcher,
	cache vibe.CachedStatsFetcherCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		cachedStats, err := cache.GetCachedStats(ctx)
		if err != nil {
			log.Printf("error getting cached stats: %v", err)
			cachedStats = &vibe.CachedStats{}
		}

		stats := &cachedStats.Stats
		if cachedStats.IsEmpty() {
			stats, err = sf.GetStats(ctx)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error fetching stats in GetStats: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			err = cache.CacheStats(ctx, *stats)
			if err != nil {
				log.Printf("error caching stats: %v", err)
			}
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
		_, _ = w.Write(body)
	}
}
