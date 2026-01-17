// Package config handles environment variables.
package config

import (
	"time"

	"fmt"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

// Config contains environment variables.
type Config struct {
	Port                       string        `envconfig:"PORT" default:"8000"`
	OtelEndpoint               string        `envconfig:"OTEL_ENDPOINT" required:"true"`
	OtelSamplerParam           float64       `envconfig:"OTEL_SAMPLER_PARAM" default:"1"`
	OtelExporterTimeout        time.Duration `envconfig:"OTEL_EXPORTER_TIMEOUT" default:"1s"` // GRPC timeout for span export requests
	OtelBatchInterval          time.Duration `envconfig:"OTEL_BATCH_INTERVAL" default:"5s"`   // Maximum interval for batching spans
	OtelBatchSize              int           `envconfig:"OTEL_BATCH_SIZE" default:"512"`      // Maximum number of spans in a batch
	ExampleAPIEndpoint         string        `envconfig:"EXAMPLE_API_ENDPOINT" required:"true"`
	ExampleAPIAccessEndpoint   string        `envconfig:"EXAMPLE_API_ACCESS_ENDPOINT" required:"true"`
	ExampleAPIClientID         string        `envconfig:"EXAMPLE_API_CLIENT_ID" required:"true"`
	ExampleAPIClientSecret     string        `envconfig:"EXAMPLE_API_CLIENT_SECRET" required:"true"`
	PubSubProjectName          string        `envconfig:"PUBSUB_PROJECT_NAME" required:"true"`
	CloudSQLInstance           string        `envconfig:"CLOUD_SQL_INSTANCE" required:"true"`
	DatabasePassword           string        `envconfig:"DATABASE_PASSWORD" required:"true"`
	DatabaseUser               string        `envconfig:"DATABASE_USER" required:"true"`
	DatabaseDB                 string        `envconfig:"DATABASE_DB" default:"postgres"`
	DatabaseOptions            string        `envconfig:"DATABASE_OPTIONS" default:"sslmode=disable"`
	DatabaseMaxConnections     int           `envconfig:"DATABASE_MAX_CONNECTIONS" default:"200"`
	DatabaseMaxIdleConnections int           `envconfig:"DATABASE_MAX_IDLE_CONNECTIONS" default:"50"`
	DatabaseMaxIdleTimeMinutes int           `envconfig:"DATABASE_MAX_IDLE_TIME_MINUTES" default:"5"`
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
