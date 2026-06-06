package main

import (
	"context"
	"log"
	"os"

	"github.com/zoff-music/vibes/config"
	"github.com/zoff-music/vibes/server"
)

func main() {
	// Enable debug logging if DEBUG=TRUE
	if os.Getenv("DEBUG") == "TRUE" || os.Getenv("DEBUG") == "true" {
		// Standard log doesn't have levels, just print that debug is enabled
		log.Println("Debug logging enabled")
	}

	log.Println("Starting ...")

	ctx := context.Background()
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var s server.Server

	if err := s.Create(ctx, config); err != nil {
		log.Fatalf("Server error from s.Create(): %v", err)
	}

	errc := make(chan error)
	go s.Serve(ctx, errc)

	err = <-errc
	if err != nil {
		log.Fatalf("Server error from s.Serve(): %v", err)
	}
}
