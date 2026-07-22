// Package config handles environment variables.
package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config contains environment variables.
type Config struct {
	Port                string        `envconfig:"PORT" default:"8080"`
	InternalPort        string        `envconfig:"INTERNAL_PORT" default:"8081"`
	OtelEndpoint        string        `envconfig:"OTEL_ENDPOINT" default:"alloy.monitoring.svc.cluster.local:4317"`
	OtelServiceName     string        `envconfig:"OTEL_SERVICE_NAME" default:"vibes-backend"`
	OtelResourceAttrs   string        `envconfig:"OTEL_RESOURCE_ATTRIBUTES" default:""`
	OtelSamplerParam    float64       `envconfig:"OTEL_SAMPLER_PARAM" default:"1"`
	OtelExporterTimeout time.Duration `envconfig:"OTEL_EXPORTER_TIMEOUT" default:"1s"`
	OtelBatchInterval   time.Duration `envconfig:"OTEL_BATCH_INTERVAL" default:"5s"`
	OtelBatchSize       int           `envconfig:"OTEL_BATCH_SIZE" default:"512"`

	DatabaseURL          string `envconfig:"DATABASE_URL" required:"true"`
	DatabaseMaxConns     int    `envconfig:"DATABASE_MAX_CONNECTIONS" default:"10"`
	DatabaseMaxIdleConns int    `envconfig:"DATABASE_MAX_IDLE_CONNECTIONS" default:"2"`
	RedisURL             string `envconfig:"REDIS_URL" default:""`
	RateLimitEnabled     bool   `envconfig:"RATE_LIMIT_ENABLED" default:"false"`
	MaxNameLength        int    `envconfig:"MAX_NAME_LENGTH" default:"100"`
	MaxQueueLength       int    `envconfig:"MAX_QUEUE_LENGTH" default:"200"`
	YouTubeAPIKey        string `envconfig:"YOUTUBE_API_KEY" default:""`
	YouTubeEndpoint      string `envconfig:"YOUTUBE_ENDPOINT" default:"https://www.googleapis.com/youtube/v3"`
	YouTubeClientID      string `envconfig:"YOUTUBE_CLIENT_ID" default:""`
	YouTubeClientSecret  string `envconfig:"YOUTUBE_CLIENT_SECRET" default:""`
	YouTubeRedirectURI   string `envconfig:"YOUTUBE_REDIRECT_URI" default:"https://localhost/api/v1/callbacks/youtube"`

	// SoundCloud configuration
	SoundCloudEndpoint     string `envconfig:"SOUNDCLOUD_ENDPOINT" default:"https://api.soundcloud.com"`
	SoundCloudClientID     string `envconfig:"SOUNDCLOUD_CLIENT_ID" default:""`
	SoundCloudClientSecret string `envconfig:"SOUNDCLOUD_CLIENT_SECRET" default:""`
	SoundCloudRedirectURI  string `envconfig:"SOUNDCLOUD_REDIRECT_URI" default:"https://localhost/api/v1/callbacks/soundcloud"`

	// Spotify configuration
	SpotifyClientID     string `envconfig:"SPOTIFY_CLIENT_ID" default:""`
	SpotifyClientSecret string `envconfig:"SPOTIFY_CLIENT_SECRET" default:""`
	SpotifyEndpoint     string `envconfig:"SPOTIFY_ENDPOINT" default:"https://api.spotify.com/v1"`
	SpotifyTokenURL     string `envconfig:"SPOTIFY_TOKEN_URL" default:"https://accounts.spotify.com/api/token"`
	SpotifyRedirectURI  string `envconfig:"SPOTIFY_REDIRECT_URI" default:"https://127.0.0.1/api/v1/callbacks/spotify"`

	// User session settings
	UserInactivityTimeout time.Duration `envconfig:"USER_INACTIVITY_TIMEOUT" default:"30m"`
	CookieSecret          string        `envconfig:"COOKIE_SECRET" default:"vibes-default-secret-change-me"`
	AdminPassword         string        `envconfig:"ADMIN_PASSWORD" default:""`
	EmbedBasePath         string        `envconfig:"EMBED_BASE_PATH" default:"/embed"`

	// Cast auth
	CastTokenSecret string `envconfig:"CAST_TOKEN_SECRET" default:""`

	// CORS
	CORSAllowedOrigins string `envconfig:"CORS_ALLOWED_ORIGINS" default:""`
}

// LoadConfig reads environment variables and populates Config.
func LoadConfig() (*Config, error) {
	// Try loading from current directory and parent directory (monorepo root)
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	var c Config

	err := envconfig.Process("", &c)
	if err != nil {
		return nil, fmt.Errorf("error processing env: %w", err)
	}

	return &c, nil
}
