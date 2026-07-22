package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zoff-music/vibes-backend/config"
	"github.com/zoff-music/vibes-backend/server"
)

// @title			Vibes API
// @version		1.0
// @description	Backend API for Vibes rooms, queues, playback, providers, casting, and admin tools.
// @BasePath		/
func main() {
	err := run()
	if err != nil {
		log.Printf("Server stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Enable debug logging if DEBUG=TRUE
	if os.Getenv("DEBUG") == "TRUE" || os.Getenv("DEBUG") == "true" {
		// Standard log doesn't have levels, just print that debug is enabled
		log.Println("Debug logging enabled")
	}

	log.Println("Starting ...")

	ctx := context.Background()
	serverConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	var s server.Server

	err = s.Create(ctx, serverConfig)
	if err != nil {
		return fmt.Errorf("error creating server: %w", err)
	}

	errc := make(chan error)
	go s.Serve(ctx, errc)

	err = <-errc
	if err != nil {
		return fmt.Errorf("error serving: %w", err)
	}

	return nil
}
