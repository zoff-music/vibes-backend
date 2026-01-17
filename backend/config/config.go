// Package config handles environment variables.
package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

// Config contains environment variables.
type Config struct {
	Port                string        `envconfig:"PORT" default:"8080"`
	OtelEndpoint        string        `envconfig:"OTEL_ENDPOINT" default:""`
	OtelSamplerParam    float64       `envconfig:"OTEL_SAMPLER_PARAM" default:"1"`
	OtelExporterTimeout time.Duration `envconfig:"OTEL_EXPORTER_TIMEOUT" default:"1s"`
	OtelBatchInterval   time.Duration `envconfig:"OTEL_BATCH_INTERVAL" default:"5s"`
	OtelBatchSize       int           `envconfig:"OTEL_BATCH_SIZE" default:"512"`

	// SQLite configuration
	DatabasePath   string `envconfig:"DATABASE_PATH" default:"./data/vibes.db"`
	MaxNameLength  int    `envconfig:"MAX_NAME_LENGTH" default:"100"`
	MaxQueueLength int    `envconfig:"MAX_QUEUE_LENGTH" default:"200"`

	// User session settings
	UserInactivityTimeout time.Duration `envconfig:"USER_INACTIVITY_TIMEOUT" default:"30m"`
}

// LoadConfig reads environment variables and populates Config.
func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Info("No .env file found")
	}

	var c Config

	err := envconfig.Process("", &c)
	if err != nil {
		return nil, fmt.Errorf("error processing env: %w", err)
	}

	return &c, nil
}
