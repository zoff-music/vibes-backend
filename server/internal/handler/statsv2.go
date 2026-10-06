package handler

import (
	"encoding/json/v2"
	"fmt"
	"log"
	"net/http"

	"github.com/zoff-music/vibes-backend/vibe"
)

// GetStatsV2 handles GET /api/v2/stats.
//
//	@Summary	Get public service statistics
//	@Tags		stats
//	@Produce	json
//	@Param		roomType	query	string	false	"Restrict all counts to a room type; omitted returns all types"	Enums(MUSIC, WATCH)
//	@Success	200	{object}	vibe.StatsV2
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/stats [get]
func GetStatsV2(
	sf vibe.StatsV2Fetcher,
	cache vibe.CachedStatsV2FetcherCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		roomType := vibe.RoomType(r.URL.Query().Get("roomType"))
		if roomType != "" && !roomType.IsValid() {
			handleError(w, fmt.Errorf("error fetching stats: roomType must be MUSIC or WATCH"), http.StatusBadRequest, false)
			return
		}

		cachedStats, err := cache.GetCachedStatsV2(ctx, roomType)
		if err != nil {
			log.Printf("error getting cached stats: %v", err)
			cachedStats = &vibe.CachedStatsV2{}
		}

		stats := &cachedStats.Stats
		if cachedStats.IsEmpty() {
			stats, err = sf.GetStatsV2(ctx, roomType)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error fetching stats in GetStatsV2: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			err = cache.CacheStatsV2(ctx, roomType, *stats)
			if err != nil {
				log.Printf("error caching stats: %v", err)
			}
		}

		body, err := json.Marshal(stats)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling stats in GetStatsV2: %w", err),
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
