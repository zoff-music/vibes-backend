package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zoff-music/vibes/config"
)

// GetProviders handles GET /api/v1/providers
func GetProviders(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providers := []string{}

		if cfg.SpotifyClientID != "" {
			providers = append(providers, "spotify")
		}
		if cfg.YouTubeAPIKey != "" || cfg.YouTubeClientID != "" {
			providers = append(providers, "youtube")
		}
		if cfg.SoundCloudClientID != "" || cfg.SoundCloudAPIKey != "" {
			providers = append(providers, "soundcloud")
		}

		body, err := json.Marshal(providers)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}
}
