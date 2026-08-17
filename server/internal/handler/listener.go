package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/zoff-music/vibes-backend/vibe"
)

// AdminListenerUsage handles GET /api/v1/admin/listeners/usage
//
//	@Summary	List listener usage points
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	vibe.AdminListenerUsage
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/admin/listeners/usage [get]
func AdminListenerUsage(
	db vibe.AdminListenerUsageLister,
	cache vibe.CachedAdminListenerUsageFetcherCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		cachedUsage, err := cache.GetCachedAdminListenerUsage(ctx)
		if err != nil {
			log.Printf("error getting cached admin listener usage: %v", err)
			cachedUsage = &vibe.CachedAdminListenerUsage{}
		}

		usage := &cachedUsage.Usage
		if cachedUsage.IsEmpty() {
			points, err := db.ListAdminListenerUsage(ctx)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error fetching admin listener usage: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			usage = &vibe.AdminListenerUsage{
				Points:      points,
				GeneratedAt: time.Now().UTC(),
			}
			err = cache.CacheAdminListenerUsage(ctx, *usage)
			if err != nil {
				log.Printf("error caching admin listener usage: %v", err)
			}
		}

		body, err := json.Marshal(usage)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin listener usage: %w", err),
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
