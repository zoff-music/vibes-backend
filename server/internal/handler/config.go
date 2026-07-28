package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zoff-music/vibes-backend/config"
)

// GetProviders handles GET /api/v1/providers
//
//	@Summary	List enabled providers
//	@Tags		config
//	@Produce	json
//	@Success	200	{array}		string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/providers [get]
func GetProviders(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providers := cfg.EnabledProviders()

		body, err := json.Marshal(providers)
		if err != nil {
			handleError(
				w,
				err,
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}
