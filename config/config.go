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

	DatabaseURL              string        `envconfig:"DATABASE_URL" required:"true"`
	DatabaseMaxConns         int           `envconfig:"DATABASE_MAX_CONNECTIONS" default:"10"`
	DatabaseMaxIdleConns     int           `envconfig:"DATABASE_MAX_IDLE_CONNECTIONS" default:"2"`
	RedisURL                 string        `envconfig:"REDIS_URL" required:"true"`
	RateLimitEnabled         bool          `envconfig:"RATE_LIMIT_ENABLED" default:"false"`
	RoomEventReplayMaxEvents int           `envconfig:"ROOM_EVENT_REPLAY_MAX_EVENTS" default:"1000"`
	RoomEventReplayMaxAge    time.Duration `envconfig:"ROOM_EVENT_REPLAY_MAX_AGE" default:"2h"`
	MaxNameLength            int           `envconfig:"MAX_NAME_LENGTH" default:"100"`
	MaxQueueLength           int           `envconfig:"MAX_QUEUE_LENGTH" default:"200"`
	RoomNameReservationTTL   time.Duration `envconfig:"ROOM_NAME_RESERVATION_TTL" default:"2m"`
	YouTubeAPIKey            string        `envconfig:"YOUTUBE_API_KEY" default:""`
	YouTubeEndpoint          string        `envconfig:"YOUTUBE_ENDPOINT" default:"https://www.googleapis.com/youtube/v3"`
	YouTubeClientID          string        `envconfig:"YOUTUBE_CLIENT_ID" default:""`
	YouTubeClientSecret      string        `envconfig:"YOUTUBE_CLIENT_SECRET" default:""`
	YouTubeRedirectURI       string        `envconfig:"YOUTUBE_REDIRECT_URI" default:"https://localhost/api/v1/callbacks/youtube"`

	// SoundCloud configuration
	SoundCloudEndpoint     string `envconfig:"SOUNDCLOUD_ENDPOINT" default:"https://api.soundcloud.com"`
	SoundCloudClientID     string `envconfig:"SOUNDCLOUD_CLIENT_ID" default:""`
	SoundCloudClientSecret string `envconfig:"SOUNDCLOUD_CLIENT_SECRET" default:""`
	SoundCloudRedirectURI  string `envconfig:"SOUNDCLOUD_REDIRECT_URI" default:"https://localhost/api/v1/callbacks/soundcloud"`

	// AI configuration
	AIModel                             string `envconfig:"AI_MODEL" default:"GROK:grok-4.3"`
	GeneratedPlaylistTrackCount         int    `envconfig:"GENERATED_PLAYLIST_TRACK_COUNT" default:"80"`
	GeneratedPlaylistSelectedTrackCount int    `envconfig:"GENERATED_PLAYLIST_SELECTED_TRACK_COUNT" default:"30"`
	RoomGenerationMaxAttempts           int    `envconfig:"ROOM_GENERATION_MAX_ATTEMPTS" default:"5"`
	RoomGenerationMaxDailyCount         int    `envconfig:"ROOM_GENERATION_MAX_DAILY_COUNT" default:"2"`
	RoomGenerationMaxExistingSongs      int    `envconfig:"ROOM_GENERATION_MAX_EXISTING_SONGS" default:"59"`

	// Grok configuration
	GrokAPIKey   string `envconfig:"GROK_API_KEY" default:""`
	GrokEndpoint string `envconfig:"GROK_ENDPOINT" default:"https://api.x.ai/v1"`

	// Gemini configuration
	GeminiAPIKey   string `envconfig:"GEMINI_API_KEY" default:""`
	GeminiEndpoint string `envconfig:"GEMINI_ENDPOINT" default:"https://generativelanguage.googleapis.com/v1beta/openai"`

	// User session settings
	UserInactivityTimeout time.Duration `envconfig:"USER_INACTIVITY_TIMEOUT" default:"30m"`
	SessionCookieMaxAge   time.Duration `envconfig:"SESSION_COOKIE_MAX_AGE" default:"87600h"`
	CookieSecret          string        `envconfig:"COOKIE_SECRET" default:"vibes-default-secret-change-me"`
	AdminPasswordPepper   string        `envconfig:"ADMIN_PASSWORD_PEPPER" default:""`
	EmbedBasePath         string        `envconfig:"EMBED_BASE_PATH" default:"/embed"`
	RemotePairingTTL      time.Duration `envconfig:"REMOTE_PAIRING_TTL" default:"5m"`

	// Cast auth
	CastTokenSecret string `envconfig:"CAST_TOKEN_SECRET" default:""`

	// CORS
	CORSAllowedOrigins string `envconfig:"CORS_ALLOWED_ORIGINS" default:""`
}

// EnabledProviders returns the music providers configured for use.
func (c *Config) EnabledProviders() []string {
	providers := []string{}

	if c.YouTubeAPIKey != "" {
		providers = append(providers, "youtube")
	}
	if c.SoundCloudClientID != "" {
		providers = append(providers, "soundcloud")
	}

	return providers
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
	if c.GeneratedPlaylistTrackCount < 1 {
		return nil, fmt.Errorf("error validating generated playlist track count: must be greater than zero")
	}
	if c.GeneratedPlaylistSelectedTrackCount < 1 {
		return nil, fmt.Errorf("error validating generated playlist selected track count: must be greater than zero")
	}
	if c.GeneratedPlaylistSelectedTrackCount > c.GeneratedPlaylistTrackCount {
		return nil, fmt.Errorf(
			"error validating generated playlist selected track count: must not exceed generated playlist track count",
		)
	}
	if c.RoomGenerationMaxAttempts < 1 {
		return nil, fmt.Errorf("error validating room generation max attempts: must be greater than zero")
	}
	if c.RoomGenerationMaxDailyCount < 1 {
		return nil, fmt.Errorf("error validating room generation max daily count: must be greater than zero")
	}
	if c.RoomGenerationMaxExistingSongs < 0 {
		return nil, fmt.Errorf("error validating room generation max existing songs: must not be negative")
	}
	if c.RemotePairingTTL <= 0 {
		return nil, fmt.Errorf("error validating remote pairing ttl: must be greater than zero")
	}
	if c.SessionCookieMaxAge <= 0 {
		return nil, fmt.Errorf("error validating session cookie max age: must be greater than zero")
	}
	if c.RoomEventReplayMaxEvents < 1 {
		return nil, fmt.Errorf("error validating room event replay max events: must be greater than zero")
	}
	if c.RoomEventReplayMaxAge <= 0 {
		return nil, fmt.Errorf("error validating room event replay max age: must be greater than zero")
	}

	return &c, nil
}
