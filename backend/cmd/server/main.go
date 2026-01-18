package main

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/zoff-music/vibes/config"
	"github.com/zoff-music/vibes/server"
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Enable debug logging if DEBUG=TRUE
	if os.Getenv("DEBUG") == "TRUE" || os.Getenv("DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	}

	log.Info("Starting ...")

	ctx := context.Background()
	config, err := config.LoadConfig()
	if err != nil {
		log.WithField("err", err.Error()).Fatal("Failed to load config")
	}

	var s server.Server

	if err := s.Create(ctx, config); err != nil {
		log.WithField("err", err.Error()).Fatal("Server error from s.Create()")
	}

	errc := make(chan error)
	go s.Serve(ctx, errc)

	err = <-errc
	if err != nil {
		log.WithField("err", err.Error()).Fatal("Server error from s.Serve()")
	}
}
